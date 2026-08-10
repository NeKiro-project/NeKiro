package nacos

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/NeKiro-project/NeKiro/registry"
)

const registrationResponseLimit = 256

const (
	heartbeatIntervalMetadataKey = "preserved.heart.beat.interval"
	heartbeatTimeoutMetadataKey  = "preserved.heart.beat.timeout"
	ipDeleteTimeoutMetadataKey   = "preserved.ip.delete.timeout"
)

type RegistrarConfig struct {
	APIOrigin         string
	NamespaceID       string
	Binding           Binding
	PortName          string
	Weight            float64
	HeartbeatInterval time.Duration
	HeartbeatTimeout  time.Duration
	IPDeleteTimeout   time.Duration
	AuthMode          string
	AccessToken       string
	Executor          RequestExecutor
}

type Registrar struct {
	origin            *url.URL
	namespaceID       string
	binding           Binding
	portName          string
	weight            float64
	heartbeatInterval time.Duration
	heartbeatTimeout  time.Duration
	ipDeleteTimeout   time.Duration
	authMode          string
	accessToken       string
	executor          RequestExecutor
	capabilities      registry.Capabilities

	mu          sync.Mutex
	closed      bool
	registering bool
	sessions    map[*registrationSession]struct{}
}

type registrationSession struct {
	registrar    *Registrar
	registration registry.Registration
	endpoint     registry.NetworkEndpoint
	lease        *registry.Lease
	stop         chan struct{}
	stopped      chan struct{}
	stopOnce     sync.Once
	opMu         sync.Mutex
	closeOnce    sync.Once
	closeErr     error
}

func NewRegistrar(config RegistrarConfig) (*Registrar, error) {
	origin, err := url.Parse(config.APIOrigin)
	if err != nil || origin.Scheme != "http" && origin.Scheme != "https" || origin.Host == "" || origin.User != nil || origin.Path != "/nacos" || origin.RawQuery != "" || origin.Fragment != "" ||
		!validText(config.NamespaceID) || config.Binding.target.Validate() != nil || !validText(config.PortName) ||
		math.IsNaN(config.Weight) || math.IsInf(config.Weight, 0) || config.Weight <= 0 || config.Weight > 10000 ||
		config.HeartbeatInterval < time.Second || config.HeartbeatInterval > time.Minute ||
		config.HeartbeatTimeout <= config.HeartbeatInterval || config.HeartbeatTimeout > 5*time.Minute ||
		config.IPDeleteTimeout <= config.HeartbeatTimeout || config.IPDeleteTimeout > 10*time.Minute || isNilInterface(config.Executor) {
		return nil, registry.ErrInvalid
	}
	if config.AuthMode != AuthNone && config.AuthMode != AuthAccessToken || config.AuthMode == AuthNone && config.AccessToken != "" || config.AuthMode == AuthAccessToken && !validText(config.AccessToken) {
		return nil, registry.ErrInvalid
	}
	capabilities, _ := registry.NewCapabilities(registry.CapabilityRegistration, registry.CapabilityDeregistration, registry.CapabilityLease, registry.CapabilityHeartbeat)
	return &Registrar{
		origin: origin, namespaceID: config.NamespaceID, binding: config.Binding, portName: config.PortName, weight: config.Weight,
		heartbeatInterval: config.HeartbeatInterval, heartbeatTimeout: config.HeartbeatTimeout, ipDeleteTimeout: config.IPDeleteTimeout,
		authMode: config.AuthMode, accessToken: config.AccessToken,
		executor: config.Executor, capabilities: capabilities, sessions: make(map[*registrationSession]struct{}),
	}, nil
}

