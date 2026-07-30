//go:build integration

package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Nene7ko/NeKiro/apps/control-plane/internal/catalog"
	"github.com/Nene7ko/NeKiro/contracts"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPublicShareStoreProjectsOnlyAuthoritativeReleases(t *testing.T) {
	ctx := context.Background()
	databaseURL := os.Getenv("NEKIRO_TEST_DATABASE_URL")
	if strings.TrimSpace(databaseURL) == "" {
		t.Fatal("NEKIRO_TEST_DATABASE_URL is required for integration tests")
	}
	configuration, err := pgx.ParseConfig(databaseURL)
	if err != nil || !strings.HasSuffix(configuration.Database, "_test") {
		t.Fatal("integration tests require a dedicated _test database")
	}
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close(ctx)
	if _, err := connection.Exec(ctx, `DROP SCHEMA IF EXISTS catalog CASCADE`); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, connection, "up"); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC)
	publicAgentID := "agt_0123456789abcdef0123456789abcdef"
	card := contracts.AgentCard{
		SchemaVersion: "0.2",
		AgentID:       "agent-public",
		Name:          "Public Agent",
		Description:   "A public integration fixture.",
		Owner:         contracts.AgentOwner{ID: "owner-public", DisplayName: "Public Owner"},
		Version:       "1.0.0",
		Protocol:      contracts.AgentProtocol{Type: "a2a", Version: "0.3.0", Transport: "JSONRPC", Endpoint: "https://agent.example/a2a"},
		Skills: []contracts.AgentSkill{{
			ID: "documents.read", Name: "Read documents", Description: "Reads documents.",
			InputSchema: contracts.JSONSchema{"type": "object"}, OutputSchema: contracts.JSONSchema{"type": "object"},
		}},
		Authentication: contracts.AgentAuthentication{Type: "none"},
		Limits:         contracts.AgentLimits{TimeoutMS: 1000, MaxInputBytes: json.Number("1024"), MaxOutputBytes: json.Number("1024")},
	}
	cardJSON, err := json.Marshal(card)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(cardJSON)
	if _, err := connection.Exec(ctx, `
INSERT INTO catalog.agent_identities (agent_id, owner_id, created_at, public_agent_id)
VALUES ('agent-public', 'owner-public', $1, $2)`, now, publicAgentID); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Exec(ctx, `
INSERT INTO catalog.providers (provider_id, owner_identity, verification_status, verification_method, verified_at, created_at, updated_at)
VALUES ('provider-public', 'owner-public', 'verified', 'http_well_known', $1, $1, $1)`, now); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Exec(ctx, `
INSERT INTO catalog.agent_versions (agent_id, version, schema_version, card, card_name, card_description, card_digest, publication_status, registered_at, published_at, publication_sequence, legacy_unverified)
VALUES ('agent-public', '1.0.0', '0.2', $1, $2, $3, $4, 'published', $5, $5, 1, false)`, string(cardJSON), card.Name, card.Description, digest[:], now); err != nil {
		t.Fatal(err)
	}
	evidence := sha256.Sum256([]byte("verified-public"))
	if _, err := connection.Exec(ctx, `
INSERT INTO catalog.endpoint_bindings (binding_id, provider_id, agent_id, agent_card_version, endpoint, endpoint_origin, endpoint_path, verification_method, verification_status, verification_evidence_digest, created_at, updated_at, verified_at)
VALUES ('binding-public', 'provider-public', 'agent-public', '1.0.0', 'https://agent.example/a2a', 'https://agent.example', '/a2a', 'http_well_known', 'verified', $1, $2, $2, $2)`, evidence[:], now); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Exec(ctx, `
INSERT INTO catalog.agent_releases (release_id, provider_id, agent_id, agent_card_version, card_digest, endpoint_binding_id, endpoint_origin, endpoint_path, verification_method, verification_evidence_digest, state, created_at, updated_at, verified_at, published_at)
VALUES ('release-public', 'provider-public', 'agent-public', '1.0.0', $1, 'binding-public', 'https://agent.example', '/a2a', 'http_well_known', $2, 'published', $3, $3, $3, $3)`, digest[:], evidence[:], now); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store, err := NewStore(pool)
	if err != nil {
		t.Fatal(err)
	}

	view, err := store.GetPublicShare(ctx, publicAgentID)
	if err != nil {
		t.Fatal(err)
	}
	if view.PublicAgentID != publicAgentID || !view.RegisteredAt.Equal(now) || len(view.Releases) != 1 {
		t.Fatalf("public view = %#v", view)
	}
	if release := view.Releases[0]; release.ReleaseID != "release-public" || release.AgentID != card.AgentID || release.Name != card.Name || release.CardDigest != digest || !release.PublishedAt.Equal(now) {
		t.Fatalf("public release = %#v", release)
	}

	if _, err := connection.Exec(ctx, `
INSERT INTO catalog.agent_identities (agent_id, owner_id, created_at, public_agent_id)
VALUES ('agent-empty', 'owner-empty', $1, 'agt_abcdefabcdefabcdefabcdefabcdefab')`, now); err != nil {
		t.Fatal(err)
	}
	empty, err := store.GetPublicShare(ctx, "agt_abcdefabcdefabcdefabcdefabcdefab")
	if err != nil || empty.PublicAgentID == "" || len(empty.Releases) != 0 {
		t.Fatalf("empty public view = %#v error=%v", empty, err)
	}
	if _, err := store.GetPublicShare(ctx, "agt_ffffffffffffffffffffffffffffffff"); !errors.Is(err, catalog.ErrNotFound) {
		t.Fatalf("unknown public Agent error = %v", err)
	}
}
