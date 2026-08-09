// Package nacos implements Nacos instance discovery.
package nacos

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"

	"github.com/NeKiro-project/NeKiro/registry"
)

const (
	AuthNone        = "none"
	AuthAccessToken = "access_token"
)

const maxInstances = 4096

const instanceIDMetadataKey = "nekiro.instanceId"

type RequestExecutor interface {
	Do(*http.Request) (*http.Response, error)
}

type BindingInput struct {
	Target      registry.ReleaseTarget
	ServiceName string
	GroupName   string
	ClusterName string
}

type Binding struct {
	target      registry.ReleaseTarget
	serviceName string
	groupName   string
	clusterName string
}

func NewBinding(input BindingInput) (Binding, error) {
	if input.Target.Validate() != nil || !validText(input.ServiceName) || !validText(input.GroupName) || !validText(input.ClusterName) {
		return Binding{}, registry.ErrInvalid
	}
	return Binding{target: input.Target, serviceName: input.ServiceName, groupName: input.GroupName, clusterName: input.ClusterName}, nil
}

func (binding Binding) Target() registry.ReleaseTarget { return binding.target }
func (binding Binding) ServiceName() string            { return binding.serviceName }
func (binding Binding) GroupName() string              { return binding.groupName }
func (binding Binding) ClusterName() string            { return binding.clusterName }

type BindingSource interface {
	Binding(context.Context, registry.ReleaseTarget) (Binding, error)
}

type DirectoryConfig struct {
	APIOrigin        string
	NamespaceID      string
	PortName         string
	MaxResponseBytes int64
	AuthMode         string
	AccessToken      string
	Executor         RequestExecutor
	Bindings         BindingSource
	Subscriber       NamingSubscriptionExecutor
	PendingChanges   int
}

type Directory struct {
	origin           *url.URL
	namespaceID      string
	portName         string
	maxResponseBytes int64
	authMode         string
	accessToken      string
	executor         RequestExecutor
	bindings         BindingSource
	subscriber       NamingSubscriptionExecutor
	pendingChanges   int
	capabilities     registry.Capabilities
	mu               sync.Mutex
	closed           bool
	sessions         map[*observationSession]struct{}
}

func NewDirectory(config DirectoryConfig) (*Directory, error) {
	origin, err := validateConfig(config)
	if err != nil {
		return nil, err
	}
	capabilityValues := []registry.Capability{registry.CapabilitySnapshot}
	if config.Subscriber != nil {
		capabilityValues = append(capabilityValues, registry.CapabilityObserve)
	}
	capabilities, err := registry.NewCapabilities(capabilityValues...)
	if err != nil {
		return nil, err
	}
	return &Directory{
		origin: origin, namespaceID: config.NamespaceID, portName: config.PortName,
		maxResponseBytes: config.MaxResponseBytes, authMode: config.AuthMode,
		accessToken: config.AccessToken, executor: config.Executor, bindings: config.Bindings,
		subscriber: config.Subscriber, pendingChanges: config.PendingChanges,
		capabilities: capabilities, sessions: make(map[*observationSession]struct{}),
	}, nil
}

func (directory *Directory) Snapshot(ctx context.Context, target registry.ReleaseTarget) (registry.InstanceSnapshot, error) {
	if ctx == nil || target.Validate() != nil {
		return registry.InstanceSnapshot{}, registry.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return registry.InstanceSnapshot{}, canceled(err)
	}
	directory.mu.Lock()
	closed := directory.closed
	directory.mu.Unlock()
	if closed {
		return registry.InstanceSnapshot{}, registry.ErrClosed
	}
	binding, err := directory.bindings.Binding(ctx, target)
	if err != nil {
		return registry.InstanceSnapshot{}, err
	}
	if !binding.target.Equal(target) {
		return registry.InstanceSnapshot{}, registry.ErrInvalid
	}
	endpoint := *directory.origin
	endpoint.Path = strings.TrimSuffix(endpoint.Path, "/") + "/v1/ns/instance/list"
	query := endpoint.Query()
	query.Set("serviceName", binding.serviceName)
	query.Set("groupName", binding.groupName)
	query.Set("clusters", binding.clusterName)
	query.Set("namespaceId", directory.namespaceID)
	query.Set("healthyOnly", "false")
	if directory.authMode == AuthAccessToken {
		query.Set("accessToken", directory.accessToken)
	}
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return registry.InstanceSnapshot{}, registry.ErrInvalid
	}
	response, err := directory.executor.Do(request)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return registry.InstanceSnapshot{}, canceled(ctxErr)
		}
		return registry.InstanceSnapshot{}, registry.ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return registry.InstanceSnapshot{}, registry.ErrUnauthorized
	}
	if response.StatusCode == http.StatusNotFound {
		return registry.InstanceSnapshot{}, registry.ErrMissing
	}
	if response.StatusCode != http.StatusOK {
		return registry.InstanceSnapshot{}, registry.ErrUnavailable
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, directory.maxResponseBytes+1))
	if err != nil {
		return registry.InstanceSnapshot{}, registry.ErrUnavailable
	}
	if int64(len(data)) > directory.maxResponseBytes {
		return registry.InstanceSnapshot{}, registry.ErrInvalid
	}
	return snapshotFromPayload(data, binding, directory.portName, 0)
}

