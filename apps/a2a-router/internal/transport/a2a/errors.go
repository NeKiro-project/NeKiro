package a2a

import (
	"context"
	"errors"

	a2atransport "github.com/NeKiro-project/nekiro-a2a-transport-go"
	"github.com/NeKiro-project/NeKiro/contracts"
)

// classifiedError carries the platform classification across the transport
// seam without exposing transport implementation types to the API package.
type classifiedError struct {
	code  contracts.PlatformErrorCode
	cause error
}

func (err *classifiedError) Error() string {
	return string(err.code)
}

func (err *classifiedError) Unwrap() error {
	return err.cause
}

func (err *classifiedError) PlatformErrorCode() contracts.PlatformErrorCode {
	return err.code
}

func classify(code contracts.PlatformErrorCode, cause error) error {
	if cause == nil {
		cause = errors.New(string(code))
	}
	return &classifiedError{code: code, cause: cause}
}

func classifyTransportError(err error) error {
	if err == nil {
		return nil
	}
	var alreadyClassified interface {
		error
		PlatformErrorCode() contracts.PlatformErrorCode
	}
	if errors.As(err, &alreadyClassified) {
		return alreadyClassified
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return classify(contracts.ErrorCodeTimeout, transportCause(err))
	}
	if errors.Is(err, context.Canceled) {
		return classify(contracts.ErrorCodeCanceled, transportCause(err))
	}
	switch kind, ok := a2atransport.FailureKindOf(err); {
	case ok && kind == a2atransport.FailureDeadlineExceeded:
		return classify(contracts.ErrorCodeTimeout, transportCause(err))
	case ok && kind == a2atransport.FailureCanceled:
		return classify(contracts.ErrorCodeCanceled, transportCause(err))
	case ok && kind == a2atransport.FailureRemoteAgent:
		return classify(contracts.ErrorCodeAgentExecutionFailed, transportCause(err))
	case ok && kind == a2atransport.FailureUnavailable:
		return classify(contracts.ErrorCodeAgentUnavailable, transportCause(err))
	case ok && kind == a2atransport.FailureResponseTooLarge:
		return classify(contracts.ErrorCodeAgentResponseTooLarge, transportCause(err))
	case ok && (kind == a2atransport.FailureInvalidArgument || kind == a2atransport.FailureProtocol):
		return classify(contracts.ErrorCodeA2AProtocol, transportCause(err))
	default:
		return classify(contracts.ErrorCodeA2AProtocol, err)
	}
}

func transportCause(err error) error {
	var failure *a2atransport.Failure
	if errors.As(err, &failure) {
		if cause := errors.Unwrap(failure); cause != nil {
			return cause
		}
	}
	return err
}
