package gateway

import (
	"context"
	"errors"
)

// Outcome identifies a typed terminal outcome exposed by the external
// Gateway foundation. Route lifecycle states are represented by RouteState,
// not by errors.
type Outcome string

const (
	OutcomeInvalid      Outcome = "invalid"
	OutcomeUnsupported  Outcome = "unsupported"
	OutcomeUnauthorized Outcome = "unauthorized"
	OutcomeUnavailable  Outcome = "unavailable"
	OutcomeRejected     Outcome = "rejected"
	OutcomeNotReady     Outcome = "not_ready"
	OutcomeNotFound     Outcome = "not_found"
	OutcomeStale        Outcome = "stale"
	OutcomeCanceled     Outcome = "canceled"
	OutcomeClosed       Outcome = "closed"
)

// OutcomeCause is an allowlisted, provider-safe classification associated with
// a typed outcome. It must never contain provider payloads, route input,
// credentials, or arbitrary transport text.
type OutcomeCause string

const (
	CauseNone                       OutcomeCause = ""
	CauseInvalidInput               OutcomeCause = "invalid_input"
	CauseUnknownOutcome             OutcomeCause = "unknown_outcome"
	CauseUnknownCause               OutcomeCause = "unknown_cause"
	CauseRequiredCapability         OutcomeCause = "required_capability"
	CauseRouterDiscoveryUnsupported OutcomeCause = "router_discovery_unsupported"
	CauseRevisionReused             OutcomeCause = "revision_reused"
	CauseRouteNotDrainable          OutcomeCause = "route_not_drainable"
	CauseHTTPUnauthorized           OutcomeCause = "http_unauthorized"
	CauseHTTPForbidden              OutcomeCause = "http_forbidden"
	CauseProviderUnavailable        OutcomeCause = "provider_unavailable"
	CauseRateLimited                OutcomeCause = "rate_limited"
	CauseProviderRejected           OutcomeCause = "provider_rejected"
	CauseDataPlaneNotReady          OutcomeCause = "data_plane_not_ready"
	CauseRevisionMismatch           OutcomeCause = "revision_mismatch"
)

// OutcomeError is an immutable typed outcome error. Cause is restricted to a
// safe local classification; it must not contain provider payloads,
// credentials, route input, or response data.
type OutcomeError struct {
	outcome Outcome
	cause   OutcomeCause
	wrapped error
}

func (e *OutcomeError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.cause == "" {
		return "external gateway: " + string(e.outcome)
	}
	return "external gateway: " + string(e.outcome) + ": " + string(e.cause)
}

// Unwrap exposes only a local cancellation or deadline cause. It never
// exposes arbitrary provider errors.
func (e *OutcomeError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.wrapped
}

// Is makes the outcome sentinels usable with errors.Is.
func (e *OutcomeError) Is(target error) bool {
	targetOutcome, ok := target.(*OutcomeError)
	return ok && e != nil && targetOutcome != nil && e.outcome == targetOutcome.outcome
}

// Outcome returns this error's typed outcome code.
func (e *OutcomeError) Outcome() Outcome {
	if e == nil {
		return ""
	}
	return e.outcome
}

// Cause returns the safe local classification associated with this error.
func (e *OutcomeError) Cause() OutcomeCause {
	if e == nil {
		return ""
	}
	return e.cause
}

var (
	ErrInvalid      = &OutcomeError{outcome: OutcomeInvalid}
	ErrUnsupported  = &OutcomeError{outcome: OutcomeUnsupported}
	ErrUnauthorized = &OutcomeError{outcome: OutcomeUnauthorized}
	ErrUnavailable  = &OutcomeError{outcome: OutcomeUnavailable}
	ErrRejected     = &OutcomeError{outcome: OutcomeRejected}
	ErrNotReady     = &OutcomeError{outcome: OutcomeNotReady}
	ErrNotFound     = &OutcomeError{outcome: OutcomeNotFound}
	ErrStale        = &OutcomeError{outcome: OutcomeStale}
	ErrCanceled     = &OutcomeError{outcome: OutcomeCanceled}
	ErrClosed       = &OutcomeError{outcome: OutcomeClosed}
)

// NewOutcomeError creates a typed outcome error with an allowlisted safe
// classification. Unknown outcome or cause values become invalid input rather
// than expanding the public taxonomy or reflecting caller-controlled text.
func NewOutcomeError(outcome Outcome, cause OutcomeCause) *OutcomeError {
	if !validOutcome(outcome) {
		return &OutcomeError{outcome: OutcomeInvalid, cause: CauseUnknownOutcome}
	}
	if !validOutcomeCause(outcome, cause) {
		return &OutcomeError{outcome: OutcomeInvalid, cause: CauseUnknownCause}
	}
	return &OutcomeError{outcome: outcome, cause: cause}
}

