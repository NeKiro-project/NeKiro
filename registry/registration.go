package registry

import (
	"context"
	"sync"
)

// RegistrationInput identifies one exact runtime instance to publish.
type RegistrationInput struct {
	Target   ReleaseTarget
	Instance Instance
}

// Registration is an immutable exact-Release runtime registration request.
type Registration struct {
	target   ReleaseTarget
	instance Instance
}

func NewRegistration(input RegistrationInput) (Registration, error) {
	if input.Target.Validate() != nil || input.Instance.Validate() != nil || input.Instance.State() != InstanceStateReady {
		return Registration{}, ErrInvalid
	}
	return Registration{target: input.Target, instance: cloneRegistrationInstance(input.Instance)}, nil
}

func (r Registration) Validate() error {
	if r.target.Validate() != nil || r.instance.Validate() != nil || r.instance.State() != InstanceStateReady {
		return ErrInvalid
	}
	return nil
}

func (r Registration) Target() ReleaseTarget { return r.target }
func (r Registration) Instance() Instance    { return cloneRegistrationInstance(r.instance) }

// InstanceRegistrar publishes one exact runtime and owns its lease lifecycle.
type InstanceRegistrar interface {
	Register(context.Context, Registration) (InstanceLease, error)
	Capabilities() Capabilities
	Close() error
}

// InstanceLease represents one continuously maintained ephemeral registration.
type InstanceLease interface {
	Done() <-chan struct{}
	Err() error
	Close(context.Context) error
}

type Lease struct {
	done      chan struct{}
	doneOnce  sync.Once
	closeOnce sync.Once
	mu        sync.RWMutex
	terminal  error
	closeErr  error
	closeFunc func(context.Context) error
}

// NewLease creates the provider side of an InstanceLease.
func NewLease(closeFunc func(context.Context) error) (*Lease, error) {
	if closeFunc == nil {
		return nil, ErrInvalid
	}
	return &Lease{done: make(chan struct{}), closeFunc: closeFunc}, nil
}

func (l *Lease) Done() <-chan struct{} { return l.done }

func (l *Lease) Err() error {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.terminal
}

func (l *Lease) Close(ctx context.Context) error {
	if ctx == nil {
		return ErrInvalid
	}
	l.closeOnce.Do(func() { l.closeErr = l.closeFunc(ctx) })
	return l.closeErr
}

// Terminate latches one provider-safe terminal outcome and wakes observers.
func (l *Lease) Terminate(err error) {
	if err == nil {
		err = ErrClosed
	}
	l.doneOnce.Do(func() {
		l.mu.Lock()
		l.terminal = typedTerminal(err)
		l.mu.Unlock()
		close(l.done)
	})
}

func cloneRegistrationInstance(instance Instance) Instance {
	var zone *string
	if value, ok := instance.Zone(); ok {
		zone = &value
	}
	var weight *int
	if value, ok := instance.Weight(); ok {
		weight = &value
	}
	copy, err := NewInstance(InstanceInput{
		ID: instance.ID(), Endpoints: instance.Endpoints(), Ready: instance.Ready(),
		Serving: instance.Serving(), Terminating: instance.Terminating(),
		Zone: zone, Weight: weight, Metadata: instance.Metadata(),
	})
	if err != nil {
		return Instance{}
	}
	return copy
}

var _ InstanceLease = (*Lease)(nil)
