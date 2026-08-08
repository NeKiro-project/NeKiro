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
	closes   int
	closeErr error
}

func (reader *stubReader) Get(context.Context, configcenter.Key) (configcenter.Snapshot, error) {
	reader.gets++
	return reader.snapshot, reader.err
}
func (reader *stubReader) Close() error {
	reader.closes++
	return reader.closeErr
}

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

func TestDirectoryLifecycleExposesOnlySnapshotCapability(t *testing.T) {
	key, _ := configcenter.ParseKey("router/instance-directory")
	reader := &stubReader{snapshot: mustSnapshot(t, key, []byte(`{"schemaVersion":"1","revision":"stack-1","targets":[]}`)), closeErr: errors.New("close failed")}
	directory, err := New(reader, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := directory.Check(t.Context()); err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !directory.Capabilities().Supports(registry.CapabilitySnapshot) || directory.Capabilities().Supports(registry.CapabilityObserve) {
		t.Fatalf("capabilities=%#v", directory.Capabilities())
	}
	if _, err := directory.Observe(t.Context(), registry.ReleaseTarget{}); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("Observe error=%v", err)
	}
	if err := directory.Close(); err != reader.closeErr || reader.closes != 1 {
		t.Fatalf("Close error=%v closes=%d", err, reader.closes)
	}
	if _, err := New(nil, key); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("New(nil) error=%v", err)
	}
	if _, err := New(reader, configcenter.Key{}); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("New(invalid key) error=%v", err)
	}
}

func TestDirectoryMapsConfigCenterFailures(t *testing.T) {
	for _, test := range []struct {
		name   string
		source error
		want   error
	}{
		{name: "missing", source: configcenter.ErrMissing, want: registry.ErrMissing},
		{name: "invalid", source: configcenter.ErrInvalid, want: registry.ErrInvalid},
		{name: "unsafe", source: configcenter.ErrUnsafeState, want: registry.ErrInvalid},
		{name: "too large", source: configcenter.ErrPayloadTooLarge, want: registry.ErrInvalid},
		{name: "unauthorized", source: configcenter.ErrUnauthorized, want: registry.ErrUnauthorized},
		{name: "canceled", source: configcenter.ErrCanceled, want: registry.ErrCanceled},
		{name: "closed", source: configcenter.ErrReaderClosed, want: registry.ErrClosed},
		{name: "unavailable", source: errors.New("provider detail"), want: registry.ErrUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := mapSourceError(test.source); !errors.Is(got, test.want) {
				t.Fatalf("mapSourceError(%v)=%v want=%v", test.source, got, test.want)
			}
		})
	}
}

func TestDirectoryAddressValidationIsCanonicalAndProviderBounded(t *testing.T) {
	for _, test := range []struct {
		name        string
		addressType registry.AddressType
		address     string
		want        bool
	}{
		{name: "IPv4", addressType: registry.AddressTypeIPv4, address: "127.0.0.1", want: true},
		{name: "noncanonical IPv4", addressType: registry.AddressTypeIPv4, address: "127.000.000.001"},
		{name: "IPv6", addressType: registry.AddressTypeIPv6, address: "::1", want: true},
		{name: "IPv4 as IPv6", addressType: registry.AddressTypeIPv6, address: "127.0.0.1"},
		{name: "DNS", addressType: registry.AddressTypeDNS, address: "runtime-b.default", want: true},
		{name: "uppercase DNS", addressType: registry.AddressTypeDNS, address: "Runtime-B"},
		{name: "unknown", addressType: registry.AddressType("Unix"), address: "runtime-b"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := validAddress(test.addressType, test.address); got != test.want {
				t.Fatalf("validAddress(%q, %q)=%v want=%v", test.addressType, test.address, got, test.want)
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
