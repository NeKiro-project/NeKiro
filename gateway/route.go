package gateway

import (
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/NeKiro-project/NeKiro/contracts"
)

var (
	safeIdentifierRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
	cardDigestRE     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	backendRefRE     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,255}$`)
)

// ProviderName identifies one provider implementation without exposing its
// control surface, endpoint, or credentials.
type ProviderName string

// NewProviderName validates a provider-safe provider name.
func NewProviderName(value string) (ProviderName, error) {
	name := ProviderName(value)
	if err := name.Validate(); err != nil {
		return "", err
	}
	return name, nil
}

// Validate verifies that a provider name is safe to expose in this contract.
func (n ProviderName) Validate() error {
	if !safeIdentifierRE.MatchString(string(n)) {
		return newInvalidError("provider_name")
	}
	return nil
}

func (n ProviderName) String() string { return string(n) }

// RouteKey identifies a provider-neutral desired route. It does not encode an
// endpoint, Agent Card, credential, or backend control-surface address.
type RouteKey struct {
	value string
}

// NewRouteKey validates one exact route key.
func NewRouteKey(value string) (RouteKey, error) {
	key := RouteKey{value: value}
	if err := key.Validate(); err != nil {
		return RouteKey{}, err
	}
	return key, nil
}

// Validate verifies that this value is a complete route key.
func (k RouteKey) Validate() error {
	if !safeIdentifierRE.MatchString(k.value) {
		return newInvalidError("route_key")
	}
	return nil
}

func (k RouteKey) Value() string  { return k.value }
func (k RouteKey) String() string { return k.value }

// Equal reports byte-exact route key equality.
func (k RouteKey) Equal(other RouteKey) bool { return k.value == other.value }

// RouteRevision is an exact desired or provider-observed revision token. It
// has no ordering semantics; stale state is reported explicitly by RouteState.
type RouteRevision struct {
	value string
}

// NewRouteRevision validates one exact route revision token.
func NewRouteRevision(value string) (RouteRevision, error) {
	revision := RouteRevision{value: value}
	if err := revision.Validate(); err != nil {
		return RouteRevision{}, err
	}
	return revision, nil
}

// Validate verifies that this value is a complete route revision token.
func (r RouteRevision) Validate() error {
	if !safeIdentifierRE.MatchString(r.value) {
		return newInvalidError("route_revision")
	}
	return nil
}

func (r RouteRevision) Value() string  { return r.value }
func (r RouteRevision) String() string { return r.value }

// Equal reports byte-exact revision equality.
func (r RouteRevision) Equal(other RouteRevision) bool { return r.value == other.value }

// BackendRef is an opaque provider-local backend reference. It intentionally
// accepts only an identifier-shaped value, never an arbitrary URI, credential,
// request payload, or alternate endpoint.
type BackendRef string

// NewBackendRef validates one opaque provider-local backend reference.
func NewBackendRef(value string) (BackendRef, error) {
	ref := BackendRef(value)
	if err := ref.Validate(); err != nil {
		return "", err
	}
	return ref, nil
}

// Validate verifies that a backend reference is safe for this contract.
func (r BackendRef) Validate() error {
	if !backendRefRE.MatchString(string(r)) || strings.Contains(string(r), "://") {
		return newInvalidError("backend_ref")
	}
	return nil
}

func (r BackendRef) String() string { return string(r) }

// DiscoveryOwner identifies the one component that owns discovery for a
// route. It is intentionally a single enum, so a route cannot name both
// Gateway and Router discovery owners.
type DiscoveryOwner string

const (
	DiscoveryOwnerGateway DiscoveryOwner = "gateway"
	DiscoveryOwnerRouter  DiscoveryOwner = "router"
)

// ReleaseProvenance is an immutable exact release identity. It does not carry
// an Agent Card or any mutable Catalog state.
type ReleaseProvenance struct {
	releaseID  string
	cardDigest string
}

func newReleaseProvenance(releaseID, cardDigest string) (ReleaseProvenance, error) {
	provenance := ReleaseProvenance{releaseID: releaseID, cardDigest: cardDigest}
	if err := provenance.Validate(); err != nil {
		return ReleaseProvenance{}, err
	}
	return provenance, nil
}

// Validate verifies that release provenance is complete and exact.
func (p ReleaseProvenance) Validate() error {
	if !safeIdentifierRE.MatchString(p.releaseID) {
		return newInvalidError("release_id")
	}
	if !cardDigestRE.MatchString(p.cardDigest) {
		return newInvalidError("card_digest")
	}
	return nil
}

func (p ReleaseProvenance) ReleaseID() string  { return p.releaseID }
func (p ReleaseProvenance) CardDigest() string { return p.cardDigest }

// Equal reports byte-exact release provenance equality.
func (p ReleaseProvenance) Equal(other ReleaseProvenance) bool {
	return p.releaseID == other.releaseID && p.cardDigest == other.cardDigest
}

// RouteSpecInput contains the complete immutable desired-route vocabulary.
// Every field is explicit: no release, endpoint, audience, discovery owner,
// capability, or backend reference is inferred.
type RouteSpecInput struct {
	Key                  RouteKey
	Revision             RouteRevision
	ReleaseID            string
	CardDigest           string
	AgentID              string
	AgentVersion         string
	EndpointOrigin       string
	EndpointPath         string
	Audience             string
	DiscoveryOwner       DiscoveryOwner
	BackendRef           BackendRef
	RequiredCapabilities []Capability
}

// RouteSpec is an immutable desired route for one exact Catalog release. It
// contains only allowlisted route identity and desired-state facts; it never
// carries an Agent Card, payload, credentials, headers, arbitrary upstream, or
// alternate release.
type RouteSpec struct {
	key            RouteKey
	revision       RouteRevision
	release        ReleaseProvenance
	agentID        string
	agentVersion   string
	endpointOrigin string
	endpointPath   string
	audience       string
	discoveryOwner DiscoveryOwner
	backendRef     BackendRef
	required       Capabilities
}

// NewRouteSpec validates and copies one complete desired route. It preserves
// the selected discovery owner but does not infer an alternate mode or a
// selector implementation.
func NewRouteSpec(input RouteSpecInput) (RouteSpec, error) {
	release, err := newReleaseProvenance(input.ReleaseID, input.CardDigest)
	if err != nil {
		return RouteSpec{}, err
	}
	required, err := NewCapabilities(input.RequiredCapabilities...)
	if err != nil {
		return RouteSpec{}, err
	}
	spec := RouteSpec{
		key:            input.Key,
		revision:       input.Revision,
		release:        release,
		agentID:        input.AgentID,
		agentVersion:   input.AgentVersion,
		endpointOrigin: input.EndpointOrigin,
		endpointPath:   input.EndpointPath,
		audience:       input.Audience,
		discoveryOwner: input.DiscoveryOwner,
		backendRef:     input.BackendRef,
		required:       required,
	}
	if err := spec.Validate(); err != nil {
		return RouteSpec{}, err
	}
	return spec, nil
}

// Validate verifies that this value is a complete immutable route. It does
// not resolve the endpoint or call a provider.
func (s RouteSpec) Validate() error {
	if err := s.key.Validate(); err != nil {
		return err
	}
	if err := s.revision.Validate(); err != nil {
		return err
	}
	if err := s.release.Validate(); err != nil {
		return err
	}
	if !safeIdentifierRE.MatchString(s.agentID) {
		return newInvalidError("agent_id")
	}
	if _, err := semver.StrictNewVersion(s.agentVersion); err != nil {
		return newInvalidError("agent_version")
	}
	if !validCanonicalOrigin(s.endpointOrigin) {
		return newInvalidError("endpoint_origin")
	}
	if !validCanonicalEndpointPath(s.endpointPath) {
		return newInvalidError("endpoint_path")
	}
	if len(s.CanonicalEndpoint()) > 2048 {
		return newInvalidError("canonical_endpoint")
	}
	if !validCanonicalOrigin(s.audience) {
		return newInvalidError("audience")
	}
	expectedAudience, err := contracts.CanonicalRouterAgentAudience(s.CanonicalEndpoint())
	if err != nil || expectedAudience != s.audience {
		return newInvalidError("audience")
	}
	if err := s.backendRef.Validate(); err != nil {
		return err
	}
	if err := s.required.Validate(); err != nil {
		return err
	}
	switch s.discoveryOwner {
	case DiscoveryOwnerGateway, DiscoveryOwnerRouter:
		return nil
	default:
		return newInvalidError("discovery_owner")
	}
}

func (s RouteSpec) Key() RouteKey                  { return s.key }
func (s RouteSpec) Revision() RouteRevision        { return s.revision }
func (s RouteSpec) Release() ReleaseProvenance     { return s.release }
func (s RouteSpec) ReleaseID() string              { return s.release.releaseID }
func (s RouteSpec) CardDigest() string             { return s.release.cardDigest }
func (s RouteSpec) AgentID() string                { return s.agentID }
func (s RouteSpec) AgentVersion() string           { return s.agentVersion }
func (s RouteSpec) EndpointOrigin() string         { return s.endpointOrigin }
func (s RouteSpec) EndpointPath() string           { return s.endpointPath }
func (s RouteSpec) Audience() string               { return s.audience }
func (s RouteSpec) DiscoveryOwner() DiscoveryOwner { return s.discoveryOwner }
func (s RouteSpec) BackendRef() BackendRef         { return s.backendRef }

// CanonicalEndpoint returns the exact route endpoint assembled from the
// independently validated immutable origin and path. It performs no
// normalization or substitution.
func (s RouteSpec) CanonicalEndpoint() string { return s.endpointOrigin + s.endpointPath }

// RequiredCapabilities returns an immutable copy of the route's requirements.
func (s RouteSpec) RequiredCapabilities() Capabilities { return cloneCapabilities(s.required) }

// Equal reports byte-exact equality across all desired route facts.
func (s RouteSpec) Equal(other RouteSpec) bool {
	return s.key.Equal(other.key) &&
		s.revision.Equal(other.revision) &&
		s.release.Equal(other.release) &&
		s.agentID == other.agentID &&
		s.agentVersion == other.agentVersion &&
		s.endpointOrigin == other.endpointOrigin &&
		s.endpointPath == other.endpointPath &&
		s.audience == other.audience &&
		s.discoveryOwner == other.discoveryOwner &&
		s.backendRef == other.backendRef &&
		s.required.Equal(other.required)
}

func validExactText(value string) bool {
	if value == "" || strings.TrimSpace(value) != value {
		return false
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

func validCanonicalOrigin(raw string) bool {
	if !validExactText(raw) {
		return false
	}
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Opaque != "" || parsed.Host == "" || parsed.User != nil ||
		parsed.Path != "" || parsed.RawPath != "" || parsed.RawQuery != "" || parsed.ForceQuery ||
		parsed.Fragment != "" || parsed.RawFragment != "" {
		return false
	}
	canonical, ok := canonicalOrigin(parsed)
	return ok && canonical == raw
}

func validCanonicalEndpointPath(raw string) bool {
	if !validExactText(raw) || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.ContainsAny(raw, "?#%") {
		return false
	}
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.IsAbs() || parsed.Opaque != "" || parsed.Host != "" || parsed.User != nil ||
		parsed.RawPath != "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || parsed.RawFragment != "" ||
		parsed.Path != raw || strings.ContainsAny(parsed.Path, "\\\r\n") {
		return false
	}
	for _, segment := range strings.Split(parsed.Path, "/") {
		if segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func canonicalOrigin(parsed *url.URL) (string, bool) {
	if parsed == nil || parsed.Scheme != strings.ToLower(parsed.Scheme) || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", false
	}
	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	if host == "" || strings.Contains(host, "%") || strings.ContainsAny(host, "\r\n") {
		return "", false
	}
	portText := parsed.Port()
	if portText == "" && strings.HasSuffix(parsed.Host, ":") {
		return "", false
	}
	port := 0
	var err error
	if portText != "" {
		port, err = strconv.Atoi(portText)
		if err != nil || port < 1 || port > 65535 {
			return "", false
		}
	}
	hostPort := host
	if net.ParseIP(host) != nil {
		if port != 0 {
			hostPort = net.JoinHostPort(host, strconv.Itoa(port))
		} else if strings.Contains(host, ":") {
			hostPort = "[" + host + "]"
		}
	} else if port != 0 {
		hostPort = host + ":" + strconv.Itoa(port)
	}
	if (parsed.Scheme == "http" && port == 80) || (parsed.Scheme == "https" && port == 443) {
		if strings.Contains(host, ":") {
			hostPort = "[" + host + "]"
		} else {
			hostPort = host
		}
	}
	return parsed.Scheme + "://" + hostPort, true
}
