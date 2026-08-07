package configcenter

import (
	"context"
	"errors"
	"testing"
)

type factoryReader struct{}

func (factoryReader) Get(context.Context, Key) (Snapshot, error) { return Snapshot{}, nil }
func (factoryReader) Observe(context.Context, Key) (Observation, error) {
	return Observation{}, nil
}
func (factoryReader) Close() error { return nil }

type nilFactoryReader struct{}

func (*nilFactoryReader) Get(context.Context, Key) (Snapshot, error) { return Snapshot{}, nil }
func (*nilFactoryReader) Observe(context.Context, Key) (Observation, error) {
	return Observation{}, nil
}
func (*nilFactoryReader) Close() error { return nil }

func TestFactoryCopiesExactProviderMapAndHasNoDefault(t *testing.T) {
	provider, err := ParseProviderID("file")
	if err != nil {
		t.Fatal(err)
	}
	reader := factoryReader{}
	readers := map[ProviderID]DynamicConfiguration{provider: reader}
	factory, err := NewFactory(readers)
	if err != nil {
		t.Fatal(err)
	}
	delete(readers, provider)
	resolved, err := factory.Reader(provider)
	if err != nil || resolved == nil {
		t.Fatalf("copied provider map lost reader: %v", err)
	}

	newProvider, err := ParseProviderID("other")
	if err != nil {
		t.Fatal(err)
	}
	readers[newProvider] = reader
	if _, err := factory.Reader(newProvider); err == nil || !errors.Is(err, ErrUnsupported) {
		t.Fatalf("factory acquired a provider added after construction: %v", err)
	}
	if _, err := factory.Reader(ProviderID("missing")); err == nil || !errors.Is(err, ErrUnsupported) {
		t.Fatalf("unknown provider error = %v, want unsupported", err)
	}
}

func TestFactoryRejectsAbsentOrInvalidComposition(t *testing.T) {
	if _, err := NewFactory(nil); err == nil || !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil map error = %v, want invalid", err)
	}
	if _, err := NewFactory(map[ProviderID]DynamicConfiguration{ProviderID("bad/id"): factoryReader{}}); err == nil || !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid provider error = %v, want invalid", err)
	}
	var typedNil *nilFactoryReader
	if _, err := NewFactory(map[ProviderID]DynamicConfiguration{ProviderID("file"): typedNil}); err == nil || !errors.Is(err, ErrInvalid) {
		t.Fatalf("typed nil reader error = %v, want invalid", err)
	}
	if _, err := NewFactory(map[ProviderID]DynamicConfiguration{ProviderID("bad/id"): nil}); err == nil || !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil reader error = %v, want invalid", err)
	}
}

var _ DynamicConfiguration = factoryReader{}
