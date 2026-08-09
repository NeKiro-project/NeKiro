// Package nacos provides the explicit Nacos Config Center adapter.
package nacos

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	configcenter "github.com/NeKiro-project/NeKiro/config_center"
)

const ProviderID = configcenter.ProviderID("nacos")

const mappedDataIDPrefix = "nekiro.key.v1."

const (
	AuthNone        = "none"
	AuthAccessToken = "access_token"
)

type RequestExecutor interface {
	Do(*http.Request) (*http.Response, error)
}

type ReaderConfig struct {
	APIOrigin          string
	NamespaceID        string
	GroupName          string
	MaxPayloadBytes    int64
	WatchEnabled       bool
	LongPollTimeout    time.Duration
	SubscriptionBuffer int
	AuthMode           string
	AccessToken        string
	Executor           RequestExecutor
}

type Reader struct {
	origin             *url.URL
	namespaceID        string
	groupName          string
	maxPayloadBytes    int64
	watchEnabled       bool
	longPollTimeout    time.Duration
	subscriptionBuffer int
	authMode           string
	accessToken        string
	executor           RequestExecutor
	mu                 sync.Mutex
	closed             bool
	interrupted        bool
	subscriptions      map[*subscription]struct{}
}

func NewReader(config ReaderConfig) (*Reader, error) {
	origin, err := validateConfig(config)
	if err != nil {
		return nil, err
	}
	return &Reader{
		origin: origin, namespaceID: config.NamespaceID, groupName: config.GroupName,
		maxPayloadBytes: config.MaxPayloadBytes, authMode: config.AuthMode,
		watchEnabled: config.WatchEnabled, longPollTimeout: config.LongPollTimeout,
		subscriptionBuffer: config.SubscriptionBuffer, accessToken: config.AccessToken,
		executor: config.Executor, subscriptions: make(map[*subscription]struct{}),
	}, nil
}

func (reader *Reader) Get(ctx context.Context, key configcenter.Key) (configcenter.Snapshot, error) {
	if ctx == nil || !key.Valid() {
		return configcenter.Snapshot{}, readerError(configcenter.CodeInvalid, key, configcenter.OperationGet)
	}
	if err := ctx.Err(); err != nil {
		return configcenter.Snapshot{}, canceledError(key, configcenter.OperationGet, err)
	}
	reader.mu.Lock()
	closed := reader.closed
	interrupted := reader.interrupted
	reader.mu.Unlock()
	if closed {
		return configcenter.Snapshot{}, readerError(configcenter.CodeReaderClosed, key, configcenter.OperationGet)
	}
	if interrupted {
		return configcenter.Snapshot{}, watchInterruptedError(key, configcenter.OperationGet, "")
	}
	state, err := reader.readState(ctx, key, configcenter.OperationGet)
	if err != nil {
		return configcenter.Snapshot{}, err
	}
	if !state.present {
		snapshot, snapshotErr := configcenter.NewMissingSnapshot(key, configcenter.UnscopedRevision())
		if snapshotErr != nil {
			return configcenter.Snapshot{}, snapshotErr
		}
		return snapshot, readerError(configcenter.CodeMissing, key, configcenter.OperationGet)
	}
	return configcenter.NewPresentSnapshot(key, state.content, configcenter.UnscopedRevision())
}

type remoteState struct {
	present bool
	content []byte
}

func (reader *Reader) readState(ctx context.Context, key configcenter.Key, operation configcenter.Operation) (remoteState, error) {
	endpoint := *reader.origin
	endpoint.Path = strings.TrimSuffix(endpoint.Path, "/") + "/v1/cs/configs"
	query := endpoint.Query()
	query.Set("dataId", dataIDForKey(key))
	query.Set("group", reader.groupName)
	query.Set("tenant", reader.namespaceID)
	if reader.authMode == AuthAccessToken {
		query.Set("accessToken", reader.accessToken)
	}
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return remoteState{}, readerError(configcenter.CodeInvalid, key, operation)
	}
	response, err := reader.executor.Do(request)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return remoteState{}, canceledError(key, operation, ctxErr)
		}
		return remoteState{}, readerError(configcenter.CodeUnavailable, key, operation)
	}
	defer response.Body.Close()
	if err := ctx.Err(); err != nil {
		return remoteState{}, canceledError(key, operation, err)
	}
	if err := reader.lifecycleError(key, operation); err != nil {
		return remoteState{}, err
	}
	if response.StatusCode == http.StatusNotFound {
		return remoteState{}, nil
	}
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return remoteState{}, readerError(configcenter.CodeUnauthorized, key, operation)
	}
	if response.StatusCode != http.StatusOK {
		return remoteState{}, readerError(configcenter.CodeUnavailable, key, operation)
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, reader.maxPayloadBytes+1))
	if err != nil {
		return remoteState{}, readerError(configcenter.CodeUnavailable, key, operation)
	}
	if int64(len(content)) > reader.maxPayloadBytes {
		return remoteState{}, readerError(configcenter.CodePayloadTooLarge, key, operation)
	}
	if err := ctx.Err(); err != nil {
		return remoteState{}, canceledError(key, operation, err)
	}
	if err := reader.lifecycleError(key, operation); err != nil {
		return remoteState{}, err
	}
	return remoteState{present: true, content: bytes.Clone(content)}, nil
}

