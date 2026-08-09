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
	closeErr error
}

func (reader stubReader) Get(context.Context, configcenter.Key) (configcenter.Snapshot, error) {
	return reader.snapshot, reader.err
}
func (reader stubReader) Close() error { return reader.closeErr }

type stubBindingSource struct{}

func (stubBindingSource) Binding(context.Context, registry.ReleaseTarget) (registrynacos.Binding, error) {
	return registrynacos.Binding{}, registry.ErrMissing
}

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

func TestDirectoryExposesBindingCapabilitiesAndLifecycle(t *testing.T) {
	key, _ := configcenter.ParseKey("router.nacos-bindings")
	digest := strings.Repeat("a", 64)
	payload := []byte(`{"schemaVersion":"1","revision":"stack-1","targets":[{"agentId":"runtime-b","agentCardVersion":"1.0.0","releaseId":"release-b","cardDigest":"` + digest + `","canonicalEndpoint":"http://runtime-b:8092/","audience":"http://runtime-b:8092","serviceName":"runtime-b","groupName":"NEKIRO","clusterName":"DEFAULT"}]}`)
	snapshot, _ := configcenter.NewPresentSnapshot(key, payload, configcenter.UnscopedRevision())
	closeErr := errors.New("reader close failed")
	directory, err := New(stubReader{snapshot: snapshot, closeErr: closeErr}, key, registrynacos.DirectoryConfig{APIOrigin: "http://nacos.test/nacos", NamespaceID: "public", PortName: "a2a", MaxResponseBytes: 4096, AuthMode: registrynacos.AuthNone, Executor: http.DefaultClient})
	if err != nil {
		t.Fatal(err)
	}
	target, _ := registry.NewReleaseTarget(registry.ReleaseTargetInput{AgentID: "runtime-b", AgentCardVersion: "1.0.0", ReleaseID: "release-b", CardDigest: digest, CanonicalEndpoint: "http://runtime-b:8092/", Audience: "http://runtime-b:8092"})
	binding, err := directory.Binding(t.Context(), target)
	if err != nil || !binding.Target().Equal(target) || binding.ServiceName() != "runtime-b" || binding.GroupName() != "NEKIRO" || binding.ClusterName() != "DEFAULT" {
		t.Fatalf("Binding=%#v error=%v", binding, err)
	}
	if err := directory.Check(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := directory.Observe(t.Context(), target); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("Observe error=%v", err)
	}
	if !directory.Capabilities().Supports(registry.CapabilitySnapshot) {
		t.Fatal("snapshot capability is missing")
	}
	if err := directory.Close(); !errors.Is(err, closeErr) {
		t.Fatalf("Close error=%v", err)
	}
}

func TestDirectoryRejectsInvalidConstructionAndDocuments(t *testing.T) {
	key, _ := configcenter.ParseKey("router.nacos-bindings")
	provider := registrynacos.DirectoryConfig{APIOrigin: "http://nacos.test/nacos", NamespaceID: "public", PortName: "a2a", MaxResponseBytes: 4096, AuthMode: registrynacos.AuthNone, Executor: http.DefaultClient}
	if _, err := New(nil, key, provider); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("nil reader error=%v", err)
	}
	if _, err := New(stubReader{}, configcenter.Key{}, provider); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("invalid key error=%v", err)
	}
	owned := provider
	owned.Bindings = stubBindingSource{}
	if _, err := New(stubReader{}, key, owned); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("preowned binding source error=%v", err)
	}
	invalidProvider := provider
	invalidProvider.PortName = ""
	if _, err := New(stubReader{}, key, invalidProvider); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("invalid provider error=%v", err)
	}
	missing, _ := configcenter.NewMissingSnapshot(key, configcenter.UnscopedRevision())
	directory, _ := New(stubReader{snapshot: missing}, key, provider)
	if err := directory.Check(t.Context()); !errors.Is(err, registry.ErrMissing) {
		t.Fatalf("absent document error=%v", err)
	}
	digest := strings.Repeat("a", 64)
	target := `{"agentId":"runtime-b","agentCardVersion":"1.0.0","releaseId":"release-b","cardDigest":"` + digest + `","canonicalEndpoint":"http://runtime-b:8092/","audience":"http://runtime-b:8092","serviceName":"runtime-b","groupName":"NEKIRO","clusterName":"DEFAULT"}`
	duplicate, _ := configcenter.NewPresentSnapshot(key, []byte(`{"schemaVersion":"1","revision":"stack-1","targets":[`+target+`,`+target+`]}`), configcenter.UnscopedRevision())
	directory, _ = New(stubReader{snapshot: duplicate}, key, provider)
	if err := directory.Check(t.Context()); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("duplicate binding error=%v", err)
	}
}

func TestDirectoryMapsEveryConfigSourceFailure(t *testing.T) {
	key, _ := configcenter.ParseKey("router.nacos-bindings")
	provider := registrynacos.DirectoryConfig{APIOrigin: "http://nacos.test/nacos", NamespaceID: "public", PortName: "a2a", MaxResponseBytes: 4096, AuthMode: registrynacos.AuthNone, Executor: http.DefaultClient}
	for name, test := range map[string]struct {
		source error
		want   error
	}{
		"missing":        {source: configcenter.ErrMissing, want: registry.ErrMissing},
		"invalid":        {source: configcenter.ErrInvalid, want: registry.ErrInvalid},
		"unsafe":         {source: configcenter.ErrUnsafeState, want: registry.ErrInvalid},
		"large":          {source: configcenter.ErrPayloadTooLarge, want: registry.ErrInvalid},
		"unauthorized":   {source: configcenter.ErrUnauthorized, want: registry.ErrUnauthorized},
		"canceled":       {source: configcenter.ErrCanceled, want: registry.ErrCanceled},
		"closed":         {source: configcenter.ErrReaderClosed, want: registry.ErrClosed},
		"unavailable":    {source: configcenter.ErrUnavailable, want: registry.ErrUnavailable},
		"unknown source": {source: errors.New("unknown"), want: registry.ErrUnavailable},
	} {
		t.Run(name, func(t *testing.T) {
			directory, _ := New(stubReader{err: test.source}, key, provider)
			if err := directory.Check(t.Context()); !errors.Is(err, test.want) {
				t.Fatalf("Check error=%v want=%v", err, test.want)
			}
		})
	}
}
