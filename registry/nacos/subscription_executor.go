package nacos

import (
	"context"
	"sync"

	"github.com/NeKiro-project/NeKiro/registry"
)

const NamingSubscriptionExecutorVersionV1 = "v1"

// NamingSubscriptionGuarantees excludes the recovery and cache behavior of
// general-purpose Nacos SDK clients from this provider boundary.
type NamingSubscriptionGuarantees struct {
	Version                    string
	AtomicInitialPushHandoff   bool
	ExactlyOneSubscribeAttempt bool
	NoRetry                    bool
	NoReconnect                bool
	NoResponseCache            bool
	NoFailover                 bool
	NoImplicitPolling          bool
	NoHiddenReauthentication   bool
}

func (guarantees NamingSubscriptionGuarantees) Validate() error {
	if guarantees.Version != NamingSubscriptionExecutorVersionV1 ||
		!guarantees.AtomicInitialPushHandoff || !guarantees.ExactlyOneSubscribeAttempt ||
		!guarantees.NoRetry || !guarantees.NoReconnect || !guarantees.NoResponseCache ||
		!guarantees.NoFailover || !guarantees.NoImplicitPolling || !guarantees.NoHiddenReauthentication {
		return registry.ErrInvalid
	}
	return nil
}

type NamingSubscribeRequest struct {
	namespaceID string
	serviceName string
	groupName   string
	clusterName string
}

func newNamingSubscribeRequest(namespaceID string, binding Binding) NamingSubscribeRequest {
	return NamingSubscribeRequest{namespaceID: namespaceID, serviceName: binding.serviceName, groupName: binding.groupName, clusterName: binding.clusterName}
}

func (request NamingSubscribeRequest) NamespaceID() string { return request.namespaceID }
func (request NamingSubscribeRequest) ServiceName() string { return request.serviceName }
func (request NamingSubscribeRequest) GroupName() string   { return request.groupName }
func (request NamingSubscribeRequest) ClusterName() string { return request.clusterName }

// NamingPushStream yields complete raw Nacos ServiceInfo JSON payloads. EOF,
// transport failure, or Close is terminal; implementations must not reconnect.
type NamingPushStream interface {
	Next(context.Context) ([]byte, error)
	Close() error
}

type NamingSubscriptionExecutor interface {
	Guarantees() NamingSubscriptionGuarantees
	Subscribe(context.Context, NamingSubscribeRequest) (NamingSubscription, error)
}

type NamingSubscription struct {
	initial []byte
	stream  NamingPushStream
	close   *subscriptionClose
}

type subscriptionClose struct {
	once sync.Once
	err  error
}

func NewNamingSubscription(initial []byte, stream NamingPushStream) (NamingSubscription, error) {
	if len(initial) == 0 || isNilInterface(stream) {
		return NamingSubscription{}, registry.ErrInvalid
	}
	return NamingSubscription{initial: append([]byte(nil), initial...), stream: stream, close: &subscriptionClose{}}, nil
}

func (subscription NamingSubscription) InitialPayload() []byte {
	return append([]byte(nil), subscription.initial...)
}

func (subscription NamingSubscription) Next(ctx context.Context) ([]byte, error) {
	if isNilInterface(subscription.stream) {
		return nil, registry.ErrInvalid
	}
	return subscription.stream.Next(ctx)
}

func (subscription NamingSubscription) Close() error {
	if isNilInterface(subscription.stream) || subscription.close == nil {
		return nil
	}
	subscription.close.once.Do(func() { subscription.close.err = subscription.stream.Close() })
	return subscription.close.err
}