func (reader *Reader) lifecycleError(key configcenter.Key, operation configcenter.Operation) error {
	reader.mu.Lock()
	closed := reader.closed
	interrupted := reader.interrupted
	reader.mu.Unlock()
	if closed {
		return readerError(configcenter.CodeReaderClosed, key, operation)
	}
	if interrupted {
		return watchInterruptedError(key, operation, "")
	}
	return nil
}

// Observe returns an observation whose first long-poll compares the exact
// initial content digest. A transition between the initial GET and listener
// request is therefore reported by Nacos instead of being lost in a handoff
// gap.
func (reader *Reader) Observe(ctx context.Context, key configcenter.Key) (configcenter.Observation, error) {
	if ctx == nil || !key.Valid() {
		return configcenter.Observation{}, readerError(configcenter.CodeInvalid, key, configcenter.OperationObserve)
	}
	if err := ctx.Err(); err != nil {
		return configcenter.Observation{}, canceledError(key, configcenter.OperationObserve, err)
	}
	reader.mu.Lock()
	if reader.closed {
		reader.mu.Unlock()
		return configcenter.Observation{}, readerError(configcenter.CodeReaderClosed, key, configcenter.OperationObserve)
	}
	if reader.interrupted {
		reader.mu.Unlock()
		return configcenter.Observation{}, watchInterruptedError(key, configcenter.OperationObserve, "")
	}
	if !reader.watchEnabled {
		reader.mu.Unlock()
		return configcenter.Observation{}, readerError(configcenter.CodeUnsupported, key, configcenter.OperationObserve)
	}
	reader.mu.Unlock()

	state, err := reader.readState(ctx, key, configcenter.OperationObserve)
	if err != nil {
		return configcenter.Observation{}, err
	}
	revision := configcenter.NewObservationRevision()
	initial, err := snapshotForState(key, state, revision)
	if err != nil {
		return configcenter.Observation{}, err
	}
	subscription := newSubscription(reader, key, state, revision)

	reader.mu.Lock()
	if reader.closed {
		reader.mu.Unlock()
		subscription.cancel()
		return configcenter.Observation{}, readerError(configcenter.CodeReaderClosed, key, configcenter.OperationObserve)
	}
	if reader.interrupted {
		reader.mu.Unlock()
		subscription.cancel()
		return configcenter.Observation{}, watchInterruptedError(key, configcenter.OperationObserve, "")
	}
	reader.subscriptions[subscription] = struct{}{}
	reader.mu.Unlock()

	if err := ctx.Err(); err != nil {
		_ = subscription.Close()
		return configcenter.Observation{}, canceledError(key, configcenter.OperationObserve, err)
	}
	go subscription.watch()
	return configcenter.Observation{Initial: initial, Subscription: subscription}, nil
}

// dataIDForKey maps every strict provider-neutral key to one collision-free
// Nacos dataId. Existing keys that are already legal remain unchanged unless
// they occupy the reserved mapping prefix.
func dataIDForKey(key configcenter.Key) string {
	value := key.String()
	if ValidDataID(value) && !strings.HasPrefix(value, mappedDataIDPrefix) {
		return value
	}
	return mappedDataIDPrefix + base64.RawURLEncoding.EncodeToString([]byte(value))
}

// ValidDataID reports whether text can be used as an exact Nacos dataId.
func ValidDataID(value string) bool {
	if value == "" || len(value) > 255 {
		return false
	}
	for _, character := range value {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || strings.ContainsRune("._:-", character) {
			continue
		}
		return false
	}
	return true
}

