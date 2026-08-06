package configcenter

// EventKind identifies the explicit state transition represented by an event.
type EventKind string

const (
	// EventUpdate carries a present snapshot, including a present-empty value.
	EventUpdate EventKind = "update"
	// EventDelete carries a missing snapshot and never represents empty content.
	EventDelete EventKind = "delete"
)

// ConfigurationEvent is an immutable ordered transition for one observation.
type ConfigurationEvent struct {
	kind     EventKind
	snapshot Snapshot
}

// NewUpdateEvent creates an update event for a present, observation-scoped
// snapshot.
func NewUpdateEvent(snapshot Snapshot) (ConfigurationEvent, error) {
	return newConfigurationEvent(EventUpdate, snapshot)
}

// NewDeleteEvent creates a delete event for a missing, observation-scoped
// snapshot.
func NewDeleteEvent(snapshot Snapshot) (ConfigurationEvent, error) {
	return newConfigurationEvent(EventDelete, snapshot)
}

// Kind returns the explicit event kind.
func (event ConfigurationEvent) Kind() EventKind {
	return event.kind
}

// Snapshot returns the event's immutable snapshot value.
func (event ConfigurationEvent) Snapshot() Snapshot {
	return event.snapshot
}

func newConfigurationEvent(kind EventKind, snapshot Snapshot) (ConfigurationEvent, error) {
	if !snapshot.valid() || !snapshot.revision.Scoped() ||
		(kind == EventUpdate && !snapshot.present) ||
		(kind == EventDelete && snapshot.present) {
		return ConfigurationEvent{}, NewError(CodeInvalid, ErrorDetails{
			Key:       snapshot.key,
			Operation: OperationEvent,
			Revision:  snapshot.revision,
		})
	}
	return ConfigurationEvent{kind: kind, snapshot: snapshot}, nil
}
