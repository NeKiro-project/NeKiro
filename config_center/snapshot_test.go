package configcenter

import (
	"bytes"
	"errors"
	"testing"
)

func TestSnapshotCopiesInputAndOutputBytes(t *testing.T) {
	key, err := ParseKey("alpha")
	if err != nil {
		t.Fatal(err)
	}
	input := []byte("before")
	snapshot, err := NewPresentSnapshot(key, input, UnscopedRevision())
	if err != nil {
		t.Fatal(err)
	}
	input[0] = 'X'
	if got := string(snapshot.Content()); got != "before" {
		t.Fatalf("snapshot retained caller mutation: %q", got)
	}
	returned := snapshot.Content()
	returned[0] = 'Y'
	if got := string(snapshot.Content()); got != "before" {
		t.Fatalf("snapshot exposed retained bytes: %q", got)
	}

	empty, err := NewPresentSnapshot(key, []byte{}, UnscopedRevision())
	if err != nil {
		t.Fatal(err)
	}
	if !empty.Present() || len(empty.Content()) != 0 {
		t.Fatalf("present-empty snapshot lost state: present=%v content=%v", empty.Present(), empty.Content())
	}
	missing, err := NewMissingSnapshot(key, UnscopedRevision())
	if err != nil {
		t.Fatal(err)
	}
	if missing.Present() || missing.Content() != nil {
		t.Fatalf("missing snapshot was not distinct: present=%v content=%v", missing.Present(), missing.Content())
	}
	if !bytes.Equal(snapshot.Content(), []byte("before")) {
		t.Fatal("snapshot content changed unexpectedly")
	}
}

func TestEventsRequireExplicitStateAndObservationRevision(t *testing.T) {
	key, err := ParseKey("alpha")
	if err != nil {
		t.Fatal(err)
	}
	scope := NewObservationRevision()
	present, err := NewPresentSnapshot(key, []byte("v"), scope)
	if err != nil {
		t.Fatal(err)
	}
	missing, err := NewMissingSnapshot(key, scope)
	if err != nil {
		t.Fatal(err)
	}
	update, err := NewUpdateEvent(present)
	if err != nil || update.Kind() != EventUpdate || !update.Snapshot().Present() {
		t.Fatalf("valid update event rejected: %#v %v", update, err)
	}
	deletion, err := NewDeleteEvent(missing)
	if err != nil || deletion.Kind() != EventDelete || deletion.Snapshot().Present() {
		t.Fatalf("valid delete event rejected: %#v %v", deletion, err)
	}
	for _, makeEvent := range []func() (ConfigurationEvent, error){
		func() (ConfigurationEvent, error) { return NewUpdateEvent(missing) },
		func() (ConfigurationEvent, error) { return NewDeleteEvent(present) },
		func() (ConfigurationEvent, error) {
			unscoped, constructorErr := NewPresentSnapshot(key, []byte("v"), UnscopedRevision())
			if constructorErr != nil {
				return ConfigurationEvent{}, constructorErr
			}
			return NewUpdateEvent(unscoped)
		},
	} {
		if _, eventErr := makeEvent(); eventErr == nil || !errors.Is(eventErr, ErrInvalid) {
			t.Fatalf("invalid event error = %v, want invalid", eventErr)
		}
	}
}

func TestSnapshotRejectsInvalidKey(t *testing.T) {
	var invalid Key
	if _, err := NewPresentSnapshot(invalid, []byte("x"), UnscopedRevision()); err == nil || !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid present snapshot error = %v", err)
	}
	if _, err := NewMissingSnapshot(invalid, UnscopedRevision()); err == nil || !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid missing snapshot error = %v", err)
	}
}
