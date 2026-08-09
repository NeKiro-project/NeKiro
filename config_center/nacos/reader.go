// Package nacos provides the explicit snapshot-only Nacos Config Center adapter.
package nacos

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	configcenter "github.com/NeKiro-project/NeKiro/config_center"
)

const ProviderID = configcenter.ProviderID("nacos")

const (
	AuthNone        = "none"
	AuthAccessToken = "access_token"
)

type RequestExecutor interface {
	Do(*http.Request) (*http.Response, error)
}

type ReaderConfig struct {
	APIOrigin       string
	NamespaceID     string
	GroupName       string
	MaxPayloadBytes int64
	AuthMode        string
	AccessToken     string
	Executor        RequestExecutor
}

type Reader struct {
	origin          *url.URL
	namespaceID     string
	groupName       string
	maxPayloadBytes int64
	authMode        string
	accessToken     string
	executor        RequestExecutor
	mu              sync.Mutex
	closed          bool
}

func NewReader(config ReaderConfig) (*Reader, error) {
	origin, err := validateConfig(config)
	if err != nil {
		return nil, err
	}
	return &Reader{
		origin: origin, namespaceID: config.NamespaceID, groupName: config.GroupName,
		maxPayloadBytes: config.MaxPayloadBytes, authMode: config.AuthMode,
		accessToken: config.AccessToken, executor: config.Executor,
	}, nil
}

func (reader *Reader) Get(ctx context.Context, key configcenter.Key) (configcenter.Snapshot, error) {
	if ctx == nil || !key.Valid() || !ValidDataID(key.String()) {
		return configcenter.Snapshot{}, readerError(configcenter.CodeInvalid, key, configcenter.OperationGet)
	}
	if err := ctx.Err(); err != nil {
		return configcenter.Snapshot{}, canceledError(key, configcenter.OperationGet, err)
	}
	reader.mu.Lock()
	closed := reader.closed
	reader.mu.Unlock()
	if closed {
		return configcenter.Snapshot{}, readerError(configcenter.CodeReaderClosed, key, configcenter.OperationGet)
	}

	endpoint := *reader.origin
	endpoint.Path = strings.TrimSuffix(endpoint.Path, "/") + "/v1/cs/configs"
	query := endpoint.Query()
	query.Set("dataId", key.String())
	query.Set("group", reader.groupName)
	query.Set("tenant", reader.namespaceID)
	if reader.authMode == AuthAccessToken {
		query.Set("accessToken", reader.accessToken)
	}
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return configcenter.Snapshot{}, readerError(configcenter.CodeInvalid, key, configcenter.OperationGet)
	}
	response, err := reader.executor.Do(request)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return configcenter.Snapshot{}, canceledError(key, configcenter.OperationGet, ctxErr)
		}
		return configcenter.Snapshot{}, readerError(configcenter.CodeUnavailable, key, configcenter.OperationGet)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		snapshot, snapshotErr := configcenter.NewMissingSnapshot(key, configcenter.UnscopedRevision())
		if snapshotErr != nil {
			return configcenter.Snapshot{}, snapshotErr
		}
		return snapshot, readerError(configcenter.CodeMissing, key, configcenter.OperationGet)
	}
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return configcenter.Snapshot{}, readerError(configcenter.CodeUnauthorized, key, configcenter.OperationGet)
	}
	if response.StatusCode != http.StatusOK {
		return configcenter.Snapshot{}, readerError(configcenter.CodeUnavailable, key, configcenter.OperationGet)
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, reader.maxPayloadBytes+1))
	if err != nil {
		return configcenter.Snapshot{}, readerError(configcenter.CodeUnavailable, key, configcenter.OperationGet)
	}
	if int64(len(content)) > reader.maxPayloadBytes {
		return configcenter.Snapshot{}, readerError(configcenter.CodePayloadTooLarge, key, configcenter.OperationGet)
	}
	if err := ctx.Err(); err != nil {
		return configcenter.Snapshot{}, canceledError(key, configcenter.OperationGet, err)
	}
	reader.mu.Lock()
	closed = reader.closed
	reader.mu.Unlock()
	if closed {
		return configcenter.Snapshot{}, readerError(configcenter.CodeReaderClosed, key, configcenter.OperationGet)
	}
	return configcenter.NewPresentSnapshot(key, content, configcenter.UnscopedRevision())
}

// ValidDataID reports whether a provider-neutral key can be used as an exact
// Nacos dataId without translation.
func ValidDataID(value string) bool {
	if value == "" || len(value) > 128 {
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

func (reader *Reader) Observe(context.Context, configcenter.Key) (configcenter.Observation, error) {
	return configcenter.Observation{}, readerError(configcenter.CodeUnsupported, configcenter.Key{}, configcenter.OperationObserve)
}

func (reader *Reader) Close() error {
	reader.mu.Lock()
	reader.closed = true
	reader.mu.Unlock()
	return nil
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

var _ configcenter.DynamicConfiguration = (*Reader)(nil)