func (directory *Directory) Observe(ctx context.Context, target registry.ReleaseTarget) (registry.InstanceObservation, error) {
	if ctx == nil || target.Validate() != nil {
		return registry.InstanceObservation{}, registry.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return registry.InstanceObservation{}, canceled(err)
	}
	directory.mu.Lock()
	closed := directory.closed
	subscriber := directory.subscriber
	directory.mu.Unlock()
	if closed {
		return registry.InstanceObservation{}, registry.ErrClosed
	}
	if subscriber == nil {
		return registry.InstanceObservation{}, registry.ErrInvalid
	}
	binding, err := directory.bindings.Binding(ctx, target)
	if err != nil {
		return registry.InstanceObservation{}, err
	}
	if !binding.target.Equal(target) {
		return registry.InstanceObservation{}, registry.ErrInvalid
	}
	source, err := subscriber.Subscribe(ctx, newNamingSubscribeRequest(directory.namespaceID, binding))
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return registry.InstanceObservation{}, canceled(ctxErr)
		}
		return registry.InstanceObservation{}, safeSubscribeError(err)
	}
	initialPayload := source.InitialPayload()
	if int64(len(initialPayload)) > directory.maxResponseBytes {
		_ = source.Close()
		return registry.InstanceObservation{}, registry.ErrInvalid
	}
	initial, err := snapshotFromPayload(initialPayload, binding, directory.portName, 0)
	if err != nil {
		_ = source.Close()
		return registry.InstanceObservation{}, err
	}
	watch, publisher, err := registry.NewInstanceWatch(directory.pendingChanges)
	if err != nil {
		_ = source.Close()
		return registry.InstanceObservation{}, err
	}
	session := newObservationSession(directory, target, binding, initial, source, watch, publisher)
	if !directory.registerSession(session) {
		_ = source.Close()
		publisher.Terminate(registry.ErrClosed)
		return registry.InstanceObservation{}, registry.ErrClosed
	}
	if err := ctx.Err(); err != nil {
		_ = session.close()
		return registry.InstanceObservation{}, canceled(err)
	}
	go session.run()
	return registry.NewInstanceObservation(initial, session.watch)
}

func (directory *Directory) Capabilities() registry.Capabilities { return directory.capabilities }

func (directory *Directory) Close() error {
	directory.mu.Lock()
	if directory.closed {
		directory.mu.Unlock()
		return nil
	}
	directory.closed = true
	sessions := make([]*observationSession, 0, len(directory.sessions))
	for session := range directory.sessions {
		sessions = append(sessions, session)
	}
	directory.sessions = make(map[*observationSession]struct{})
	directory.mu.Unlock()
	for _, session := range sessions {
		_ = session.close()
	}
	return nil
}

func (directory *Directory) registerSession(session *observationSession) bool {
	directory.mu.Lock()
	defer directory.mu.Unlock()
	if directory.closed {
		return false
	}
	directory.sessions[session] = struct{}{}
	return true
}

func (directory *Directory) unregisterSession(session *observationSession) {
	directory.mu.Lock()
	delete(directory.sessions, session)
	directory.mu.Unlock()
}

