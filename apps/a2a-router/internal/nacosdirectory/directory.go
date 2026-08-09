package nacosdirectory

import (
	"context"
	"errors"

	configcenter "github.com/NeKiro-project/NeKiro/config_center"
	"github.com/NeKiro-project/NeKiro/contracts"
	"github.com/NeKiro-project/NeKiro/registry"
	registrynacos "github.com/NeKiro-project/NeKiro/registry/nacos"
)

type Reader interface {
	Get(context.Context, configcenter.Key) (configcenter.Snapshot, error)
	Close() error
}

type Directory struct {
	reader   Reader
	key      configcenter.Key
	provider *registrynacos.Directory
}

func New(reader Reader, key configcenter.Key, providerConfig registrynacos.DirectoryConfig) (*Directory, error) {
	if reader == nil || !key.Valid() || providerConfig.Bindings != nil {
		return nil, registry.ErrInvalid
	}
	directory := &Directory{reader: reader, key: key}
	providerConfig.Bindings = directory
	provider, err := registrynacos.NewDirectory(providerConfig)
	if err != nil {
		return nil, err
	}
	directory.provider = provider
	return directory, nil
}

func (directory *Directory) Snapshot(ctx context.Context, target registry.ReleaseTarget) (registry.InstanceSnapshot, error) {
	return directory.provider.Snapshot(ctx, target)
}

func (directory *Directory) Binding(ctx context.Context, target registry.ReleaseTarget) (registrynacos.Binding, error) {
	document, err := directory.loadDocument(ctx)
	if err != nil {
		return registrynacos.Binding{}, err
	}
	for _, wire := range document.Targets {
		candidate, err := releaseTarget(wire)
		if err != nil {
			return registrynacos.Binding{}, registry.ErrInvalid
		}
		if candidate.Equal(target) {
			return registrynacos.NewBinding(registrynacos.BindingInput{Target: candidate, ServiceName: wire.ServiceName, GroupName: wire.GroupName, ClusterName: wire.ClusterName})
		}
	}
	return registrynacos.Binding{}, registry.ErrMissing
}

func (directory *Directory) Check(ctx context.Context) error {
	_, err := directory.loadDocument(ctx)
	return err
}

func (directory *Directory) loadDocument(ctx context.Context) (contracts.RouterNacosInstanceBindingsV1, error) {
	snapshot, err := directory.reader.Get(ctx, directory.key)
	if err != nil {
		return contracts.RouterNacosInstanceBindingsV1{}, mapSourceError(err)
	}
	if !snapshot.Present() {
		return contracts.RouterNacosInstanceBindingsV1{}, registry.ErrMissing
	}
	document, err := contracts.DecodeRouterNacosInstanceBindingsV1(snapshot.Content())
	if err != nil {
		return contracts.RouterNacosInstanceBindingsV1{}, registry.ErrInvalid
	}
	seen := make(map[string]struct{}, len(document.Targets))
	for _, wire := range document.Targets {
		target, err := releaseTarget(wire)
		if err != nil {
			return contracts.RouterNacosInstanceBindingsV1{}, registry.ErrInvalid
		}
		if _, duplicate := seen[target.ReleaseID()]; duplicate {
			return contracts.RouterNacosInstanceBindingsV1{}, registry.ErrInvalid
		}
		seen[target.ReleaseID()] = struct{}{}
		if _, err := registrynacos.NewBinding(registrynacos.BindingInput{Target: target, ServiceName: wire.ServiceName, GroupName: wire.GroupName, ClusterName: wire.ClusterName}); err != nil {
			return contracts.RouterNacosInstanceBindingsV1{}, registry.ErrInvalid
		}
	}
	return document, nil
}

func (directory *Directory) Observe(ctx context.Context, target registry.ReleaseTarget) (registry.InstanceObservation, error) {
	return directory.provider.Observe(ctx, target)
}

func (directory *Directory) Capabilities() registry.Capabilities {
	return directory.provider.Capabilities()
}

func (directory *Directory) Close() error {
	return errors.Join(directory.provider.Close(), directory.reader.Close())
}

func releaseTarget(wire contracts.RouterNacosInstanceBindingTargetV1) (registry.ReleaseTarget, error) {
	return registry.NewReleaseTarget(registry.ReleaseTargetInput{
		AgentID: wire.AgentID, AgentCardVersion: wire.AgentCardVersion, ReleaseID: wire.ReleaseID,
		CardDigest: wire.CardDigest, CanonicalEndpoint: wire.CanonicalEndpoint, Audience: wire.Audience,
	})
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
var _ registrynacos.BindingSource = (*Directory)(nil)
