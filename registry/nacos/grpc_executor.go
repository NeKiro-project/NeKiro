package nacos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/NeKiro-project/NeKiro/registry"
	nacosgrpc "github.com/nacos-group/nacos-sdk-go/v2/api/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
)

type GRPCExecutorConfig struct {
	Target               string
	ClientIP             string
	RequestTimeout       time.Duration
	TransportCredentials credentials.TransportCredentials
	RequestHeaders       map[string]string
}

type GRPCExecutor struct {
	target         string
	clientIP       string
	requestTimeout time.Duration
	credentials    credentials.TransportCredentials
	headers        map[string]string
}

func NewGRPCExecutor(config GRPCExecutorConfig) (*GRPCExecutor, error) {
	host, port, err := net.SplitHostPort(config.Target)
	portNumber, portErr := strconv.Atoi(port)
	parsedIP := net.ParseIP(config.ClientIP)
	if err != nil || portErr != nil || host == "" || portNumber < 1 || portNumber > 65535 || strings.Contains(config.Target, "://") ||
		parsedIP == nil || parsedIP.String() != config.ClientIP ||
		config.RequestTimeout < 100*time.Millisecond || config.RequestTimeout > 30*time.Second ||
		isNilInterface(config.TransportCredentials) {
		return nil, registry.ErrInvalid
	}
	headers := make(map[string]string, len(config.RequestHeaders))
	for key, value := range config.RequestHeaders {
		if !validText(key) || !validText(value) {
			return nil, registry.ErrInvalid
		}
		headers[key] = value
	}
	return &GRPCExecutor{target: config.Target, clientIP: config.ClientIP, requestTimeout: config.RequestTimeout, credentials: config.TransportCredentials, headers: headers}, nil
}

func (*GRPCExecutor) Guarantees() NamingSubscriptionGuarantees {
	return NamingSubscriptionGuarantees{
		Version: NamingSubscriptionExecutorVersionV1, AtomicInitialPushHandoff: true,
		ExactlyOneSubscribeAttempt: true, NoRetry: true, NoReconnect: true,
		NoResponseCache: true, NoFailover: true, NoImplicitPolling: true,
		NoHiddenReauthentication: true,
	}
}

func (executor *GRPCExecutor) Subscribe(ctx context.Context, request NamingSubscribeRequest) (NamingSubscription, error) {
	if ctx == nil || !validText(request.namespaceID) || !validText(request.serviceName) || !validText(request.groupName) || !validText(request.clusterName) {
		return NamingSubscription{}, registry.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return NamingSubscription{}, canceled(err)
	}
	var dialed atomic.Bool
	dialer := func(ctx context.Context, _ string) (net.Conn, error) {
		if !dialed.CompareAndSwap(false, true) {
			return nil, errors.New("nacos grpc connection attempt rejected")
		}
		return (&net.Dialer{}).DialContext(ctx, "tcp", executor.target)
	}
	connection, err := grpc.NewClient("passthrough:///"+executor.target,
		grpc.WithTransportCredentials(executor.credentials), grpc.WithContextDialer(dialer), grpc.WithDisableRetry())
	if err != nil {
		return NamingSubscription{}, registry.ErrUnavailable
	}
	fail := func(stage string) (NamingSubscription, error) {
		_ = connection.Close()
		return NamingSubscription{}, fmt.Errorf("nacos grpc %s: %w", stage, registry.ErrUnavailable)
	}
	client := nacosgrpc.NewRequestClient(connection)
	checkPayload, err := executor.unary(ctx, client, "ServerCheckRequest", map[string]any{"module": "internal", "requestId": requestID()})
	if err != nil || checkPayload.GetMetadata().GetType() != "ServerCheckResponse" {
		return fail("server_check_request_" + status.Code(err).String())
	}
	var check struct {
		ResultCode   int    `json:"resultCode"`
		ErrorCode    int    `json:"errorCode"`
		ConnectionID string `json:"connectionId"`
	}
	if json.Unmarshal(checkPayload.GetBody().GetValue(), &check) != nil || check.ResultCode != 200 || check.ErrorCode != 0 || check.ConnectionID == "" {
		return fail("server_check_response")
	}
	streamContext, cancel := context.WithCancel(context.Background())
	stream, err := nacosgrpc.NewBiRequestStreamClient(connection).RequestBiStream(streamContext)
	if err != nil {
		cancel()
		return fail("open_push_stream")
	}
	setup := map[string]any{"module": "internal", "requestId": requestID(), "clientVersion": "NeKiro-Core", "tenant": request.namespaceID, "labels": map[string]string{"source": "sdk", "module": "naming"}, "clientAbilities": map[string]any{}}
	if err := stream.Send(executor.payload("ConnectionSetupRequest", setup, nil)); err != nil {
		cancel()
		return fail("connection_setup")
	}
	// Nacos does not acknowledge ConnectionSetupRequest. Its published clients
	// wait briefly before the first unary request so the stream is associated.
	setupTimer := time.NewTimer(100 * time.Millisecond)
	select {
	case <-ctx.Done():
		setupTimer.Stop()
		cancel()
		return fail("connection_setup_canceled")
	case <-setupTimer.C:
	}
	body := map[string]any{"namespace": request.namespaceID, "serviceName": request.serviceName, "groupName": request.groupName, "module": "naming", "requestId": requestID(), "subscribe": true, "clusters": request.clusterName}
	payload, err := executor.unary(ctx, client, "SubscribeServiceRequest", body)
	if err != nil || payload.GetMetadata().GetType() != "SubscribeServiceResponse" {
		cancel()
		return fail("subscribe_request_" + status.Code(err).String())
	}
	var response struct {
		ResultCode  int             `json:"resultCode"`
		ErrorCode   int             `json:"errorCode"`
		Success     bool            `json:"success"`
		ServiceInfo json.RawMessage `json:"serviceInfo"`
	}
	if json.Unmarshal(payload.GetBody().GetValue(), &response) != nil || !response.Success || response.ResultCode != 200 || response.ErrorCode != 0 || len(response.ServiceInfo) == 0 {
		cancel()
		return fail("subscribe_response")
	}
	push := &grpcPushStream{executor: executor, connection: connection, stream: stream, cancel: cancel, request: request}
	return NewNamingSubscription(response.ServiceInfo, push)
}