func newInvalidError(_ string) *OutcomeError {
	return NewOutcomeError(OutcomeInvalid, CauseInvalidInput)
}

// NewCanceledError creates a canceled outcome which exposes only the local
// context cancellation/deadline through errors.Is. Provider errors must be
// classified with NewOutcomeError instead of being wrapped.
func NewCanceledError(ctxErr error) *OutcomeError {
	if ctxErr == nil {
		ctxErr = context.Canceled
	}
	if ctxErr != context.Canceled && ctxErr != context.DeadlineExceeded {
		return NewOutcomeError(OutcomeInvalid, CauseInvalidInput)
	}
	return &OutcomeError{outcome: OutcomeCanceled, wrapped: ctxErr}
}

func newCanceledError(ctxErr error) *OutcomeError { return NewCanceledError(ctxErr) }

func validOutcome(outcome Outcome) bool {
	switch outcome {
	case OutcomeInvalid, OutcomeUnsupported, OutcomeUnauthorized, OutcomeUnavailable,
		OutcomeRejected, OutcomeNotReady, OutcomeNotFound, OutcomeStale,
		OutcomeCanceled, OutcomeClosed:
		return true
	default:
		return false
	}
}

func validOutcomeCause(outcome Outcome, cause OutcomeCause) bool {
	switch outcome {
	case OutcomeInvalid:
		switch cause {
		case CauseNone, CauseInvalidInput, CauseUnknownOutcome, CauseUnknownCause, CauseRevisionReused:
			return true
		}
	case OutcomeUnsupported:
		return cause == CauseNone || cause == CauseRequiredCapability || cause == CauseRouterDiscoveryUnsupported
	case OutcomeUnauthorized:
		return cause == CauseNone || cause == CauseHTTPUnauthorized || cause == CauseHTTPForbidden
	case OutcomeUnavailable:
		return cause == CauseNone || cause == CauseProviderUnavailable || cause == CauseRateLimited
	case OutcomeRejected:
		return cause == CauseNone || cause == CauseProviderRejected || cause == CauseRouteNotDrainable
	case OutcomeNotReady:
		return cause == CauseNone || cause == CauseDataPlaneNotReady
	case OutcomeNotFound:
		return cause == CauseNone
	case OutcomeStale:
		return cause == CauseNone || cause == CauseRevisionMismatch
	case OutcomeCanceled, OutcomeClosed:
		return cause == CauseNone
	}
	return false
}

// OutcomeOf reports the typed outcome carried by err, including through
// ordinary wrapping.
func OutcomeOf(err error) (Outcome, bool) {
	var outcomeError *OutcomeError
	if errors.As(err, &outcomeError) && outcomeError != nil {
		return outcomeError.outcome, true
	}
	return "", false
}

// IsOutcome reports whether err has the given typed outcome.
func IsOutcome(err error, outcome Outcome) bool {
	if !validOutcome(outcome) {
		return false
	}
	return errors.Is(err, outcomeSentinel(outcome))
}

func IsInvalid(err error) bool      { return IsOutcome(err, OutcomeInvalid) }
func IsUnsupported(err error) bool  { return IsOutcome(err, OutcomeUnsupported) }
func IsUnauthorized(err error) bool { return IsOutcome(err, OutcomeUnauthorized) }
func IsUnavailable(err error) bool  { return IsOutcome(err, OutcomeUnavailable) }
func IsRejected(err error) bool     { return IsOutcome(err, OutcomeRejected) }
func IsNotReady(err error) bool     { return IsOutcome(err, OutcomeNotReady) }
func IsNotFound(err error) bool     { return IsOutcome(err, OutcomeNotFound) }
func IsStale(err error) bool        { return IsOutcome(err, OutcomeStale) }
func IsCanceled(err error) bool     { return IsOutcome(err, OutcomeCanceled) }
func IsClosed(err error) bool       { return IsOutcome(err, OutcomeClosed) }

func outcomeSentinel(outcome Outcome) error {
	switch outcome {
	case OutcomeInvalid:
		return ErrInvalid
	case OutcomeUnsupported:
		return ErrUnsupported
	case OutcomeUnauthorized:
		return ErrUnauthorized
	case OutcomeUnavailable:
		return ErrUnavailable
	case OutcomeRejected:
		return ErrRejected
	case OutcomeNotReady:
		return ErrNotReady
	case OutcomeNotFound:
		return ErrNotFound
	case OutcomeStale:
		return ErrStale
	case OutcomeCanceled:
		return ErrCanceled
	case OutcomeClosed:
		return ErrClosed
	default:
		return nil
	}
}
