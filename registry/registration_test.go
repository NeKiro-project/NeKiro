package registry

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRegistrationCopiesExactTargetAndInstance(t *testing.T) {
	target := registrationTarget(t)
	zone := "zone-a"
	weight := 80
	endpoint, _ := NewNetworkEndpoint(NetworkEndpointInput{AddressType: AddressTypeIPv4, Address: "127.0.0.1", PortName: "a2a", Port: 8092, Protocol: TransportProtocolTCP})
	instance, _ := NewInstance(InstanceInput{ID: "runtime-a", Endpoints: []NetworkEndpoint{endpoint}, Ready: true, Serving: true, Zone: &zone, Weight: &weight, Metadata: map[string]string{"safe": "value"}})
	registration, err := NewRegistration(RegistrationInput{Target: target, Instance: instance})
	if err != nil {
		t.Fatal(err)
	}
	if err := registration.Validate(); err != nil {
		t.Fatal(err)
	}
	copy := registration.Instance()
	metadata := copy.Metadata()
	metadata["safe"] = "changed"
	if !registration.Target().Equal(target) || !registration.Instance().Equal(instance) {
		t.Fatal("registration did not preserve immutable identity")
	}
	if _, err := NewRegistration(RegistrationInput{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid registration error=%v", err)
	}
	unavailable, _ := NewInstance(InstanceInput{ID: "runtime-a", Endpoints: []NetworkEndpoint{endpoint}})
	if _, err := NewRegistration(RegistrationInput{Target: target, Instance: unavailable}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unavailable registration error=%v", err)
	}
	if err := (Registration{}).Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("zero registration error=%v", err)
	}
}

func TestLeaseLatchesOneTypedTerminalOutcome(t *testing.T) {
	closes := 0
	lease, err := NewLease(func(context.Context) error { closes++; return nil })
	if err != nil {
		t.Fatal(err)
	}
	lease.Terminate(ErrUnavailable)
	lease.Terminate(ErrClosed)
	<-lease.Done()
	if !errors.Is(lease.Err(), ErrUnavailable) {
		t.Fatalf("lease terminal=%v", lease.Err())
	}
	if err := lease.Close(nil); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil close error=%v", err)
	}
	if err := lease.Close(t.Context()); err != nil || closes != 1 {
		t.Fatalf("Close error=%v calls=%d", err, closes)
	}
	if err := lease.Close(t.Context()); err != nil || closes != 1 {
		t.Fatalf("second Close error=%v calls=%d", err, closes)
	}
	if _, err := NewLease(nil); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil close function error=%v", err)
	}
}

func registrationTarget(t *testing.T) ReleaseTarget {
	t.Helper()
	target, err := NewReleaseTarget(ReleaseTargetInput{
		AgentID: "runtime-a", AgentCardVersion: "1.0.0", ReleaseID: "release-a",
		CardDigest: strings.Repeat("a", 64), CanonicalEndpoint: "http://runtime-a:8092/a2a", Audience: "http://runtime-a:8092",
	})
	if err != nil {
		t.Fatal(err)
	}
	return target
}
