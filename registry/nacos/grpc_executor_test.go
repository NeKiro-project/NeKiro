package nacos

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/NeKiro-project/NeKiro/registry"
	nacosgrpc "github.com/nacos-group/nacos-sdk-go/v2/api/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestGRPCExecutorValidatesConfigurationAndCopiesHeaders(t *testing.T) {
	base := GRPCExecutorConfig{
		Target: "127.0.0.1:9848", ClientIP: "127.0.0.1", RequestTimeout: time.Second,
		TransportCredentials: insecure.NewCredentials(), RequestHeaders: map[string]string{"accessToken": "token-a"},
	}
	for name, mutate := range map[string]func(*GRPCExecutorConfig){
		"target scheme": func(config *GRPCExecutorConfig) { config.Target = "grpc://127.0.0.1:9848" },
		"target port":   func(config *GRPCExecutorConfig) { config.Target = "127.0.0.1:0" },
		"client IP":     func(config *GRPCExecutorConfig) { config.ClientIP = "localhost" },
		"short timeout": func(config *GRPCExecutorConfig) { config.RequestTimeout = 99 * time.Millisecond },
		"long timeout":  func(config *GRPCExecutorConfig) { config.RequestTimeout = 31 * time.Second },
		"credentials":   func(config *GRPCExecutorConfig) { config.TransportCredentials = nil },
		"header name":   func(config *GRPCExecutorConfig) { config.RequestHeaders = map[string]string{" bad": "token-a"} },
		"header value":  func(config *GRPCExecutorConfig) { config.RequestHeaders = map[string]string{"accessToken": " token-a"} },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := base
			mutate(&candidate)
			if _, err := NewGRPCExecutor(candidate); !errors.Is(err, registry.ErrInvalid) {
				t.Fatalf("NewGRPCExecutor error=%v", err)
			}
		})
	}
	executor, err := NewGRPCExecutor(base)
	if err != nil {
		t.Fatal(err)
	}
	base.RequestHeaders["accessToken"] = "changed"
	if executor.headers["accessToken"] != "token-a" {
		t.Fatalf("executor headers=%v", executor.headers)
	}
	if err := executor.Guarantees().Validate(); err != nil {
		t.Fatalf("Guarantees error=%v", err)
	}
}

func TestGRPCExecutorRejectsInvalidSubscriptionInputs(t *testing.T) {
	executor, err := NewGRPCExecutor(GRPCExecutorConfig{
		Target: "127.0.0.1:9848", ClientIP: "127.0.0.1", RequestTimeout: time.Second,
		TransportCredentials: insecure.NewCredentials(),
	})
	if err != nil {
		t.Fatal(err)
	}
	request := NamingSubscribeRequest{namespaceID: "public", serviceName: "runtime-b", groupName: "NEKIRO", clusterName: "DEFAULT"}
	if _, err := executor.Subscribe(nil, request); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("nil context error=%v", err)
	}
	if _, err := executor.Subscribe(t.Context(), NamingSubscribeRequest{}); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("invalid request error=%v", err)
	}
	canceledContext, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := executor.Subscribe(canceledContext, request); !errors.Is(err, registry.ErrCanceled) || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled context error=%v", err)
	}
}

func TestGRPCExecutorPreservesCancellationDuringSetup(t *testing.T) {
	provider := newFakeNamingGRPC()
	executor := openTestGRPCExecutor(t, provider)
	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		<-provider.setup
		cancel()
	}()
	request := NamingSubscribeRequest{namespaceID: "public", serviceName: "runtime-b", groupName: "NEKIRO", clusterName: "DEFAULT"}
	if _, err := executor.Subscribe(ctx, request); !errors.Is(err, registry.ErrCanceled) || !errors.Is(err, context.Canceled) {
		t.Fatalf("Subscribe error=%v", err)
	}
}