func (reader *Reader) Close() error {
	reader.mu.Lock()
	if reader.closed {
		reader.mu.Unlock()
		return nil
	}
	reader.closed = true
	subscriptions := reader.takeSubscriptionsLocked()
	reader.mu.Unlock()
	for _, subscription := range subscriptions {
		subscription.terminate(readerError(configcenter.CodeReaderClosed, subscription.key, configcenter.OperationNext))
	}
	return nil
}

func (reader *Reader) takeSubscriptionsLocked() []*subscription {
	subscriptions := make([]*subscription, 0, len(reader.subscriptions))
	for subscription := range reader.subscriptions {
		subscriptions = append(subscriptions, subscription)
	}
	reader.subscriptions = make(map[*subscription]struct{})
	return subscriptions
}

func (reader *Reader) closeSubscription(subscription *subscription) {
	reader.mu.Lock()
	if _, ok := reader.subscriptions[subscription]; ok {
		delete(reader.subscriptions, subscription)
		subscription.terminate(readerError(configcenter.CodeSubscriptionClosed, subscription.key, configcenter.OperationNext))
	}
	reader.mu.Unlock()
}

func (reader *Reader) interruptAll(cause configcenter.Code) {
	reader.mu.Lock()
	if reader.closed || reader.interrupted {
		reader.mu.Unlock()
		return
	}
	reader.interrupted = true
	subscriptions := reader.takeSubscriptionsLocked()
	reader.mu.Unlock()
	for _, subscription := range subscriptions {
		subscription.terminate(watchInterruptedError(subscription.key, configcenter.OperationNext, cause))
	}
}

func snapshotForState(key configcenter.Key, state remoteState, revision configcenter.Revision) (configcenter.Snapshot, error) {
	if !state.present {
		return configcenter.NewMissingSnapshot(key, revision)
	}
	return configcenter.NewPresentSnapshot(key, state.content, revision)
}

func validateConfig(config ReaderConfig) (*url.URL, error) {
	origin, err := url.Parse(config.APIOrigin)
	if err != nil || origin.Scheme != "http" && origin.Scheme != "https" || origin.Host == "" || origin.User != nil ||
		origin.Path != "/nacos" || origin.RawPath != "" || origin.RawQuery != "" || origin.ForceQuery || origin.Fragment != "" ||
		!validText(config.NamespaceID) || !validText(config.GroupName) || config.MaxPayloadBytes <= 0 || config.Executor == nil {
		return nil, readerError(configcenter.CodeInvalid, configcenter.Key{}, configcenter.OperationRead)
	}
	if config.AuthMode != AuthNone && config.AuthMode != AuthAccessToken ||
		config.AuthMode == AuthNone && config.AccessToken != "" ||
		config.AuthMode == AuthAccessToken && !validText(config.AccessToken) {
		return nil, readerError(configcenter.CodeInvalid, configcenter.Key{}, configcenter.OperationRead)
	}
	if config.WatchEnabled {
		if config.LongPollTimeout < 100*time.Millisecond || config.LongPollTimeout > 5*time.Minute || config.SubscriptionBuffer < 1 || config.SubscriptionBuffer > 1024 {
			return nil, readerError(configcenter.CodeInvalid, configcenter.Key{}, configcenter.OperationRead)
		}
	} else if config.LongPollTimeout != 0 || config.SubscriptionBuffer != 0 {
		return nil, readerError(configcenter.CodeInvalid, configcenter.Key{}, configcenter.OperationRead)
	}
	return origin, nil
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

func readerError(code configcenter.Code, key configcenter.Key, operation configcenter.Operation) error {
	return configcenter.NewError(code, configcenter.ErrorDetails{Provider: ProviderID, Key: key, Operation: operation})
}

func canceledError(key configcenter.Key, operation configcenter.Operation, err error) error {
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		err = context.Canceled
	}
	return configcenter.NewCanceledError(configcenter.ErrorDetails{Provider: ProviderID, Key: key, Operation: operation}, err)
}

func watchInterruptedError(key configcenter.Key, operation configcenter.Operation, cause configcenter.Code) error {
	return configcenter.NewError(configcenter.CodeWatchInterrupted, configcenter.ErrorDetails{
		Provider: ProviderID, Key: key, Operation: operation, CauseKind: watchCause(cause),
	})
}

func watchCause(code configcenter.Code) configcenter.Code {
	switch code {
	case configcenter.CodeUnsafeState, configcenter.CodeUnauthorized, configcenter.CodeUnavailable, configcenter.CodePayloadTooLarge:
		return code
	default:
		return ""
	}
}

var _ configcenter.DynamicConfiguration = (*Reader)(nil)
