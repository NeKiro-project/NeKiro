package nacosdirectory

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	configcenter "github.com/NeKiro-project/NeKiro/config_center"
	"github.com/NeKiro-project/NeKiro/registry"
	registrynacos "github.com/NeKiro-project/NeKiro/registry/nacos"
)

type stubReader struct {
	snapshot configcenter.Snapshot
	err      error
}

func (reader stubReader) Get(context.Context, configcenter.Key) (configcenter.Snapshot, error) {
	return reader.snapshot, reader.err
}
func (stubReader) Close() error { return nil }

func TestDirectoryJoinsExactBindingAndNacosInstances(t *testing.T) {
	key, _ := configcenter.ParseKey("router/nacos-bindings")
	digest := strings.Repeat("a", 64)
	payload := []byte(`{"schemaVersion":"1","revision":"stack-1","targets":[{"agentId":"runtime-b","agentCardVersion":"1.0.0","releaseId":"release-b","cardDigest":"` + digest + `","canonicalEndpoint":"http://runtime-b:8092/","audience":"http://runtime-b:8092","serviceName":"runtime-b","groupName":"NEKIRO","clusterName":"DEFAULT"}]}`)
	snapshot, _ := configcenter.NewPresentSnapshot(key, payload, configcenter.UnscopedRevision())
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"hosts":[{"instanceId":"provider-generated","ip":"172.28.0.12","port":8092,"healthy":true,"enabled":true,"ephemeral":true,"clusterName":"DEFAULT","metadata":{"nekiro.instanceId":"runtime-b-directory"}}]}`))
	}))
	defer server.Close()
	directory, err := New(stubReader{snapshot: snapshot}, key, registrynacos.DirectoryConfig{APIOrigin: server.URL + "/nacos", NamespaceID: "public", PortName: "a2a", MaxResponseBytes: 4096, AuthMode: registrynacos.AuthNone, Executor: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	target, _ := registry.NewReleaseTarget(registry.ReleaseTargetInput{AgentID: "runtime-b", AgentCardVersion: "1.0.0", ReleaseID: "release-b", CardDigest: digest, CanonicalEndpoint: "http://runtime-b:8092/", Audience: "http://runtime-b:8092"})
	result, err := directory.Snapshot(t.Context(), target)
	if err != nil || len(result.Instances()) != 1 || result.Instances()[0].ID() != "runtime-b-directory" {
		t.Fatalf("Snapshot=%#v error=%v", result, err)
	}
	other, _ := registry.NewReleaseTarget(registry.ReleaseTargetInput{AgentID: "runtime-b", AgentCardVersion: "1.0.0", ReleaseID: "release-other", CardDigest: digest, CanonicalEndpoint: "http://runtime-b:8092/", Audience: "http://runtime-b:8092"})
	if _, err := directory.Snapshot(t.Context(), other); !errors.Is(err, registry.ErrMissing) {
		t.Fatalf("unbound Snapshot error=%v", err)
	}
}

func TestDirectoryCheckRejectsInvalidOrMissingBindings(t *testing.T) {
	key, _ := configcenter.ParseKey("router/nacos-bindings")
	invalid, _ := configcenter.NewPresentSnapshot(key, []byte(`{"schemaVersion":"1","revision":"stack-1","targets":[{"unknown":true}]}`), configcenter.UnscopedRevision())
	for name, reader := range map[string]stubReader{
		"missing": {err: configcenter.ErrMissing},
		"invalid": {snapshot: invalid},
	} {
		t.Run(name, func(t *testing.T) {
			directory, _ := New(reader, key, registrynacos.DirectoryConfig{APIOrigin: "http://nacos.test/nacos", NamespaceID: "public", PortName: "a2a", MaxResponseBytes: 4096, AuthMode: registrynacos.AuthNone, Executor: http.DefaultClient})
			err := directory.Check(t.Context())
			if name == "missing" && !errors.Is(err, registry.ErrMissing) || name == "invalid" && !errors.Is(err, registry.ErrInvalid) {
				t.Fatalf("Check error=%v", err)
			}
		})
	}
}
