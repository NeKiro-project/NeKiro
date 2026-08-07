package configcenter

import (
	"errors"
	"testing"
)

func TestRevisionOrderingIsScopedAndGapFree(t *testing.T) {
	first := NewObservationRevision()
	second, err := AdvanceRevision(first)
	if err != nil {
		t.Fatal(err)
	}
	third, err := AdvanceRevision(second)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Scoped() || first.Order() != 0 || second.Order() != 1 || third.Order() != 2 {
		t.Fatalf("unexpected local revision sequence: %#v %#v %#v", first, second, third)
	}
	if err := ValidateNextRevision(first, second); err != nil {
		t.Fatalf("valid successor rejected: %v", err)
	}

	cases := []struct {
		name string
		prev Revision
		next Revision
		want Code
	}{
		{"duplicate", first, first, CodeRevisionDuplicate},
		{"stale", second, first, CodeRevisionStale},
		{"gap", first, third, CodeRevisionGap},
		{"different scope", first, NewObservationRevision(), CodeRevisionOutOfOrder},
		{"unscoped previous", UnscopedRevision(), second, CodeRevisionOutOfOrder},
		{"unscoped candidate", first, UnscopedRevision(), CodeRevisionOutOfOrder},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got, ok := CodeOf(ValidateNextRevision(testCase.prev, testCase.next))
			if !ok || got != testCase.want {
				t.Fatalf("classification = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestAdvanceUnscopedRevisionIsExplicitlyRejected(t *testing.T) {
	_, err := AdvanceRevision(UnscopedRevision())
	if !errors.Is(err, ErrRevisionOutOfOrder) {
		t.Fatalf("AdvanceRevision(unscoped) = %v, want out_of_order", err)
	}
}
