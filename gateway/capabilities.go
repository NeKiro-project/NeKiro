package gateway

import "sort"

// Capability identifies one optional external Gateway behavior. A provider
// must advertise a capability before accepting a RouteSpec that requires it.
type Capability string

const (
	CapabilityForwarding           Capability = "forwarding"
	CapabilitySSEForwarding        Capability = "sse_forwarding"
	CapabilitySSEFlush             Capability = "sse_flush"
	CapabilityCancellationAffinity Capability = "cancellation_affinity"
	CapabilityDrain                Capability = "drain"
	CapabilityDataPlaneReadiness   Capability = "data_plane_readiness"
	CapabilityRetryPolicyControl   Capability = "retry_policy_control"
	CapabilityInstanceSelection    Capability = "instance_selection"
)

// Capabilities is an immutable set of explicitly advertised capabilities.
// The zero value represents an explicit empty set when returned by a provider;
// callers that construct requirements should use NewCapabilities so unknown
// and duplicate values are rejected.
type Capabilities struct {
	set map[Capability]struct{}
}

// NewCapabilities validates and copies an immutable capability set.
func NewCapabilities(values ...Capability) (Capabilities, error) {
	set := make(map[Capability]struct{}, len(values))
	for _, value := range values {
		if !validCapability(value) {
			return Capabilities{}, newInvalidError("capability")
		}
		if _, exists := set[value]; exists {
			return Capabilities{}, newInvalidError("duplicate_capability")
		}
		set[value] = struct{}{}
	}
	return Capabilities{set: set}, nil
}

// Validate verifies that a capability set contains only v1 capabilities.
func (c Capabilities) Validate() error {
	for value := range c.set {
		if !validCapability(value) {
			return newInvalidError("capability")
		}
	}
	return nil
}

func validCapability(value Capability) bool {
	switch value {
	case CapabilityForwarding, CapabilitySSEForwarding, CapabilitySSEFlush,
		CapabilityCancellationAffinity, CapabilityDrain, CapabilityDataPlaneReadiness,
		CapabilityRetryPolicyControl, CapabilityInstanceSelection:
		return true
	default:
		return false
	}
}

// Supports reports whether this capability was explicitly advertised.
func (c Capabilities) Supports(value Capability) bool {
	_, ok := c.set[value]
	return ok
}

// Values returns a sorted copy of the capability set.
func (c Capabilities) Values() []Capability {
	values := make([]Capability, 0, len(c.set))
	for value := range c.set {
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return values
}

// Missing returns the sorted required capabilities not present in c.
func (c Capabilities) Missing(required Capabilities) []Capability {
	missing := make([]Capability, 0)
	for _, capability := range required.Values() {
		if !c.Supports(capability) {
			missing = append(missing, capability)
		}
	}
	return missing
}

// Equal reports whether both capability sets contain exactly the same values.
func (c Capabilities) Equal(other Capabilities) bool {
	if len(c.set) != len(other.set) {
		return false
	}
	for value := range c.set {
		if !other.Supports(value) {
			return false
		}
	}
	return true
}

func cloneCapabilities(capabilities Capabilities) Capabilities {
	copyCapabilities, err := NewCapabilities(capabilities.Values()...)
	if err != nil {
		panic("gateway received invalid capabilities")
	}
	return copyCapabilities
}
