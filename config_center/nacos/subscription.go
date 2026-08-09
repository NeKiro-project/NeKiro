package nacos

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	configcenter "github.com/NeKiro-project/NeKiro/config_center"
)

const (
	listenerRecordSeparator = "\x01"
	listenerFieldSeparator  = "\x02"
)

type subscription struct {
	reader *Reader
	key    configcenter.Key
	events chan configcenter.ConfigurationEvent
	done   chan struct{}
	ctx    context.Context
	cancel context.CancelFunc

	terminalMu sync.Mutex
	terminal   error
	suppressed atomic.Uint64

	currentMu sync.Mutex
	current   remoteState
	revision  configcenter.Revision
}

func newSubscription(reader *Reader, key configcenter.Key, current remoteState, revision configcenter.Revision) *subscription {
	watchContext, cancel := context.WithCancel(context.Background())
	return &subscription{
		reader:   reader,
		key:      key,
		events:   make(chan configcenter.ConfigurationEvent, reader.subscriptionBuffer),
		done:     make(chan struct{}),
		ctx:      watchContext,
		cancel:   cancel,
		current:  remoteState{present: current.present, content: bytes.Clone(current.content)},
		revision: revision,
	}
}

func (subscription *subscription) Next(ctx context.Context) (configcenter.ConfigurationEvent, error) {
	if ctx == nil {
		return configcenter.ConfigurationEvent{}, readerError(configcenter.CodeInvalid, subscription.key, configcenter.OperationNext)
	}
	if err := ctx.Err(); err != nil {
		return configcenter.ConfigurationEvent{}, canceledError(subscription.key, configcenter.OperationNext, err)
	}
	select {
	case <-subscription.done:
		return configcenter.ConfigurationEvent{}, subscription.terminalError()
	default:
	}
	select {
	case <-subscription.done:
		return configcenter.ConfigurationEvent{}, subscription.terminalError()
	case event := <-subscription.events:
		return event, nil
	case <-ctx.Done():
		return configcenter.ConfigurationEvent{}, canceledError(subscription.key, configcenter.OperationNext, ctx.Err())
	}
}

func (subscription *subscription) Close() error {
	subscription.reader.closeSubscription(subscription)
	return nil
}

func (subscription *subscription) Stats() configcenter.SubscriptionStats {
	return configcenter.SubscriptionStats{SuppressedNotifications: subscription.suppressed.Load()}
}

func (subscription *subscription) terminate(err error) {
	subscription.terminalMu.Lock()
	if subscription.terminal != nil {
		subscription.terminalMu.Unlock()
		return
	}
	subscription.terminal = err
	subscription.cancel()
	close(subscription.done)
	subscription.terminalMu.Unlock()
}

func (subscription *subscription) terminalError() error {
	subscription.terminalMu.Lock()
	defer subscription.terminalMu.Unlock()
	if subscription.terminal == nil {
		panic("config center nacos: terminal subscription without terminal error")
	}
	return subscription.terminal
}

func (subscription *subscription) watch() {
	ctx := subscription.ctx
	for {
		changed, err := subscription.listen(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			cause, _ := configcenter.CodeOf(err)
			subscription.reader.interruptAll(cause)
			return
		}
		if !changed {
			continue
		}
		state, err := subscription.reader.readState(ctx, subscription.key, configcenter.OperationWatch)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			cause, _ := configcenter.CodeOf(err)
			subscription.reader.interruptAll(cause)
			return
		}
		if !subscription.apply(state) {
			subscription.reader.interruptAll("")
			return
		}
	}
}

