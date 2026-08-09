package nacos

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NeKiro-project/NeKiro/registry"
)

type bindingSource struct{ binding Binding }

func (source bindingSource) Binding(context.Context, registry.ReleaseTarget) (Binding, error) {
	return source.binding, nil
}

func TestDirectoryReadsOneExactNacosSnapshot(t *testing.T) {
	target := testTarget(t)
	binding, _ := NewBinding(BindingInput{Target: target, ServiceName: "runtime-b", GroupName: "NEKIRO", ClusterName: "DEFAULT"})
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		query := request.URL.Query()
		if request.URL.Path != "/nacos/v1/ns/instance/list" || query.Get("serviceName") != "runtime-b" || query.Get("groupName") != "NEKIRO" || query.Get("clusters") != "DEFAULT" || query.Get("namespaceId") != "public" || query.Get("healthyOnly") != "false" {
			t.Errorf("request URL=%s", request.URL.String())
		}
		_, _ = writer.Write([]byte(`{"hosts":[{"instanceId":"provider-generated","ip":"172.28.0.12","port":8092,"healthy":true,"enabled":true,"ephemeral":true,"clusterName":"DEFAULT","metadata":{"nekiro.instanceId":"runtime-b-directory"}}]}`))
	}))
	t.Cleanup(server.Close)
	directory, err := NewDirectory(DirectoryConfig{APIOrigin: server.URL + "/nacos", NamespaceID: "public", PortName: "a2a", MaxResponseBytes: 4096, AuthMode: AuthNone, Executor: server.Client(), Bindings: bindingSource{binding}})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := directory.Snapshot(t.Context(), target)
	if err != nil || requests != 1 || snapshot.State() != registry.SnapshotStatePopulated || len(snapshot.Instances()) != 1 || snapshot.Instances()[0].ID() != "runtime-b-directory" || snapshot.Instances()[0].State() != registry.InstanceStateReady {
		t.Fatalf("Snapshot=%#v requests=%d error=%v", snapshot, requests, err)
	}
}

func TestDirectoryKeepsUnhealthyNacosInstanceUnavailable(t *testing.T) {
	target := testTarget(t)
	binding, _ := NewBinding(BindingInput{Target: target, ServiceName: "runtime-b", GroupName: "NEKIRO", ClusterName: "DEFAULT"})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"hosts":[{"instanceId":"provider-generated","ip":"172.28.0.12","port":8092,"healthy":false,"enabled":true,"ephemeral":true,"clusterName":"DEFAULT","metadata":{"nekiro.instanceId":"runtime-b-directory"}}]}`))
	}))
	defer server.Close()
	directory, _ := NewDirectory(DirectoryConfig{APIOrigin: server.URL + "/nacos", NamespaceID: "public", PortName: "a2a", MaxResponseBytes: 4096, AuthMode: AuthNone, Executor: server.Client(), Bindings: bindingSource{binding}})
	snapshot, err := directory.Snapshot(t.Context(), target)
	if err != nil || snapshot.Instances()[0].State() != registry.InstanceStateUnavailable {
		t.Fatalf("Snapshot=%#v error=%v", snapshot, err)
	}
}

func TestDirectoryRejectsProviderIdentityAndFailureChanges(t *testing.T) {
	target := testTarget(t)
	binding, _ := NewBinding(BindingInput{Target: target, ServiceName: "runtime-b", GroupName: "NEKIRO", ClusterName: "DEFAULT"})
	for name, test := range map[string]struct {
		status int
		body   string
		want   error
	}{
		"unauthorized":        {status: http.StatusUnauthorized, want: registry.ErrUnauthorized},
		"unavailable":         {status: http.StatusServiceUnavailable, want: registry.ErrUnavailable},
		"persistent":          {status: http.StatusOK, body: `{"hosts":[{"ip":"172.28.0.12","port":8092,"healthy":true,"enabled":true,"ephemeral":false,"clusterName":"DEFAULT","metadata":{"nekiro.instanceId":"runtime-b-directory"}}]}`, want: registry.ErrInvalid},
		"wrong cluster":       {status: http.StatusOK, body: `{"hosts":[{"ip":"172.28.0.12","port":8092,"healthy":true,"enabled":true,"ephemeral":true,"clusterName":"OTHER","metadata":{"nekiro.instanceId":"runtime-b-directory"}}]}`, want: registry.ErrInvalid},
		"missing instance ID": {status: http.StatusOK, body: `{"hosts":[{"ip":"172.28.0.12","port":8092,"healthy":true,"enabled":true,"ephemeral":true,"clusterName":"DEFAULT","metadata":{}}]}`, want: registry.ErrInvalid},
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(test.status)
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()
			directory, _ := NewDirectory(DirectoryConfig{APIOrigin: server.URL + "/nacos", NamespaceID: "public", PortName: "a2a", MaxResponseBytes: 4096, AuthMode: AuthNone, Executor: server.Client(), Bindings: bindingSource{binding}})
			_, err := directory.Snapshot(t.Context(), target)
			if !errors.Is(err, test.want) {
				t.Fatalf("Snapshot error=%v want=%v", err, test.want)
			}
		})
	}
}

func testTarget(t *testing.T) registry.ReleaseTarget {
	t.Helper()
	target, err := registry.NewReleaseTarget(registry.ReleaseTargetInput{AgentID: "runtime-b", AgentCardVersion: "1.0.0", ReleaseID: "release-b", CardDigest: strings.Repeat("a", 64), CanonicalEndpoint: "http://runtime-b:8092/", Audience: "http://runtime-b:8092"})
	if err != nil {
		t.Fatal(err)
	}
	return target
}