func TestGRPCExecutorNextHonorsCancellationWithoutClosingSubscription(t *testing.T) {
	executor, provider := openTestSubscription(t)
	defer executor.Close()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := executor.Next(ctx); !errors.Is(err, registry.ErrCanceled) || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled Next error=%v", err)
	}

	waiting, stopWaiting := context.WithTimeout(t.Context(), 25*time.Millisecond)
	defer stopWaiting()
	if _, err := executor.Next(waiting); !errors.Is(err, registry.ErrCanceled) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timed-out Next error=%v", err)
	}

	provider.push <- []byte(changedServiceInfo)
	nextContext, cancelNext := context.WithTimeout(t.Context(), time.Second)
	defer cancelNext()
	payload, err := executor.Next(nextContext)
	if err != nil || string(payload) != changedServiceInfo {
		t.Fatalf("Next after cancellation payload=%s error=%v", payload, err)
	}
}

func TestGRPCExecutorCloseWakesBlockedNext(t *testing.T) {
	subscription, _ := openTestSubscription(t)
	result := make(chan error, 1)
	go func() {
		_, err := subscription.Next(context.Background())
		result <- err
	}()
	if err := subscription.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, registry.ErrClosed) {
			t.Fatalf("blocked Next error=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Close did not wake blocked Next")
	}
}

func TestGRPCExecutorRejectsInvalidHandshakeResponses(t *testing.T) {
	request := NamingSubscribeRequest{namespaceID: "public", serviceName: "runtime-b", groupName: "NEKIRO", clusterName: "DEFAULT"}
	for name, mutate := range map[string]func(*fakeNamingGRPC){
		"server check type": func(provider *fakeNamingGRPC) { provider.serverCheckType = "UnexpectedResponse" },
		"server check body": func(provider *fakeNamingGRPC) {
			provider.serverCheckBody = map[string]any{"resultCode": 500, "errorCode": 0, "connectionId": "connection-1"}
		},
		"subscribe type": func(provider *fakeNamingGRPC) { provider.subscribeType = "UnexpectedResponse" },
		"subscribe body": func(provider *fakeNamingGRPC) {
			provider.subscribeBody = map[string]any{"resultCode": 200, "errorCode": 0, "success": true}
		},
	} {
		t.Run(name, func(t *testing.T) {
			provider := newFakeNamingGRPC()
			mutate(provider)
			executor := openTestGRPCExecutor(t, provider)
			if _, err := executor.Subscribe(t.Context(), request); !errors.Is(err, registry.ErrUnavailable) {
				t.Fatalf("Subscribe error=%v", err)
			}
		})
	}
}

