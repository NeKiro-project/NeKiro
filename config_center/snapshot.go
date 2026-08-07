package configcenter

import "bytes"

// Snapshot is one immutable configuration state. A missing snapshot has
// Present false; a present snapshot may legitimately have zero content bytes.
type Snapshot struct {
	key      Key
	present  bool
	content  []byte
	revision Revision
}

// NewPresentSnapshot creates a present immutable snapshot. The constructor
// copies content, and Content returns a fresh copy on every call.
func NewPresentSnapshot(key Key, content []byte, revision Revision) (Snapshot, error) {
	if !key.Valid() {
		return Snapshot{}, NewError(CodeInvalid, ErrorDetails{
			Key:       key,
			Operation: OperationSnapshot,
		})
	}
	return Snapshot{
		key:      key,
		present:  true,
		content:  bytes.Clone(content),
		revision: revision,
	}, nil
}

// NewMissingSnapshot creates an immutable missing snapshot. Missing has no
// content and remains distinct from a successful present-empty snapshot.
func NewMissingSnapshot(key Key, revision Revision) (Snapshot, error) {
	if !key.Valid() {
		return Snapshot{}, NewError(CodeInvalid, ErrorDetails{
			Key:       key,
			Operation: OperationSnapshot,
		})
	}
	return Snapshot{key: key, revision: revision}, nil
}

// Key returns the snapshot's exact validated key.
func (snapshot Snapshot) Key() Key {
	return snapshot.key
}

// Present reports whether the key existed when the snapshot was acquired.
func (snapshot Snapshot) Present() bool {
	return snapshot.present
}

// Content returns a caller-owned copy of the opaque bytes. Missing snapshots
// return nil; callers must use Present to distinguish missing from present
// empty content.
func (snapshot Snapshot) Content() []byte {
	return bytes.Clone(snapshot.content)
}

// Revision returns the snapshot's observation-local or unscoped revision.
func (snapshot Snapshot) Revision() Revision {
	return snapshot.revision
}

func (snapshot Snapshot) valid() bool {
	return snapshot.key.Valid() && (snapshot.present || snapshot.content == nil)
}
