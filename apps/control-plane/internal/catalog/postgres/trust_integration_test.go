//go:build integration

package postgres

import (
	"context"
	"crypto/sha256"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Nene7ko/NeKiro/apps/control-plane/internal/catalog"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTrustedPublicationStorePersistsSingleUseVerification(t *testing.T) {
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
	if _, err := connection.Exec(ctx, `DROP SCHEMA IF EXISTS catalog CASCADE`); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, connection, "up"); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Exec(ctx, `INSERT INTO catalog.agent_identities (agent_id, owner_id, created_at) VALUES ('agent-trust', 'provider-trust', now())`); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Exec(ctx, `INSERT INTO catalog.agent_versions (agent_id, version, schema_version, card, card_name, card_description, card_digest, publication_status, registered_at) VALUES ('agent-trust', '1.0.0', '0.2', '{}', 'Trust Agent', 'Trust Agent', $1, 'draft', now())`, make([]byte, 32)); err != nil {
		t.Fatal(err)
	}
	if err := connection.Close(ctx); err != nil {
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
	now := time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC)
	binding, err := store.CreateBinding(ctx, catalog.Provider{ProviderID: "provider-trust", OwnerIdentity: "provider-trust", VerificationStatus: catalog.VerificationUnverified, VerificationMethod: catalog.VerificationMethodHTTPWellKnown, CreatedAt: now, UpdatedAt: now}, catalog.EndpointBinding{BindingID: "binding-trust", ProviderID: "provider-trust", AgentID: "agent-trust", AgentCardVersion: "1.0.0", Endpoint: "https://agent.example/a2a", Origin: "https://agent.example", Path: "/a2a", VerificationMethod: catalog.VerificationMethodHTTPWellKnown, VerificationStatus: catalog.VerificationPending, CreatedAt: now, UpdatedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	proofDigest := sha256.Sum256([]byte("proof"))
	challenge := catalog.VerificationChallenge{ChallengeID: "challenge-trust", BindingID: binding.BindingID, ProofDigest: proofDigest, ExpiresAt: now.Add(time.Minute), CreatedAt: now}
	if err := store.CreateChallenge(ctx, challenge); err != nil {
		t.Fatal(err)
	}
	reserved, _, err := store.ReserveChallenge(ctx, binding.BindingID, challenge.ChallengeID, now.Add(time.Second))
	if err != nil || reserved.UsedAt == nil {
		t.Fatalf("reserve challenge=%#v error=%v", reserved, err)
	}
	if _, _, err := store.ReserveChallenge(ctx, binding.BindingID, challenge.ChallengeID, now.Add(2*time.Second)); err != catalog.ErrChallengeReused {
		t.Fatalf("second reservation error=%v", err)
	}
	verified, err := store.SetBindingVerification(ctx, binding.BindingID, catalog.VerificationVerified, nil, &proofDigest, now.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if verified.VerificationStatus != catalog.VerificationVerified || verified.VerifiedAt == nil || verified.VerificationEvidenceDigest == nil || *verified.VerificationEvidenceDigest != proofDigest {
		t.Fatalf("verified binding=%#v", verified)
	}
	failure := "wrong_proof"
	if _, err := store.SetBindingVerification(ctx, binding.BindingID, catalog.VerificationFailed, &failure, nil, now.Add(3*time.Second)); err != catalog.ErrTrustConflict {
		t.Fatalf("verified binding overwrite error=%v", err)
	}
}
