package gateway

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestOutcomeErrorsRemainTypedAndCauseSafe(t *testing.T) {
	cases := []struct {
		name     string
		sentinel error
		outcome  Outcome
		helper   func(error) bool
	}{
		{"invalid", ErrInvalid, OutcomeInvalid, IsInvalid},
		{"unsupported", ErrUnsupported, OutcomeUnsupported, IsUnsupported},
		{"unauthorized", ErrUnauthorized, OutcomeUnauthorized, IsUnauthorized},
		{"unavailable", ErrUnavailable, OutcomeUnavailable, IsUnavailable},
		{"rejected", ErrRejected, OutcomeRejected, IsRejected},
		{"not ready", ErrNotReady, OutcomeNotReady, IsNotReady},
		{"not found", ErrNotFound, OutcomeNotFound, IsNotFound},
		{"stale", ErrStale, OutcomeStale, IsStale},
		{"canceled", ErrCanceled, OutcomeCanceled, IsCanceled},
		{"closed", ErrClosed, OutcomeClosed, IsClosed},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := fmt.Errorf("operation context: %w", NewOutcomeError(testCase.outcome, CauseNone))
			if !errors.Is(err, testCase.sentinel) || !testCase.helper(err) {
				t.Fatalf("error %v did not match %v", err, testCase.sentinel)
			}
			if got, ok := OutcomeOf(err); !ok || got != testCase.outcome {
				t.Fatalf("OutcomeOf = %q, %v, want %q, true", got, ok, testCase.outcome)
			}
		})
	}

	unsafe := NewOutcomeError(OutcomeUnavailable, OutcomeCause("token=secret"))
	if !errors.Is(unsafe, ErrInvalid) || unsafe.Cause() != CauseUnknownCause || strings.Contains(unsafe.Error(), "secret") {
		t.Fatalf("unsafe cause = %v, cause %q; secret leaked", unsafe, unsafe.Cause())
	}
	wrongPair := NewOutcomeError(OutcomeClosed, CauseProviderUnavailable)
	if !errors.Is(wrongPair, ErrInvalid) || wrongPair.Cause() != CauseUnknownCause {
		t.Fatalf("invalid outcome/cause pair = %v (%q)", wrongPair, wrongPair.Cause())
	}
	var nilOutcome *OutcomeError
	if errors.Is(ErrInvalid, nilOutcome) {
		t.Fatal("typed nil outcome unexpectedly matched")
	}
}

func TestCanceledErrorPreservesOnlyLocalContextCause(t *testing.T) {
	err := NewCanceledError(context.DeadlineExceeded)
	if !errors.Is(err, ErrCanceled) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("canceled error = %v, want canceled/deadline", err)
	}
	if strings.Contains(err.Error(), "deadline exceeded") {
		t.Fatalf("context text leaked into public error: %v", err)
	}
	unsafe := NewCanceledError(fmt.Errorf("provider token=secret: %w", context.Canceled))
	if !errors.Is(unsafe, ErrInvalid) || strings.Contains(unsafe.Error(), "secret") || errors.Is(unsafe, context.Canceled) {
		t.Fatalf("unsafe cancellation cause = %v", unsafe)
	}
}
