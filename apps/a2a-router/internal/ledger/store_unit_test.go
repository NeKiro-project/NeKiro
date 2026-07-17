package ledger

import (
	"errors"
	"testing"

	"github.com/Nene7ko/NeKiro/contracts"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestOrderTraceProjectionsPlacesParentsBeforeChildren(t *testing.T) {
	ordered, err := orderTraceProjections([]contracts.InvocationRecordV4{
		{InvocationID: "child", ParentInvocationID: "parent"},
		{InvocationID: "parent"},
	})
	if err != nil {
		t.Fatalf("orderTraceProjections() error = %v", err)
	}
	if len(ordered) != 2 || ordered[0].InvocationID != "parent" || ordered[1].InvocationID != "child" {
		t.Fatalf("ordered=%#v", ordered)
	}
}

func TestOrderTraceProjectionsRejectsMissingOrCyclicParents(t *testing.T) {
	for _, test := range []struct {
		name   string
		values []contracts.InvocationRecordV4
	}{
		{name: "missing", values: []contracts.InvocationRecordV4{{InvocationID: "child", ParentInvocationID: "missing"}}},
		{name: "cycle", values: []contracts.InvocationRecordV4{{InvocationID: "a", ParentInvocationID: "b"}, {InvocationID: "b", ParentInvocationID: "a"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := orderTraceProjections(test.values); err == nil {
				t.Fatal("orderTraceProjections() accepted invalid lineage")
			}
		})
	}
}

func TestLedgerStoreHelpersClassifyFactsAndWriteErrors(t *testing.T) {
	if nullableText("") != nil || nullableText("value") != "value" {
		t.Fatal("nullableText() did not preserve empty/non-empty semantics")
	}
	if sameOptionalInt64(nil, nil) != true || sameOptionalInt64(nil, new(int64)) || !sameOptionalInt64(pointerToInt64(4), pointerToInt64(4)) || sameOptionalInt64(pointerToInt64(4), pointerToInt64(5)) {
		t.Fatal("sameOptionalInt64() classification is incorrect")
	}
	event := contracts.InvocationEventV03{}
	if eventErrorCode(event) != "" {
		t.Fatal("eventErrorCode() returned a value for an event without an error")
	}
	for _, test := range []struct {
		code contracts.PlatformErrorCode
		want error
	}{
		{code: "23503", want: ErrConflict},
		{code: "23505", want: ErrConflict},
		{code: "23514", want: ErrValidation},
	} {
		if err := classifyWriteError("write", &pgconn.PgError{Code: string(test.code)}); !errors.Is(err, test.want) {
			t.Fatalf("classifyWriteError(%q)=%v, want %v", test.code, err, test.want)
		}
	}
	if err := classifyWriteError("write", errors.New("database unavailable")); !errors.Is(err, ErrDependency) {
		t.Fatal("non-PostgreSQL write error was not classified as dependency")
	}
}

func pointerToInt64(value int64) *int64 { return &value }
