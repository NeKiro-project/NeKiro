package configcenter

import "math"

// Revision identifies a point within one observation only. A zero Revision is
// intentionally unscoped and is used by one-shot Get snapshots.
//
// The observation scope is opaque. It cannot be constructed by callers, which
// prevents a one-shot revision from accidentally participating in observation
// ordering and keeps revisions free of provider-persistent meaning.
type Revision struct {
	scope *revisionScope
	order uint64
}

// marker prevents zero-sized allocation coalescing from making two scopes
// compare equal.
type revisionScope struct{ marker byte }

// UnscopedRevision returns the explicit non-comparable revision used by
// one-shot Get snapshots.
func UnscopedRevision() Revision {
	return Revision{}
}

// NewObservationRevision returns the initial (order zero) revision for a new
// observation scope.
func NewObservationRevision() Revision {
	return Revision{scope: &revisionScope{}}
}

// Scoped reports whether revision belongs to an observation. A Get snapshot
// always has an unscoped revision.
func (revision Revision) Scoped() bool {
	return revision.scope != nil
}

// Order returns the local order within an observation. It has no meaning when
// Scoped reports false and no ordering meaning across observations.
func (revision Revision) Order() uint64 {
	return revision.order
}

// AdvanceRevision returns the exact successor of an observation revision.
// Advancing an unscoped revision is invalid. An exhausted local order fails
// explicitly rather than silently wrapping to zero.
func AdvanceRevision(revision Revision) (Revision, error) {
	if !revision.Scoped() {
		return Revision{}, NewError(CodeRevisionOutOfOrder, ErrorDetails{
			Operation: OperationRevision,
			Revision:  revision,
		})
	}
	if revision.order == math.MaxUint64 {
		return Revision{}, NewError(CodeUnavailable, ErrorDetails{
			Operation: OperationRevision,
			Revision:  revision,
		})
	}
	return Revision{scope: revision.scope, order: revision.order + 1}, nil
}

// ValidateNextRevision verifies that candidate is the immediate successor of
// previous in the same observation. It does not create recovery or cursor
// semantics: callers must handle a classified invalid relationship directly.
func ValidateNextRevision(previous, candidate Revision) error {
	if !previous.Scoped() || !candidate.Scoped() || previous.scope != candidate.scope {
		return NewError(CodeRevisionOutOfOrder, ErrorDetails{
			Operation: OperationRevision,
			Revision:  candidate,
		})
	}

	switch {
	case candidate.order == previous.order:
		return NewError(CodeRevisionDuplicate, ErrorDetails{
			Operation: OperationRevision,
			Revision:  candidate,
		})
	case candidate.order < previous.order:
		return NewError(CodeRevisionStale, ErrorDetails{
			Operation: OperationRevision,
			Revision:  candidate,
		})
	case previous.order == math.MaxUint64 || candidate.order != previous.order+1:
		return NewError(CodeRevisionGap, ErrorDetails{
			Operation: OperationRevision,
			Revision:  candidate,
		})
	default:
		return nil
	}
}
