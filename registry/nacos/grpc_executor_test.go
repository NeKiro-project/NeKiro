package nacos

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"testing"
	"time"

	nacosgrpc "github.com/nacos-group/nacos-sdk-go/v2/api/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/anypb"
)

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
	if err != nil || string(payload) != changedServiceInfo {
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

type fakeNamingGRPC struct {
	nacosgrpc.UnimplementedRequestServer
	nacosgrpc.UnimplementedBiRequestStreamServer
	setup chan struct{}
	push  chan []byte
	ack   chan string
}

func newFakeNamingGRPC() *fakeNamingGRPC {
	return &fakeNamingGRPC{setup: make(chan struct{}), push: make(chan []byte, 1), ack: make(chan string, 2)}
}

func (provider *fakeNamingGRPC) Request(ctx context.Context, request *nacosgrpc.Payload) (*nacosgrpc.Payload, error) {
	switch request.GetMetadata().GetType() {
	case "ServerCheckRequest":
		if request.GetMetadata().GetHeaders()["accessToken"] != "token-a" {
			return nil, context.Canceled
		}
		return testPayload("ServerCheckResponse", map[string]any{"resultCode": 200, "errorCode": 0, "success": true, "connectionId": "connection-1"}), nil
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
		return testPayload("SubscribeServiceResponse", map[string]any{"resultCode": 200, "errorCode": 0, "success": true, "serviceInfo": json.RawMessage(initialServiceInfo)}), nil
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
		request := testPayload("NotifySubscriberRequest", map[string]any{
			"requestId": "push-1", "namespace": "public", "serviceName": "runtime-b", "groupName": "NEKIRO",
			"module": "naming", "serviceInfo": json.RawMessage(payload),
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
