package configcenter

import (
	"context"
	"errors"
)

// Code classifies a provider-neutral configuration outcome.
type Code string

const (
	CodeInvalid            Code = "invalid"
	CodeMissing            Code = "missing"
	CodeUnsafeState        Code = "unsafe_state"
	CodeUnauthorized       Code = "unauthorized"
	CodeUnavailable        Code = "unavailable"
	CodeUnsupported        Code = "unsupported"
	CodeCanceled           Code = "canceled"
	CodeSubscriptionClosed Code = "subscription_closed"
	CodeReaderClosed       Code = "reader_closed"
	CodePublisherClosed    Code = "publisher_closed"
	CodeWatchInterrupted   Code = "watch_interrupted"
	CodePayloadTooLarge    Code = "payload_too_large"
	CodeRevisionDuplicate  Code = "revision_duplicate"
	CodeRevisionStale      Code = "revision_stale"
	CodeRevisionGap        Code = "revision_gap"
	CodeRevisionOutOfOrder Code = "revision_out_of_order"
)

// Operation identifies a safe operation boundary in an Error.
type Operation string

const (
	OperationValidateKey      Operation = "validate_key"
	OperationValidateProvider Operation = "validate_provider"
	OperationRevision         Operation = "revision"
	OperationSnapshot         Operation = "snapshot"
	OperationEvent            Operation = "event"
	OperationFactory          Operation = "factory"
	OperationGet              Operation = "get"
	OperationObserve          Operation = "observe"
	OperationNext             Operation = "next"
	OperationRead             Operation = "read"
	OperationWatch            Operation = "watch"
	OperationSubscription     Operation = "subscription"
	OperationPublish          Operation = "publish"
	OperationDelete           Operation = "delete"
	OperationClose            Operation = "close"
)

var (
	ErrInvalid            = &Error{code: CodeInvalid}
	ErrMissing            = &Error{code: CodeMissing}
	ErrUnsafeState        = &Error{code: CodeUnsafeState}
	ErrUnauthorized       = &Error{code: CodeUnauthorized}
	ErrUnavailable        = &Error{code: CodeUnavailable}
	ErrUnsupported        = &Error{code: CodeUnsupported}
	ErrCanceled           = &Error{code: CodeCanceled}
	ErrSubscriptionClosed = &Error{code: CodeSubscriptionClosed}
	ErrReaderClosed       = &Error{code: CodeReaderClosed}
	ErrPublisherClosed    = &Error{code: CodePublisherClosed}
	ErrWatchInterrupted   = &Error{code: CodeWatchInterrupted}
	ErrPayloadTooLarge    = &Error{code: CodePayloadTooLarge}
	ErrRevisionDuplicate  = &Error{code: CodeRevisionDuplicate}
	ErrRevisionStale      = &Error{code: CodeRevisionStale}
	ErrRevisionGap        = &Error{code: CodeRevisionGap}
	ErrRevisionOutOfOrder = &Error{code: CodeRevisionOutOfOrder}
)

// ErrorDetails contains only safe metadata. It intentionally has no field for
// configuration content, a filesystem path, credentials, or a dependency error.
type ErrorDetails struct {
	Provider  ProviderID
	Key       Key
	Operation Operation
	Revision  Revision
	CauseKind Code
}

// Error is an inspectable typed configuration error.
type Error struct {
	code    Code
	details ErrorDetails
	context error
}

// NewError creates an error using one closed classification. Supplying an
// unknown code is a programmer error rather than a hidden runtime fallback.
func NewError(code Code, details ErrorDetails) *Error {
	if !code.valid() {
		panic("configcenter: unknown error code")
	}
	if details.CauseKind != "" && (code != CodeWatchInterrupted || !details.CauseKind.watchCause()) {
		panic("configcenter: invalid watch interruption cause")
	}
	return &Error{code: code, details: details}
}

// NewCanceledError records a canceled operation while preserving only the
// standard context cancellation cause for errors.Is callers.
func NewCanceledError(details ErrorDetails, cause error) *Error {
	if details.CauseKind != "" {
		panic("configcenter: canceled error cannot have a cause kind")
	}
	if !errors.Is(cause, context.Canceled) && !errors.Is(cause, context.DeadlineExceeded) {
		panic("configcenter: canceled error requires a context cancellation cause")
	}
	return &Error{code: CodeCanceled, details: details, context: cause}
}

// Error returns a stable classification string without dynamic metadata.
func (err *Error) Error() string {
	if err == nil {
		return "<nil>"
	}
	return "config center " + string(err.code)
}

// Is matches errors by classification.
func (err *Error) Is(target error) bool {
	other, ok := target.(*Error)
	return ok && err != nil && err.code == other.code
}

// Unwrap exposes only standard context cancellation causes. Provider failures
// are intentionally not wrapped because their implementation details are not
// part of this contract.
func (err *Error) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.context
}

// Code returns the closed classification.
func (err *Error) Code() Code {
	if err == nil {
		return ""
	}
	return err.code
}

// Details returns safe immutable metadata.
func (err *Error) Details() ErrorDetails {
	if err == nil {
		return ErrorDetails{}
	}
	return err.details
}

// CodeOf extracts a closed classification from err.
func CodeOf(err error) (Code, bool) {
	var typed *Error
	if !errors.As(err, &typed) {
		return "", false
	}
	return typed.Code(), true
}

func (code Code) valid() bool {
	switch code {
	case CodeInvalid, CodeMissing, CodeUnsafeState, CodeUnauthorized, CodeUnavailable,
		CodeUnsupported, CodeCanceled, CodeSubscriptionClosed, CodeReaderClosed,
		CodePublisherClosed, CodeWatchInterrupted, CodePayloadTooLarge,
		CodeRevisionDuplicate, CodeRevisionStale, CodeRevisionGap,
		CodeRevisionOutOfOrder:
		return true
	default:
		return false
	}
}

func (code Code) watchCause() bool {
	switch code {
	case CodeUnsafeState, CodeUnauthorized, CodeUnavailable, CodePayloadTooLarge:
		return true
	default:
		return false
	}
}
