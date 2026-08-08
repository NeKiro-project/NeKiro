package configdirectory

import (
	"context"
	"errors"
	"strings"
	"testing"

	configcenter "github.com/NeKiro-project/NeKiro/config_center"
	"github.com/NeKiro-project/NeKiro/registry"
)

type stubReader struct {
	snapshot configcenter.Snapshot
	err      error
	gets     int
}

func (reader *stubReader) Get(context.Context, configcenter.Key) (configcenter.Snapshot, error) {
	reader.gets++
	return reader.snapshot, reader.err
}
func (*stubReader) Close() error { return nil }

func TestDirectoryReturnsExactImmutableSnapshot(t *testing.T) {
	key, _ := configcenter.ParseKey("router/instance-directory")
	digest := strings.Repeat("a", 64)
	payload := []byte(`{"schemaVersion":"1","revision":"stack-1","targets":[{"agentId":"runtime-b","agentCardVersion":"1.0.0","releaseId":"release-b","cardDigest":"` + digest + `","canonicalEndpoint":"http://runtime-b:8092/","audience":"http://runtime-b:8092","instances":[{"instanceId":"runtime-b-directory","ready":true,"serving":true,"terminating":false,"endpoints":[{"addressType":"DNS","address":"runtime-b-directory","portName":"a2a","port":8092,"protocol":"TCP"}]}]}]}`)
	snapshot, err := configcenter.NewPresentSnapshot(key, payload, configcenter.UnscopedRevision())
	if err != nil {
		t.Fatal(err)
	}
	reader := &stubReader{snapshot: snapshot}
	directory, err := New(reader, key)
	if err != nil {
		t.Fatal(err)
	}
	target, err := registry.NewReleaseTarget(registry.ReleaseTargetInput{AgentID: "runtime-b", AgentCardVersion: "1.0.0", ReleaseID: "release-b", CardDigest: digest, CanonicalEndpoint: "http://runtime-b:8092/", Audience: "http://runtime-b:8092"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := directory.Snapshot(t.Context(), target)
	if err != nil {
		t.Fatalf("Snapshot failed: %v", err)
	}
	if reader.gets != 1 || result.State() != registry.SnapshotStatePopulated || len(result.Instances()) != 1 || result.Instances()[0].ID() != "runtime-b-directory" {
		t.Fatalf("result=%#v gets=%d", result, reader.gets)
	}
	if got := result.Instances()[0].Endpoints()[0]; got.AddressType() != registry.AddressTypeDNS || got.Address() != "runtime-b-directory" {
		t.Fatalf("endpoint=%#v", got)
	}
}

func TestDirectoryRejectsMissingAndInvalidDocuments(t *testing.T) {
	key, _ := configcenter.ParseKey("router/instance-directory")
	for name, reader := range map[string]*stubReader{
		"missing": {err: configcenter.ErrMissing},
		"invalid": {snapshot: mustSnapshot(t, key, []byte(`{"schemaVersion":"1","revision":"stack-1","targets":[{"unknown":true}]}`))},
	} {
		t.Run(name, func(t *testing.T) {
			directory, _ := New(reader, key)
			target, _ := registry.NewReleaseTarget(registry.ReleaseTargetInput{AgentID: "runtime-b", AgentCardVersion: "1.0.0", ReleaseID: "release-b", CardDigest: strings.Repeat("a", 64), CanonicalEndpoint: "http://runtime-b:8092/", Audience: "http://runtime-b:8092"})
			_, err := directory.Snapshot(t.Context(), target)
			if name == "missing" && !errors.Is(err, registry.ErrMissing) || name == "invalid" && !errors.Is(err, registry.ErrInvalid) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func mustSnapshot(t *testing.T, key configcenter.Key, payload []byte) configcenter.Snapshot {
	t.Helper()
	snapshot, err := configcenter.NewPresentSnapshot(key, payload, configcenter.UnscopedRevision())
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}