func (subscription *subscription) listen(ctx context.Context) (bool, error) {
	subscription.currentMu.Lock()
	digest := contentDigest(subscription.current)
	subscription.currentMu.Unlock()

	endpoint := *subscription.reader.origin
	endpoint.Path = strings.TrimSuffix(endpoint.Path, "/") + "/v1/cs/configs/listener"
	query := endpoint.Query()
	if subscription.reader.authMode == AuthAccessToken {
		query.Set("accessToken", subscription.reader.accessToken)
	}
	endpoint.RawQuery = query.Encode()
	dataID := dataIDForKey(subscription.key)
	record := strings.Join([]string{dataID, subscription.reader.groupName, digest, subscription.reader.namespaceID}, listenerFieldSeparator) + listenerRecordSeparator
	form := url.Values{"Listening-Configs": []string{record}}
	requestContext, cancel := context.WithTimeout(ctx, subscription.reader.longPollTimeout+time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, http.MethodPost, endpoint.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return false, readerError(configcenter.CodeInvalid, subscription.key, configcenter.OperationWatch)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Long-Pulling-Timeout", strconv.FormatInt(subscription.reader.longPollTimeout.Milliseconds(), 10))
	response, err := subscription.reader.executor.Do(request)
	if err != nil {
		return false, readerError(configcenter.CodeUnavailable, subscription.key, configcenter.OperationWatch)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return false, readerError(configcenter.CodeUnauthorized, subscription.key, configcenter.OperationWatch)
	}
	if response.StatusCode != http.StatusOK {
		return false, readerError(configcenter.CodeUnavailable, subscription.key, configcenter.OperationWatch)
	}
	expectedResponse := url.QueryEscape(strings.Join([]string{dataID, subscription.reader.groupName, subscription.reader.namespaceID}, listenerFieldSeparator) + listenerRecordSeparator)
	responseLimit := int64(len(expectedResponse))
	body, err := io.ReadAll(io.LimitReader(response.Body, responseLimit+1))
	if err != nil {
		return false, readerError(configcenter.CodeUnavailable, subscription.key, configcenter.OperationWatch)
	}
	if int64(len(body)) > responseLimit {
		return false, readerError(configcenter.CodePayloadTooLarge, subscription.key, configcenter.OperationWatch)
	}
	return parseListenerResponse(string(body), dataID, subscription.reader.groupName, subscription.reader.namespaceID)
}

func parseListenerResponse(body, dataID, group, tenant string) (bool, error) {
	if body == "" {
		return false, nil
	}
	decoded, err := url.QueryUnescape(body)
	if err != nil {
		return false, readerError(configcenter.CodeUnavailable, configcenter.Key{}, configcenter.OperationWatch)
	}
	records := strings.Split(decoded, listenerRecordSeparator)
	if len(records) != 2 || records[1] != "" {
		return false, readerError(configcenter.CodeUnavailable, configcenter.Key{}, configcenter.OperationWatch)
	}
	fields := strings.Split(records[0], listenerFieldSeparator)
	if len(fields) != 3 || fields[0] != dataID || fields[1] != group || fields[2] != tenant {
		return false, readerError(configcenter.CodeUnavailable, configcenter.Key{}, configcenter.OperationWatch)
	}
	return true, nil
}

func (subscription *subscription) apply(state remoteState) bool {
	subscription.currentMu.Lock()
	defer subscription.currentMu.Unlock()
	if sameRemoteState(subscription.current, state) {
		subscription.suppressed.Add(1)
		return true
	}
	nextRevision, err := configcenter.AdvanceRevision(subscription.revision)
	if err != nil {
		return false
	}
	snapshot, err := snapshotForState(subscription.key, state, nextRevision)
	if err != nil {
		return false
	}
	var event configcenter.ConfigurationEvent
	if state.present {
		event, err = configcenter.NewUpdateEvent(snapshot)
	} else {
		event, err = configcenter.NewDeleteEvent(snapshot)
	}
	if err != nil {
		return false
	}
	select {
	case subscription.events <- event:
		subscription.current = remoteState{present: state.present, content: bytes.Clone(state.content)}
		subscription.revision = nextRevision
		return true
	default:
		return false
	}
}

func contentDigest(state remoteState) string {
	if !state.present {
		return ""
	}
	digest := md5.Sum(state.content)
	return hex.EncodeToString(digest[:])
}

func sameRemoteState(left, right remoteState) bool {
	return left.present == right.present && (!left.present || bytes.Equal(left.content, right.content))
}

var _ configcenter.Subscription = (*subscription)(nil)
