package configcenter

import (
	"context"
	"errors"
	"testing"
)

func TestErrorClassificationsAndSafeDetails(t *testing.T) {
	key, err := ParseKey("alpha")
	if err != nil {
		t.Fatal(err)
	}
	provider, err := ParseProviderID("file")
	if err != nil {
		t.Fatal(err)
	}
	revision := NewObservationRevision()
	all := []struct {
		code Code
		sent *Error
	}{
		{CodeInvalid, ErrInvalid}, {CodeMissing, ErrMissing}, {CodeUnsafeState, ErrUnsafeState},
		{CodeUnauthorized, ErrUnauthorized}, {CodeUnavailable, ErrUnavailable},
		{CodeUnsupported, ErrUnsupported}, {CodeSubscriptionClosed, ErrSubscriptionClosed},
		{CodeReaderClosed, ErrReaderClosed}, {CodePublisherClosed, ErrPublisherClosed},
		{CodeWatchInterrupted, ErrWatchInterrupted}, {CodePayloadTooLarge, ErrPayloadTooLarge},
		{CodeRevisionDuplicate, ErrRevisionDuplicate}, {CodeRevisionStale, ErrRevisionStale},
		{CodeRevisionGap, ErrRevisionGap}, {CodeRevisionOutOfOrder, ErrRevisionOutOfOrder},
	}
	for _, testCase := range all {
		details := ErrorDetails{Provider: provider, Key: key, Operation: OperationRead, Revision: revision}
		if testCase.code == CodeWatchInterrupted {
			details.CauseKind = CodeUnavailable
		}
		classified := NewError(testCase.code, details)
		if !errors.Is(classified, testCase.sent) {
			t.Errorf("errors.Is(%v, %v) = false", classified, testCase.sent)
		}
		if got, ok := CodeOf(classified); !ok || got != testCase.code {
			t.Errorf("CodeOf(%v) = %q, %v", classified, got, ok)
		}
		if classified.Error() != "config center "+string(testCase.code) {
			t.Errorf("unexpected stable error text %q", classified.Error())
		}
		gotDetails := classified.Details()
		if gotDetails.Provider != provider || gotDetails.Key != key || gotDetails.Operation != OperationRead {
			t.Errorf("safe details changed: %#v", gotDetails)
		}
	}
}

func TestCanceledErrorPreservesOnlyContextCancellation(t *testing.T) {
	key, err := ParseKey("alpha")
	if err != nil {
		t.Fatal(err)
	}
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded} {
		classified := NewCanceledError(ErrorDetails{Key: key, Operation: OperationNext}, cause)
		if !errors.Is(classified, ErrCanceled) || !errors.Is(classified, cause) {
			t.Fatalf("canceled error does not preserve classification/cause: %v", classified)
		}
		if got := classified.Details().CauseKind; got != "" {
			t.Fatalf("canceled error exposed cause kind %q", got)
		}
	}
}

func TestErrorConstructorsRejectInvalidCauseMetadata(t *testing.T) {
	assertPanics(t, func() {
		NewError(CodeUnavailable, ErrorDetails{CauseKind: CodeUnsafeState})
	})
	assertPanics(t, func() {
		NewError(CodeWatchInterrupted, ErrorDetails{CauseKind: CodeMissing})
	})
	assertPanics(t, func() {
		NewCanceledError(ErrorDetails{}, errors.New("implementation detail"))
	})
}

func assertPanics(t *testing.T, callback func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("callback did not panic")
		}
	}()
	callback()
}