func snapshotFromPayload(data []byte, binding Binding, portName string, localOrder uint64) (registry.InstanceSnapshot, error) {
	instances, err := decodeInstances(data, binding, portName)
	if err != nil {
		return registry.InstanceSnapshot{}, err
	}
	digest := sha256.Sum256(data)
	revision, err := registry.NewRevision(registry.RevisionInput{SourceTokens: []string{hex.EncodeToString(digest[:])}, LocalOrder: localOrder})
	if err != nil {
		return registry.InstanceSnapshot{}, registry.ErrInvalid
	}
	state := registry.SnapshotStateEmpty
	if len(instances) != 0 {
		state = registry.SnapshotStatePopulated
	}
	return registry.NewInstanceSnapshot(registry.InstanceSnapshotInput{Target: binding.target, Revision: revision, State: state, Instances: instances})
}

type listResponse struct {
	Hosts []host `json:"hosts"`
}

type host struct {
	IP          string            `json:"ip"`
	Port        int               `json:"port"`
	Healthy     bool              `json:"healthy"`
	Enabled     bool              `json:"enabled"`
	Ephemeral   bool              `json:"ephemeral"`
	ClusterName string            `json:"clusterName"`
	Metadata    map[string]string `json:"metadata"`
}

func decodeInstances(data []byte, binding Binding, portName string) ([]registry.Instance, error) {
	var response listResponse
	if err := json.Unmarshal(data, &response); err != nil || response.Hosts == nil || len(response.Hosts) > maxInstances {
		return nil, registry.ErrInvalid
	}
	instances := make([]registry.Instance, 0, len(response.Hosts))
	for _, value := range response.Hosts {
		parsed := net.ParseIP(value.IP)
		instanceID, exists := value.Metadata[instanceIDMetadataKey]
		if parsed == nil || parsed.String() != value.IP || !value.Ephemeral || value.ClusterName != binding.clusterName || !exists {
			return nil, registry.ErrInvalid
		}
		addressType := registry.AddressTypeIPv6
		if parsed.To4() != nil {
			addressType = registry.AddressTypeIPv4
		}
		endpoint, err := registry.NewNetworkEndpoint(registry.NetworkEndpointInput{AddressType: addressType, Address: value.IP, PortName: portName, Port: value.Port, Protocol: registry.TransportProtocolTCP})
		if err != nil {
			return nil, registry.ErrInvalid
		}
		ready := value.Healthy && value.Enabled
		instance, err := registry.NewInstance(registry.InstanceInput{ID: instanceID, Endpoints: []registry.NetworkEndpoint{endpoint}, Ready: ready, Serving: ready, Terminating: false})
		if err != nil {
			return nil, registry.ErrInvalid
		}
		instances = append(instances, instance)
	}
	return instances, nil
}

func validateConfig(config DirectoryConfig) (*url.URL, error) {
	origin, err := url.Parse(config.APIOrigin)
	if err != nil || origin.Scheme != "http" && origin.Scheme != "https" || origin.Host == "" || origin.User != nil || origin.Path != "/nacos" || origin.RawQuery != "" || origin.Fragment != "" ||
		!validText(config.NamespaceID) || !validText(config.PortName) || config.MaxResponseBytes <= 0 || config.Executor == nil || config.Bindings == nil {
		return nil, registry.ErrInvalid
	}
	if config.AuthMode != AuthNone && config.AuthMode != AuthAccessToken || config.AuthMode == AuthNone && config.AccessToken != "" || config.AuthMode == AuthAccessToken && !validText(config.AccessToken) {
		return nil, registry.ErrInvalid
	}
	if isNilInterface(config.Subscriber) {
		if config.PendingChanges != 0 {
			return nil, registry.ErrInvalid
		}
	} else if config.PendingChanges <= 0 || config.Subscriber.Guarantees().Validate() != nil {
		return nil, registry.ErrInvalid
	}
	return origin, nil
}

func isNilInterface(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}

func validText(value string) bool {
	if value == "" || len(value) > 256 || strings.TrimSpace(value) != value {
		return false
	}
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return false
		}
	}
	return true
}

func canceled(err error) error {
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		err = context.Canceled
	}
	return fmt.Errorf("%w: %w", registry.ErrCanceled, err)
}

var _ registry.InstanceDirectory = (*Directory)(nil)
