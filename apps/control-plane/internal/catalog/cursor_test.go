package catalog

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/Nene7ko/NeKiro/contracts"
)

func TestNormalizeDiscoveryFilter(t *testing.T) {
	filter, err := NormalizeDiscoveryFilter(contracts.SearchAgentsQuery{})
	if err != nil {
		t.Fatalf("normalize omitted filters: %v", err)
	}
	if filter.Limit != contracts.DiscoveryDefaultLimit {
		t.Fatalf("default limit = %d, want %d", filter.Limit, contracts.DiscoveryDefaultLimit)
	}

	query := " document "
	limit := 100
	filter, err = NormalizeDiscoveryFilter(contracts.SearchAgentsQuery{Query: &query, Limit: &limit})
	if err != nil {
		t.Fatalf("normalize literal spaced query: %v", err)
	}
	if *filter.Query != query {
		t.Fatalf("query was changed to %q", *filter.Query)
	}

	for _, invalidLimit := range []int{0, 101} {
		if _, err := NormalizeDiscoveryFilter(contracts.SearchAgentsQuery{Limit: &invalidLimit}); err == nil {
			t.Fatalf("invalid limit %d was accepted", invalidLimit)
		}
	}
	blank := " \t "
	if _, err := NormalizeDiscoveryFilter(contracts.SearchAgentsQuery{Query: &blank}); err == nil {
		t.Fatal("blank query was accepted")
	}
	tooLong := strings.Repeat("界", 257)
	if _, err := NormalizeDiscoveryFilter(contracts.SearchAgentsQuery{Query: &tooLong}); err == nil {
		t.Fatal("query longer than 256 characters was accepted")
	}
}

func TestCursorRoundTripAndFilterBinding(t *testing.T) {
	capability := "document.summarize"
	limit := 2
	filter, err := NormalizeDiscoveryFilter(contracts.SearchAgentsQuery{Capability: &capability, Limit: &limit})
	if err != nil {
		t.Fatal(err)
	}
	position := DiscoveryPosition{
		PublishedAt: time.Date(2026, 7, 14, 1, 2, 3, 4, time.UTC),
		AgentID:     "agent-a",
		Version:     "1.0.0",
	}
	cursor, err := EncodeCursor(filter, 42, position)
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}
	snapshot, decoded, err := DecodeCursor(cursor, filter)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}
	if snapshot != 42 || decoded != position {
		t.Fatalf("decoded cursor = (%d, %#v), want (42, %#v)", snapshot, decoded, position)
	}

	otherCapability := "document.translate"
	otherFilter, _ := NormalizeDiscoveryFilter(contracts.SearchAgentsQuery{Capability: &otherCapability, Limit: &limit})
	if _, _, err := DecodeCursor(cursor, otherFilter); err == nil {
		t.Fatal("filter-mismatched cursor was accepted")
	}
}

func TestCursorRejectsMalformedStrictPayload(t *testing.T) {
	filter, _ := NormalizeDiscoveryFilter(contracts.SearchAgentsQuery{})
	invalid := []string{
		"not-base64",
		base64.RawURLEncoding.EncodeToString([]byte(`{"v":1,"v":1}`)),
		base64.RawURLEncoding.EncodeToString([]byte(`{"v":1,"filter_hash":"00","snapshot_publication_sequence":1,"last_published_at":"2026-07-14T00:00:00Z","last_agent_id":"agent","last_version":"1.0.0","unknown":true}`)),
		base64.RawURLEncoding.EncodeToString([]byte(`{"v":2,"filter_hash":"00","snapshot_publication_sequence":1,"last_published_at":"2026-07-14T00:00:00Z","last_agent_id":"agent","last_version":"1.0.0"}`)),
	}
	for _, cursor := range invalid {
		if _, _, err := DecodeCursor(cursor, filter); err == nil {
			t.Fatalf("invalid cursor %q was accepted", cursor)
		}
	}
}