func (registrar *Registrar) Register(ctx context.Context, registration registry.Registration) (registry.InstanceLease, error) {
	if ctx == nil || registration.Validate() != nil || !registration.Target().Equal(registrar.binding.target) || registration.Instance().State() != registry.InstanceStateReady {
		return nil, registry.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return nil, canceled(err)
	}
	endpoint, err := registrationEndpoint(registration.Instance(), registrar.portName)
	if err != nil {
		return nil, err
	}
	registrar.mu.Lock()
	if registrar.closed {
		registrar.mu.Unlock()
		return nil, registry.ErrClosed
	}
	if registrar.registering || len(registrar.sessions) != 0 {
		registrar.mu.Unlock()
		return nil, registry.ErrInvalid
	}
	registrar.registering = true
	registrar.mu.Unlock()
	session := &registrationSession{registrar: registrar, registration: registration, endpoint: endpoint, stop: make(chan struct{}), stopped: make(chan struct{})}
	lease, _ := registry.NewLease(session.close)
	session.lease = lease
	if err := session.execute(ctx, http.MethodPost, false); err != nil {
		registrar.mu.Lock()
		registrar.registering = false
		registrar.mu.Unlock()
		return nil, err
	}
	registrar.mu.Lock()
	registrar.registering = false
	if registrar.closed {
		registrar.mu.Unlock()
		cleanupContext, cancelCleanup := context.WithTimeout(context.Background(), registrar.heartbeatInterval)
		cleanupErr := session.execute(cleanupContext, http.MethodDelete, false)
		cancelCleanup()
		lease.Terminate(registry.ErrClosed)
		return nil, errors.Join(registry.ErrClosed, cleanupErr)
	}
	registrar.sessions[session] = struct{}{}
	go session.run()
	registrar.mu.Unlock()
	return lease, nil
}

func (registrar *Registrar) Capabilities() registry.Capabilities { return registrar.capabilities }

func (registrar *Registrar) Close() error {
	registrar.mu.Lock()
	if registrar.closed {
		registrar.mu.Unlock()
		return nil
	}
	registrar.closed = true
	sessions := make([]*registrationSession, 0, len(registrar.sessions))
	for session := range registrar.sessions {
		sessions = append(sessions, session)
	}
	registrar.mu.Unlock()
	var result error
	for _, session := range sessions {
		ctx, cancel := context.WithTimeout(context.Background(), registrar.heartbeatInterval)
		result = errors.Join(result, session.close(ctx))
		cancel()
	}
	return result
}

func (registrar *Registrar) remove(session *registrationSession) {
	registrar.mu.Lock()
	delete(registrar.sessions, session)
	registrar.mu.Unlock()
}

func (session *registrationSession) run() {
	defer close(session.stopped)
	ticker := time.NewTicker(session.registrar.heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-session.stop:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), session.registrar.heartbeatInterval)
			err := session.execute(ctx, http.MethodPut, true)
			cancel()
			if err != nil {
				session.lease.Terminate(err)
				session.stopOnce.Do(func() { close(session.stop) })
				return
			}
		}
	}
}

func (session *registrationSession) close(ctx context.Context) error {
	if ctx == nil {
		return registry.ErrInvalid
	}
	session.closeOnce.Do(func() {
		session.stopOnce.Do(func() { close(session.stop) })
		select {
		case <-session.stopped:
		case <-ctx.Done():
			session.closeErr = canceled(ctx.Err())
			session.lease.Terminate(session.closeErr)
			session.registrar.remove(session)
			return
		}
		session.closeErr = session.execute(ctx, http.MethodDelete, false)
		if session.closeErr == nil {
			session.lease.Terminate(registry.ErrClosed)
		} else {
			session.lease.Terminate(session.closeErr)
		}
		session.registrar.remove(session)
	})
	return session.closeErr
}