func openTestGRPCExecutor(t *testing.T, provider *fakeNamingGRPC) *GRPCExecutor {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	nacosgrpc.RegisterRequestServer(server, provider)
	nacosgrpc.RegisterBiRequestStreamServer(server, provider)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { server.Stop(); _ = listener.Close() })
	executor, err := NewGRPCExecutor(GRPCExecutorConfig{
		Target: listener.Addr().String(), ClientIP: "127.0.0.1", RequestTimeout: time.Second,
		TransportCredentials: insecure.NewCredentials(), RequestHeaders: map[string]string{"accessToken": "token-a"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return executor
}

func openTestSubscription(t *testing.T) (NamingSubscription, *fakeNamingGRPC) {
	t.Helper()
	provider := newFakeNamingGRPC()
	executor := openTestGRPCExecutor(t, provider)
	subscription, err := executor.Subscribe(t.Context(), NamingSubscribeRequest{
		namespaceID: "public", serviceName: "runtime-b", groupName: "NEKIRO", clusterName: "DEFAULT",
	})
	if err != nil {
		t.Fatal(err)
	}
	return subscription, provider
}

func TestGRPCExecutorSubscribesAndAcknowledgesPush(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	provider := newFakeNamingGRPC()
	server := grpc.NewServer()
	nacosgrpc.RegisterRequestServer(server, provider)
	nacosgrpc.RegisterBiRequestStreamServer(server, provider)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { server.Stop(); _ = listener.Close() })

	executor, err := NewGRPCExecutor(GRPCExecutorConfig{
		Target: listener.Addr().String(), ClientIP: "127.0.0.1", RequestTimeout: time.Second,
		TransportCredentials: insecure.NewCredentials(), RequestHeaders: map[string]string{"accessToken": "token-a"},
	})
	if err != nil {
		t.Fatal(err)
	}
	subscription, err := executor.Subscribe(t.Context(), NamingSubscribeRequest{namespaceID: "public", serviceName: "runtime-b", groupName: "NEKIRO", clusterName: "DEFAULT"})
	if err != nil || string(subscription.InitialPayload()) != initialServiceInfo {
		t.Fatalf("Subscribe initial=%s error=%v", subscription.InitialPayload(), err)
	}
	provider.push <- []byte(changedServiceInfo)
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	payload, err := subscription.Next(ctx)
	var pushed listResponse
	if err != nil || json.Unmarshal(payload, &pushed) != nil || len(pushed.Hosts) != 2 {
		t.Fatalf("Next payload=%s error=%v", payload, err)
	}
	select {
	case responseType := <-provider.ack:
		if responseType != "NotifySubscriberResponse" {
			t.Fatalf("ack type=%s", responseType)
		}
	case <-ctx.Done():
		t.Fatal("push was not acknowledged")
	}
	if err := subscription.Close(); err != nil {
		t.Fatal(err)
	}
	if err := subscription.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestGRPCExecutorAgainstNacos(t *testing.T) {
	target := os.Getenv("NEKIRO_TEST_NACOS_GRPC_TARGET")
	clientIP := os.Getenv("NEKIRO_TEST_NACOS_GRPC_CLIENT_IP")
	if target == "" || clientIP == "" {
		t.Skip("Nacos gRPC integration environment is not configured")
	}
	executor, err := NewGRPCExecutor(GRPCExecutorConfig{
		Target: target, ClientIP: clientIP, RequestTimeout: 5 * time.Second,
		TransportCredentials: insecure.NewCredentials(),
	})
	if err != nil {
		t.Fatal(err)
	}
	subscription, err := executor.Subscribe(t.Context(), NamingSubscribeRequest{
		namespaceID: "nekiro", serviceName: "runtime-b", groupName: "NEKIRO", clusterName: "DEFAULT",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer subscription.Close()
	var response listResponse
	if err := json.Unmarshal(subscription.InitialPayload(), &response); err != nil || len(response.Hosts) == 0 {
		t.Fatalf("initial ServiceInfo hosts=%d error=%v", len(response.Hosts), err)
	}
}

func TestDecodeSubscriberPushRequiresExactSubscribedService(t *testing.T) {
	request := NamingSubscribeRequest{namespaceID: "nekiro", serviceName: "runtime-b", groupName: "NEKIRO", clusterName: "DEFAULT"}
	valid := `{"requestId":"push-1","module":"naming","serviceInfo":{"name":"runtime-b","groupName":"NEKIRO","clusters":"DEFAULT","hosts":[]}}`
	if payload, err := decodeSubscriberPush([]byte(valid), request); err != nil || string(payload) != `{"name":"runtime-b","groupName":"NEKIRO","clusters":"DEFAULT","hosts":[]}` {
		t.Fatalf("valid push payload=%s error=%v", payload, err)
	}
	for name, payload := range map[string]string{
		"missing request": `{"module":"naming","serviceInfo":{"name":"runtime-b","groupName":"NEKIRO","clusters":"DEFAULT"}}`,
		"wrong module":    `{"requestId":"push-1","module":"config","serviceInfo":{"name":"runtime-b","groupName":"NEKIRO","clusters":"DEFAULT"}}`,
		"wrong service":   `{"requestId":"push-1","module":"naming","serviceInfo":{"name":"runtime-a","groupName":"NEKIRO","clusters":"DEFAULT"}}`,
		"wrong group":     `{"requestId":"push-1","module":"naming","serviceInfo":{"name":"runtime-b","groupName":"OTHER","clusters":"DEFAULT"}}`,
		"wrong cluster":   `{"requestId":"push-1","module":"naming","serviceInfo":{"name":"runtime-b","groupName":"NEKIRO","clusters":"OTHER"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeSubscriberPush([]byte(payload), request); err == nil {
				t.Fatal("invalid push accepted")
			}
		})
	}
}

type fakeNamingGRPC struct {
	nacosgrpc.UnimplementedRequestServer
	nacosgrpc.UnimplementedBiRequestStreamServer
	setup           chan struct{}
	push            chan []byte
	ack             chan string
	serverCheckType string
	serverCheckBody any
	subscribeType   string
	subscribeBody   any
}

func newFakeNamingGRPC() *fakeNamingGRPC {
	return &fakeNamingGRPC{
		setup: make(chan struct{}), push: make(chan []byte, 1), ack: make(chan string, 2),
		serverCheckType: "ServerCheckResponse",
		serverCheckBody: map[string]any{"resultCode": 200, "errorCode": 0, "success": true, "connectionId": "connection-1"},
		subscribeType:   "SubscribeServiceResponse",
		subscribeBody:   map[string]any{"resultCode": 200, "errorCode": 0, "success": true, "serviceInfo": json.RawMessage(initialServiceInfo)},
	}
}

func (provider *fakeNamingGRPC) Request(ctx context.Context, request *nacosgrpc.Payload) (*nacosgrpc.Payload, error) {
	switch request.GetMetadata().GetType() {
	case "ServerCheckRequest":
		if request.GetMetadata().GetHeaders()["accessToken"] != "token-a" {
			return nil, context.Canceled
		}
		return testPayload(provider.serverCheckType, provider.serverCheckBody), nil
	case "SubscribeServiceRequest":
		select {
		case <-provider.setup:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		var body struct {
			Namespace string `json:"namespace"`
			Service   string `json:"serviceName"`
			Group     string `json:"groupName"`
			Clusters  string `json:"clusters"`
		}
		if json.Unmarshal(request.GetBody().GetValue(), &body) != nil || body.Namespace != "public" || body.Service != "runtime-b" || body.Group != "NEKIRO" || body.Clusters != "DEFAULT" {
			return nil, context.Canceled
		}
		return testPayload(provider.subscribeType, provider.subscribeBody), nil
	default:
		return nil, context.Canceled
	}
}

func (provider *fakeNamingGRPC) RequestBiStream(stream grpc.BidiStreamingServer[nacosgrpc.Payload, nacosgrpc.Payload]) error {
	setup, err := stream.Recv()
	if err != nil || setup.GetMetadata().GetType() != "ConnectionSetupRequest" {
		return err
	}
	close(provider.setup)
	select {
	case payload := <-provider.push:
		var serviceInfo map[string]any
		if json.Unmarshal(payload, &serviceInfo) != nil {
			return context.Canceled
		}
		serviceInfo["name"] = "runtime-b"
		serviceInfo["groupName"] = "NEKIRO"
		serviceInfo["clusters"] = "DEFAULT"
		request := testPayload("NotifySubscriberRequest", map[string]any{
			"requestId": "push-1", "module": "naming", "serviceInfo": serviceInfo,
		})
		if err := stream.Send(request); err != nil {
			return err
		}
		response, err := stream.Recv()
		if err != nil {
			return err
		}
		provider.ack <- response.GetMetadata().GetType()
	case <-stream.Context().Done():
		return stream.Context().Err()
	}
	<-stream.Context().Done()
	return stream.Context().Err()
}

func testPayload(messageType string, body any) *nacosgrpc.Payload {
	data, _ := json.Marshal(body)
	return &nacosgrpc.Payload{Metadata: &nacosgrpc.Metadata{Type: messageType, ClientIp: "127.0.0.1"}, Body: &anypb.Any{Value: data}}
}
