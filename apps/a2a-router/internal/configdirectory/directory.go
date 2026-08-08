package configdirectory

import (
	"context"
	"errors"
	"net"
	"regexp"

	configcenter "github.com/NeKiro-project/NeKiro/config_center"
	"github.com/NeKiro-project/NeKiro/contracts"
	"github.com/NeKiro-project/NeKiro/registry"
)

var dnsAddressPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,62})(?:\.[a-z0-9](?:[a-z0-9-]{0,62}))*$`)

type Reader interface {
	Get(context.Context, configcenter.Key) (configcenter.Snapshot, error)
	Close() error
}

// Directory turns one Router-owned configuration document into point-in-time
// Instance Registry snapshots. It deliberately does not advertise Observe.
type Directory struct {
	reader       Reader
	key          configcenter.Key
	capabilities registry.Capabilities
}

func New(reader Reader, key configcenter.Key) (*Directory, error) {
	if reader == nil || !key.Valid() {
		return nil, registry.ErrInvalid
	}
	capabilities, err := registry.NewCapabilities(registry.CapabilitySnapshot)
	if err != nil {
		return nil, err
	}
	return &Directory{reader: reader, key: key, capabilities: capabilities}, nil
}

func (directory *Directory) Snapshot(ctx context.Context, target registry.ReleaseTarget) (registry.InstanceSnapshot, error) {
	if ctx == nil || target.Validate() != nil {
		return registry.InstanceSnapshot{}, registry.ErrInvalid
	}
	document, err := directory.loadDocument(ctx)
	if err != nil {
		return registry.InstanceSnapshot{}, err
	}

	var matched *contracts.RouterInstanceDirectoryTargetV1
	for index := range document.Targets {
		wire := &document.Targets[index]
		candidate, buildErr := releaseTarget(*wire)
		if buildErr != nil {
			return registry.InstanceSnapshot{}, registry.ErrInvalid
		}
		if candidate.Equal(target) {
			matched = wire
		}
	}
	if matched == nil {
		return registry.InstanceSnapshot{}, registry.ErrMissing
	}

	instances := make([]registry.Instance, 0, len(matched.Instances))
	for _, wire := range matched.Instances {
		instance, buildErr := instance(wire)
		if buildErr != nil {
			return registry.InstanceSnapshot{}, registry.ErrInvalid
		}
		instances = append(instances, instance)
	}
	revision, err := registry.NewRevision(registry.RevisionInput{SourceTokens: []string{document.Revision}})
	if err != nil {
		return registry.InstanceSnapshot{}, registry.ErrInvalid
	}
	state := registry.SnapshotStateEmpty
	if len(instances) > 0 {
		state = registry.SnapshotStatePopulated
	}
	result, err := registry.NewInstanceSnapshot(registry.InstanceSnapshotInput{Target: target, Revision: revision, State: state, Instances: instances})
	if err != nil {
		return registry.InstanceSnapshot{}, registry.ErrInvalid
	}
	return result, nil
}

// Check validates source availability and the complete current document while
// exposing no configuration value or source detail.
func (directory *Directory) Check(ctx context.Context) error {
	_, err := directory.loadDocument(ctx)
	return err
}

func (directory *Directory) loadDocument(ctx context.Context) (contracts.RouterInstanceDirectoryV1, error) {
	snapshot, err := directory.reader.Get(ctx, directory.key)
	if err != nil {
		return contracts.RouterInstanceDirectoryV1{}, mapSourceError(err)
	}
	if !snapshot.Present() {
		return contracts.RouterInstanceDirectoryV1{}, registry.ErrMissing
	}
	document, err := contracts.DecodeRouterInstanceDirectoryV1(snapshot.Content())
	if err != nil {
		return contracts.RouterInstanceDirectoryV1{}, registry.ErrInvalid
	}
	seen := make(map[string]struct{}, len(document.Targets))
	for _, wire := range document.Targets {
		target, targetErr := releaseTarget(wire)
		if targetErr != nil {
			return contracts.RouterInstanceDirectoryV1{}, registry.ErrInvalid
		}
		identity := target.ReleaseID()
		if _, duplicate := seen[identity]; duplicate {
			return contracts.RouterInstanceDirectoryV1{}, registry.ErrInvalid
		}
		seen[identity] = struct{}{}
		for _, wireInstance := range wire.Instances {
			if _, instanceErr := instance(wireInstance); instanceErr != nil {
				return contracts.RouterInstanceDirectoryV1{}, registry.ErrInvalid
			}
		}
	}
	return document, nil
}

func (directory *Directory) Observe(context.Context, registry.ReleaseTarget) (registry.InstanceObservation, error) {
	return registry.InstanceObservation{}, registry.ErrInvalid
}

func (directory *Directory) Capabilities() registry.Capabilities { return directory.capabilities }
func (directory *Directory) Close() error                        { return directory.reader.Close() }

func releaseTarget(wire contracts.RouterInstanceDirectoryTargetV1) (registry.ReleaseTarget, error) {
	return registry.NewReleaseTarget(registry.ReleaseTargetInput{
		AgentID: wire.AgentID, AgentCardVersion: wire.AgentCardVersion, ReleaseID: wire.ReleaseID,
		CardDigest: wire.CardDigest, CanonicalEndpoint: wire.CanonicalEndpoint, Audience: wire.Audience,
	})
}

func instance(wire contracts.RouterInstanceV1) (registry.Instance, error) {
	endpoints := make([]registry.NetworkEndpoint, 0, len(wire.Endpoints))
	for _, value := range wire.Endpoints {
		if !validAddress(registry.AddressType(value.AddressType), value.Address) || value.Protocol != string(registry.TransportProtocolTCP) {
			return registry.Instance{}, registry.ErrInvalid
		}
		endpoint, err := registry.NewNetworkEndpoint(registry.NetworkEndpointInput{
			AddressType: registry.AddressType(value.AddressType), Address: value.Address,
			PortName: value.PortName, Port: value.Port, Protocol: registry.TransportProtocol(value.Protocol),
		})
		if err != nil {
			return registry.Instance{}, err
		}
		endpoints = append(endpoints, endpoint)
	}
	return registry.NewInstance(registry.InstanceInput{
		ID: wire.InstanceID, Endpoints: endpoints, Ready: wire.Ready,
		Serving: wire.Serving, Terminating: wire.Terminating,
	})
}

func validAddress(addressType registry.AddressType, address string) bool {
	switch addressType {
	case registry.AddressTypeIPv4:
		parsed := net.ParseIP(address)
		return parsed != nil && parsed.To4() != nil && parsed.String() == address
	case registry.AddressTypeIPv6:
		parsed := net.ParseIP(address)
		return parsed != nil && parsed.To4() == nil && parsed.String() == address
	case registry.AddressTypeDNS:
		return len(address) <= 253 && dnsAddressPattern.MatchString(address)
	default:
		return false
	}
}

func mapSourceError(err error) error {
	switch {
	case errors.Is(err, configcenter.ErrMissing):
		return registry.ErrMissing
	case errors.Is(err, configcenter.ErrInvalid), errors.Is(err, configcenter.ErrUnsafeState), errors.Is(err, configcenter.ErrPayloadTooLarge):
		return registry.ErrInvalid
	case errors.Is(err, configcenter.ErrUnauthorized):
		return registry.ErrUnauthorized
	case errors.Is(err, configcenter.ErrCanceled):
		return registry.ErrCanceled
	case errors.Is(err, configcenter.ErrReaderClosed):
		return registry.ErrClosed
	default:
		return registry.ErrUnavailable
	}
}

var _ registry.InstanceDirectory = (*Directory)(nil)
