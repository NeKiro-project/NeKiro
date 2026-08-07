package gateway

import (
	"errors"
	"testing"
)

func TestCapabilitiesAreSortedImmutableSets(t *testing.T) {
	caps, err := NewCapabilities(CapabilityDrain, CapabilityForwarding, CapabilitySSEFlush)
	if err != nil {
		t.Fatalf("NewCapabilities: %v", err)
	}
	values := caps.Values()
	if len(values) != 3 || values[0] != CapabilityDrain || values[1] != CapabilityForwarding || values[2] != CapabilitySSEFlush {
		t.Fatalf("capability order = %v", values)
	}
	values[0] = CapabilityInstanceSelection
	if !caps.Supports(CapabilityDrain) || caps.Supports(CapabilityInstanceSelection) {
		t.Fatalf("capability set changed through returned slice: %v", caps.Values())
	}
	missing := caps.Missing(mustCapabilities(t, CapabilityDrain, CapabilityRetryPolicyControl))
	if len(missing) != 1 || missing[0] != CapabilityRetryPolicyControl {
		t.Fatalf("missing = %v", missing)
	}
}

func TestCapabilitiesRejectUnknownAndDuplicateValues(t *testing.T) {
	if _, err := NewCapabilities(CapabilityDrain, CapabilityDrain); !errors.Is(err, ErrInvalid) {
		t.Fatalf("duplicate capability error = %v, want invalid", err)
	}
	if _, err := NewCapabilities(Capability("provider-secret")); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unknown capability error = %v, want invalid", err)
	}
}

func mustCapabilities(t testing.TB, values ...Capability) Capabilities {
	t.Helper()
	caps, err := NewCapabilities(values...)
	if err != nil {
		t.Fatalf("NewCapabilities: %v", err)
	}
	return caps
}