func (session *registrationSession) execute(ctx context.Context, method string, heartbeat bool) error {
	session.opMu.Lock()
	defer session.opMu.Unlock()
	endpoint := *session.registrar.origin
	endpoint.Path = strings.TrimSuffix(endpoint.Path, "/") + "/v1/ns/instance"
	if heartbeat {
		endpoint.Path += "/beat"
	}
	query := endpoint.Query()
	query.Set("namespaceId", session.registrar.namespaceID)
	groupedServiceName := session.registrar.binding.groupName + "@@" + session.registrar.binding.serviceName
	query.Set("serviceName", groupedServiceName)
	query.Set("groupName", session.registrar.binding.groupName)
	query.Set("clusterName", session.registrar.binding.clusterName)
	query.Set("ip", session.endpoint.Address())
	query.Set("port", strconv.Itoa(session.endpoint.Port()))
	if session.registrar.authMode == AuthAccessToken {
		query.Set("accessToken", session.registrar.accessToken)
	}
	instance := session.registration.Instance()
	metadataValues := instance.Metadata()
	if metadataValues == nil {
		metadataValues = make(map[string]string)
	}
	for key := range metadataValues {
		if strings.HasPrefix(key, "preserved.") {
			return registry.ErrInvalid
		}
	}
	if identity, exists := metadataValues[instanceIDMetadataKey]; exists && identity != instance.ID() {
		return registry.ErrInvalid
	}
	metadataValues[instanceIDMetadataKey] = instance.ID()
	metadataValues[heartbeatIntervalMetadataKey] = strconv.FormatInt(session.registrar.heartbeatInterval.Milliseconds(), 10)
	metadataValues[heartbeatTimeoutMetadataKey] = strconv.FormatInt(session.registrar.heartbeatTimeout.Milliseconds(), 10)
	metadataValues[ipDeleteTimeoutMetadataKey] = strconv.FormatInt(session.registrar.ipDeleteTimeout.Milliseconds(), 10)
	metadata, _ := json.Marshal(metadataValues)
	query.Set("metadata", string(metadata))
	query.Set("ephemeral", "true")
	query.Set("enable", "true")
	query.Set("healthy", "true")
	query.Set("weight", strconv.FormatFloat(session.registrar.weight, 'f', -1, 64))
	if heartbeat {
		beat, _ := json.Marshal(map[string]any{
			"serviceName": groupedServiceName,
			"cluster":     session.registrar.binding.clusterName, "ip": session.endpoint.Address(), "port": session.endpoint.Port(),
			"metadata": metadataValues, "ephemeral": true, "scheduled": false, "weight": session.registrar.weight,
		})
		query.Set("beat", string(beat))
	}
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), bytes.NewReader(nil))
	if err != nil {
		return registry.ErrInvalid
	}
	response, err := session.registrar.executor.Do(request)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return canceled(ctxErr)
		}
		return registry.ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return registry.ErrUnauthorized
	}
	if response.StatusCode == http.StatusTooManyRequests || response.StatusCode < 200 || response.StatusCode >= 300 {
		return registry.ErrUnavailable
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, registrationResponseLimit+1))
	if err != nil {
		return registry.ErrUnavailable
	}
	if len(data) > registrationResponseLimit {
		return registry.ErrInvalid
	}
	if heartbeat {
		var response struct {
			ClientBeatInterval int64 `json:"clientBeatInterval"`
		}
		if json.Unmarshal(data, &response) != nil || response.ClientBeatInterval != session.registrar.heartbeatInterval.Milliseconds() {
			return registry.ErrInvalid
		}
	} else if strings.TrimSpace(string(data)) != "ok" {
		return registry.ErrInvalid
	}
	return nil
}

func registrationEndpoint(instance registry.Instance, portName string) (registry.NetworkEndpoint, error) {
	var selected registry.NetworkEndpoint
	found := false
	for _, endpoint := range instance.Endpoints() {
		if endpoint.PortName() != portName || endpoint.Protocol() != registry.TransportProtocolTCP || endpoint.AddressType() != registry.AddressTypeIPv4 && endpoint.AddressType() != registry.AddressTypeIPv6 {
			continue
		}
		if found {
			return registry.NetworkEndpoint{}, registry.ErrInvalid
		}
		selected = endpoint
		found = true
	}
	if !found {
		return registry.NetworkEndpoint{}, registry.ErrInvalid
	}
	return selected, nil
}

var _ registry.InstanceRegistrar = (*Registrar)(nil)
