package routing

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strconv"

	"github.com/NeKiro-project/NeKiro/apps/a2a-router/internal/transport/a2a"
	"github.com/NeKiro-project/NeKiro/registry"
)

// SnapshotSelector chooses only when the exact Release has exactly one ready
// instance endpoint for the configured port. Multi-instance load balancing is
// intentionally left to a separately approved policy.
type SnapshotSelector struct {
	directory registry.InstanceDirectory
	portName  string
}

func NewSnapshotSelector(directory registry.InstanceDirectory, portName string) (*SnapshotSelector, error) {
	if directory == nil || portName == "" || !directory.Capabilities().Supports(registry.CapabilitySnapshot) {
		return nil, errors.New("snapshot selector dependencies are required")
	}
	return &SnapshotSelector{directory: directory, portName: portName}, nil
}

func (selector *SnapshotSelector) Select(ctx context.Context, target a2a.Target, _ a2a.ContextHeaders) (a2a.Target, error) {
	canonicalEndpoint, err := canonicalEndpoint(target.Endpoint)
	if err != nil {
		return a2a.Target{}, err
	}
	release, err := registry.NewReleaseTarget(registry.ReleaseTargetInput{
		AgentID: target.AgentID, AgentCardVersion: target.Version, ReleaseID: target.ReleaseID,
		CardDigest: target.CardDigest, CanonicalEndpoint: canonicalEndpoint, Audience: target.Audience,
	})
	if err != nil {
		return a2a.Target{}, err
	}
	snapshot, err := selector.directory.Snapshot(ctx, release)
	if err != nil {
		return a2a.Target{}, err
	}
	var selected *registry.NetworkEndpoint
	for _, instance := range snapshot.Instances() {
		if instance.State() != registry.InstanceStateReady {
			continue
		}
		for _, endpoint := range instance.Endpoints() {
			if endpoint.PortName() != selector.portName || endpoint.Protocol() != registry.TransportProtocolTCP {
				continue
			}
			if selected != nil {
				return a2a.Target{}, errors.New("multiple ready instance endpoints require an explicit selection policy")
			}
			copy := endpoint
			selected = &copy
		}
	}
	if selected == nil {
		return a2a.Target{}, errors.New("no ready instance endpoint")
	}
	endpoint, err := url.Parse(target.Endpoint)
	if err != nil {
		return a2a.Target{}, err
	}
	endpoint.Host = net.JoinHostPort(selected.Address(), strconv.Itoa(selected.Port()))
	selectedTarget := target
	selectedTarget.Endpoint = endpoint.String()
	return selectedTarget, nil
}

func canonicalEndpoint(raw string) (string, error) {
	endpoint, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if endpoint.Path == "" {
		endpoint.Path = "/"
	}
	return endpoint.String(), nil
}

var _ a2a.TargetSelector = (*SnapshotSelector)(nil)