func (executor *GRPCExecutor) unary(ctx context.Context, client nacosgrpc.RequestClient, requestType string, body any) (*nacosgrpc.Payload, error) {
	requestContext, cancel := context.WithTimeout(ctx, executor.requestTimeout)
	defer cancel()
	return client.Request(requestContext, executor.payload(requestType, body, executor.headers))
}

func (executor *GRPCExecutor) payload(requestType string, body any, headers map[string]string) *nacosgrpc.Payload {
	data, _ := json.Marshal(body)
	return &nacosgrpc.Payload{Metadata: &nacosgrpc.Metadata{Type: requestType, ClientIp: executor.clientIP, Headers: cloneStringMap(headers)}, Body: &anypb.Any{Value: data}}
}

type grpcPushStream struct {
	executor   *GRPCExecutor
	connection *grpc.ClientConn
	stream     nacosgrpc.BiRequestStream_RequestBiStreamClient
	cancel     context.CancelFunc
	request    NamingSubscribeRequest
	closeOnce  sync.Once
	closeErr   error
}

func (stream *grpcPushStream) Next(ctx context.Context) ([]byte, error) {
	if ctx == nil {
		return nil, registry.ErrInvalid
	}
	for {
		payload, err := stream.stream.Recv()
		if err != nil {
			return nil, err
		}
		switch payload.GetMetadata().GetType() {
		case "ClientDetectionRequest":
			if err := stream.ack(payload, "ClientDetectionResponse", false); err != nil {
				return nil, err
			}
			continue
		case "NotifySubscriberRequest":
			var push struct {
				RequestID   string          `json:"requestId"`
				Namespace   string          `json:"namespace"`
				ServiceName string          `json:"serviceName"`
				GroupName   string          `json:"groupName"`
				ServiceInfo json.RawMessage `json:"serviceInfo"`
			}
			if json.Unmarshal(payload.GetBody().GetValue(), &push) != nil || push.RequestID == "" || push.Namespace != stream.request.namespaceID || push.ServiceName != stream.request.serviceName || push.GroupName != stream.request.groupName || len(push.ServiceInfo) == 0 {
				return nil, registry.ErrInvalid
			}
			if err := stream.ack(payload, "NotifySubscriberResponse", true); err != nil {
				return nil, err
			}
			return append([]byte(nil), push.ServiceInfo...), nil
		default:
			return nil, registry.ErrInvalid
		}
	}
}

func (stream *grpcPushStream) ack(request *nacosgrpc.Payload, responseType string, success bool) error {
	var requestBody struct {
		RequestID string `json:"requestId"`
	}
	if json.Unmarshal(request.GetBody().GetValue(), &requestBody) != nil || requestBody.RequestID == "" {
		return registry.ErrInvalid
	}
	return stream.stream.Send(stream.executor.payload(responseType, map[string]any{"resultCode": 200, "errorCode": 0, "success": success, "requestId": requestBody.RequestID}, nil))
}

func (stream *grpcPushStream) Close() error {
	stream.closeOnce.Do(func() {
		stream.cancel()
		stream.closeErr = stream.connection.Close()
	})
	return stream.closeErr
}

var requestSequence atomic.Uint64

func requestID() string { return fmt.Sprintf("nekiro-%d", requestSequence.Add(1)) }

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

var _ NamingSubscriptionExecutor = (*GRPCExecutor)(nil)
var _ NamingPushStream = (*grpcPushStream)(nil)
