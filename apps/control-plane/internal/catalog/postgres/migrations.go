package postgres

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/tern/v2/migrate"
)

const ExpectedSchemaVersion int32 = 5

var ErrSchemaVersionMismatch = errors.New("catalog schema version mismatch")

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

func loadMigrationFiles() (fs.FS, error) {
	migrationFiles, err := fs.Sub(embeddedMigrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("open embedded catalog migrations: %w", err)
	}
	return migrationFiles, nil
}

type RowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func Migrate(ctx context.Context, conn *pgx.Conn, direction string) error {
	if direction != "up" {
		return fmt.Errorf("catalog migration direction %q is unsupported", direction)
	}

	if _, err := conn.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS catalog`); err != nil {
		return fmt.Errorf("create catalog migration schema: %w", err)
	}

	migrationFiles, err := loadMigrationFiles()
	if err != nil {
		return err
	}
	migrator, err := migrate.NewMigrator(ctx, conn, "catalog.schema_version")
	if err != nil {
		return fmt.Errorf("initialize catalog migrator: %w", err)
	}
	if err := migrator.LoadMigrations(migrationFiles); err != nil {
		return fmt.Errorf("load embedded catalog migrations: %w", err)
	}
	if len(migrator.Migrations) != int(ExpectedSchemaVersion) {
		return fmt.Errorf("embedded catalog migration count: %w", ErrSchemaVersionMismatch)
	}

	if err := migrator.Migrate(ctx); err != nil {
		return fmt.Errorf("migrate catalog up: %w", err)
	}
	return nil
}

func CheckSchema(ctx context.Context, db RowQuerier) error {
	var version int32
	var identitiesPresent bool
	var clockPresent bool
	var clockReady bool
	var versionsPresent bool
	var capabilitiesPresent bool
	var cardTextReady bool
	var cardNameReady bool
	var cardDescriptionReady bool
	var legacyUnverifiedReady bool
	var providersPresent bool
	var bindingsPresent bool
	var challengesPresent bool
	var trustColumnsReady bool
	var trustForeignKeysReady bool
	var trustStatusChecksReady bool
	var trustDigestChecksReady bool
	var releasesPresent bool
	var releaseColumnsReady bool
	var releaseForeignKeysReady bool
	var releaseChecksReady bool
	var releaseIndexesReady bool
	var releaseImmutableTrigger bool
	var publicIdentityColumnReady bool
	var publicIdentityIndexReady bool
	var publicIdentityTriggerReady bool
	err := db.QueryRow(ctx, `
WITH required_columns(table_name, column_name, is_nullable, data_type) AS (VALUES
    ('agent_identities', 'provider_id', 'YES', 'character varying'),
    ('agent_identities', 'public_agent_id', 'NO', 'character varying'),
    ('providers', 'provider_id', 'NO', 'character varying'),
    ('providers', 'owner_identity', 'NO', 'character varying'),
    ('providers', 'verification_status', 'NO', 'character varying'),
    ('providers', 'verification_method', 'NO', 'character varying'),
    ('providers', 'verified_at', 'YES', 'timestamp with time zone'),
    ('providers', 'created_at', 'NO', 'timestamp with time zone'),
    ('providers', 'updated_at', 'NO', 'timestamp with time zone'),
    ('endpoint_bindings', 'binding_id', 'NO', 'character varying'),
    ('endpoint_bindings', 'provider_id', 'NO', 'character varying'),
    ('endpoint_bindings', 'agent_id', 'NO', 'character varying'),
    ('endpoint_bindings', 'agent_card_version', 'NO', 'text'),
    ('endpoint_bindings', 'endpoint', 'NO', 'text'),
    ('endpoint_bindings', 'endpoint_origin', 'NO', 'text'),
    ('endpoint_bindings', 'endpoint_path', 'NO', 'text'),
    ('endpoint_bindings', 'verification_method', 'NO', 'character varying'),
    ('endpoint_bindings', 'verification_status', 'NO', 'character varying'),
    ('endpoint_bindings', 'verification_evidence_digest', 'YES', 'bytea'),
    ('endpoint_bindings', 'verification_failure_code', 'YES', 'character varying'),
    ('endpoint_bindings', 'created_at', 'NO', 'timestamp with time zone'),
    ('endpoint_bindings', 'updated_at', 'NO', 'timestamp with time zone'),
    ('endpoint_bindings', 'verified_at', 'YES', 'timestamp with time zone'),
    ('endpoint_bindings', 'revoked_at', 'YES', 'timestamp with time zone'),
    ('verification_challenges', 'challenge_id', 'NO', 'character varying'),
    ('verification_challenges', 'binding_id', 'NO', 'character varying'),
    ('verification_challenges', 'proof_digest', 'NO', 'bytea'),
    ('verification_challenges', 'expires_at', 'NO', 'timestamp with time zone'),
    ('verification_challenges', 'used_at', 'YES', 'timestamp with time zone'),
    ('verification_challenges', 'created_at', 'NO', 'timestamp with time zone'),
    ('agent_versions', 'legacy_unverified', 'NO', 'boolean')
)
SELECT version,
       to_regclass('catalog.agent_identities') IS NOT NULL,
       to_regclass('catalog.publication_clock') IS NOT NULL,
       to_regclass('catalog.agent_versions') IS NOT NULL,
       to_regclass('catalog.agent_version_capabilities') IS NOT NULL,
       (SELECT count(*) = 1
         FROM catalog.publication_clock
         WHERE singleton = true AND last_sequence >= 0),
       (SELECT count(*) = 1
        FROM information_schema.columns
        WHERE table_schema = 'catalog'
          AND table_name = 'agent_versions'
          AND column_name = 'card'
          AND data_type = 'text'
          AND is_nullable = 'NO'),
       (SELECT count(*) = 1
        FROM information_schema.columns
        WHERE table_schema = 'catalog'
          AND table_name = 'agent_versions'
          AND column_name = 'card_name'
          AND data_type = 'text'
          AND is_nullable = 'NO'),
       (SELECT count(*) = 1
        FROM information_schema.columns
        WHERE table_schema = 'catalog'
          AND table_name = 'agent_versions'
          AND column_name = 'card_description'
          AND data_type = 'text'
          AND is_nullable = 'NO'),
       (SELECT count(*) = 1
        FROM information_schema.columns
        WHERE table_schema = 'catalog'
          AND table_name = 'agent_versions'
          AND column_name = 'legacy_unverified'
          AND data_type = 'boolean'
          AND is_nullable = 'NO'),
       to_regclass('catalog.providers') IS NOT NULL,
       to_regclass('catalog.endpoint_bindings') IS NOT NULL,
       to_regclass('catalog.verification_challenges') IS NOT NULL,
       (SELECT count(*) = 31
        FROM required_columns expected
        JOIN information_schema.columns actual
          ON actual.table_schema = 'catalog'
         AND actual.table_name = expected.table_name
         AND actual.column_name = expected.column_name
         AND actual.is_nullable = expected.is_nullable
         AND actual.data_type = expected.data_type),
       (SELECT count(*) = 4
        FROM pg_constraint constraint_row
        JOIN pg_class relation ON relation.oid = constraint_row.conrelid
        JOIN pg_namespace namespace_row ON namespace_row.oid = relation.relnamespace
        WHERE namespace_row.nspname = 'catalog'
          AND constraint_row.conname IN ('agent_identities_provider_fk', 'endpoint_bindings_provider_fk', 'endpoint_bindings_agent_version_fk', 'verification_challenges_binding_fk')
          AND constraint_row.contype = 'f'
          AND constraint_row.convalidated),
       (SELECT count(*) = 12
        FROM pg_constraint constraint_row
        JOIN pg_class relation ON relation.oid = constraint_row.conrelid
        JOIN pg_namespace namespace_row ON namespace_row.oid = relation.relnamespace
        WHERE namespace_row.nspname = 'catalog'
          AND constraint_row.conname IN ('providers_provider_id_format', 'providers_owner_identity_format', 'providers_status', 'providers_method', 'providers_state_timestamps', 'endpoint_bindings_identifier_format', 'endpoint_bindings_evidence_digest_length', 'endpoint_bindings_status', 'endpoint_bindings_method', 'endpoint_bindings_state_timestamps', 'verification_challenges_identifier_format', 'verification_challenges_proof_digest_length')
          AND constraint_row.contype = 'c'
          AND constraint_row.convalidated),
       (SELECT count(*) = 2
        FROM pg_constraint constraint_row
        JOIN pg_class relation ON relation.oid = constraint_row.conrelid
        JOIN pg_namespace namespace_row ON namespace_row.oid = relation.relnamespace
        WHERE namespace_row.nspname = 'catalog'
          AND constraint_row.conname IN ('endpoint_bindings_evidence_digest_length', 'verification_challenges_proof_digest_length')
          AND constraint_row.contype = 'c'
          AND constraint_row.convalidated),
       EXISTS (
           SELECT 1 FROM information_schema.columns
           WHERE table_schema = 'catalog' AND table_name = 'agent_identities'
             AND column_name = 'public_agent_id' AND is_nullable = 'NO' AND data_type = 'character varying'
       ),
       to_regclass('catalog.agent_identities_public_agent_id_idx') IS NOT NULL,
       EXISTS (
           SELECT 1 FROM pg_trigger trigger_row
           JOIN pg_class relation ON relation.oid = trigger_row.tgrelid
           WHERE relation.oid = to_regclass('catalog.agent_identities')
             AND trigger_row.tgname = 'agent_identities_public_agent_id_immutable'
             AND trigger_row.tgenabled = 'O' AND NOT trigger_row.tgisinternal
       )
FROM catalog.schema_version`).Scan(
		&version,
		&identitiesPresent,
		&clockPresent,
		&versionsPresent,
		&capabilitiesPresent,
		&clockReady,
		&cardTextReady,
		&cardNameReady,
		&cardDescriptionReady,
		&legacyUnverifiedReady,
		&providersPresent,
		&bindingsPresent,
		&challengesPresent,
		&trustColumnsReady,
		&trustForeignKeysReady,
		&trustStatusChecksReady,
		&trustDigestChecksReady,
		&publicIdentityColumnReady,
		&publicIdentityIndexReady,
		&publicIdentityTriggerReady,
	)
	if err != nil {
		return fmt.Errorf("read catalog schema version: %w", err)
	}
	if err := db.QueryRow(ctx, `
WITH required_release_columns(column_name, is_nullable, data_type) AS (VALUES
    ('release_id', 'NO', 'character varying'),
    ('provider_id', 'NO', 'character varying'),
    ('agent_id', 'NO', 'character varying'),
    ('agent_card_version', 'NO', 'text'),
    ('card_digest', 'NO', 'bytea'),
    ('endpoint_binding_id', 'NO', 'character varying'),
    ('endpoint_origin', 'NO', 'text'),
    ('endpoint_path', 'NO', 'text'),
    ('verification_method', 'NO', 'character varying'),
    ('verification_evidence_digest', 'YES', 'bytea'),
    ('state', 'NO', 'character varying'),
    ('created_at', 'NO', 'timestamp with time zone'),
    ('updated_at', 'NO', 'timestamp with time zone'),
    ('verified_at', 'YES', 'timestamp with time zone'),
    ('published_at', 'YES', 'timestamp with time zone'),
    ('suspended_at', 'YES', 'timestamp with time zone'),
    ('revoked_at', 'YES', 'timestamp with time zone')
)
SELECT to_regclass('catalog.agent_releases') IS NOT NULL,
       (SELECT count(*) = 17
        FROM required_release_columns expected
        JOIN information_schema.columns actual
          ON actual.table_schema = 'catalog'
         AND actual.table_name = 'agent_releases'
         AND actual.column_name = expected.column_name
         AND actual.is_nullable = expected.is_nullable
         AND actual.data_type = expected.data_type),
       (SELECT count(*) = 3
        FROM pg_constraint constraint_row
        JOIN pg_class relation ON relation.oid = constraint_row.conrelid
        JOIN pg_namespace namespace_row ON namespace_row.oid = relation.relnamespace
        WHERE namespace_row.nspname = 'catalog'
          AND relation.relname = 'agent_releases'
          AND constraint_row.conname IN ('agent_releases_provider_fk', 'agent_releases_card_fk', 'agent_releases_binding_fk')
          AND constraint_row.contype = 'f' AND constraint_row.convalidated),
       (SELECT count(*) = 9
        FROM pg_constraint constraint_row
        JOIN pg_class relation ON relation.oid = constraint_row.conrelid
        JOIN pg_namespace namespace_row ON namespace_row.oid = relation.relnamespace
        WHERE namespace_row.nspname = 'catalog'
          AND relation.relname = 'agent_releases'
          AND constraint_row.conname IN ('agent_releases_release_id_format', 'agent_releases_provider_id_format', 'agent_releases_agent_id_format', 'agent_releases_card_digest_length', 'agent_releases_evidence_digest_length', 'agent_releases_state', 'agent_releases_method', 'agent_releases_timestamp_order', 'agent_releases_state_timestamps')
          AND constraint_row.contype = 'c' AND constraint_row.convalidated),
       to_regclass('catalog.agent_releases_agent_version_idx') IS NOT NULL
       AND to_regclass('catalog.agent_releases_provider_state_idx') IS NOT NULL,
       EXISTS (
           SELECT 1 FROM pg_trigger trigger_row
           JOIN pg_class relation ON relation.oid = trigger_row.tgrelid
           WHERE relation.oid = to_regclass('catalog.agent_releases')
             AND trigger_row.tgname = 'agent_releases_bound_immutable'
             AND trigger_row.tgenabled = 'O' AND NOT trigger_row.tgisinternal
       )`).Scan(
		&releasesPresent, &releaseColumnsReady, &releaseForeignKeysReady,
		&releaseChecksReady, &releaseIndexesReady, &releaseImmutableTrigger,
	); err != nil {
		return fmt.Errorf("read Agent Release schema: %w", err)
	}
	if version != ExpectedSchemaVersion || !identitiesPresent || !clockPresent || !versionsPresent || !capabilitiesPresent || !clockReady || !cardTextReady || !cardNameReady || !cardDescriptionReady || !legacyUnverifiedReady || !providersPresent || !bindingsPresent || !challengesPresent || !trustColumnsReady || !trustForeignKeysReady || !trustStatusChecksReady || !trustDigestChecksReady || !publicIdentityColumnReady || !publicIdentityIndexReady || !publicIdentityTriggerReady || !releasesPresent || !releaseColumnsReady || !releaseForeignKeysReady || !releaseChecksReady || !releaseIndexesReady || !releaseImmutableTrigger {
		return ErrSchemaVersionMismatch
	}
	return nil
}
