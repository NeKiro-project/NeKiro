CREATE TABLE catalog.providers (
    provider_id varchar(128) COLLATE "C" PRIMARY KEY,
    owner_identity varchar(128) COLLATE "C" NOT NULL,
    verification_status varchar(16) NOT NULL,
    verification_method varchar(64) NOT NULL,
    verified_at timestamptz,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT providers_provider_id_format CHECK (provider_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    CONSTRAINT providers_owner_identity_format CHECK (owner_identity ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    CONSTRAINT providers_status CHECK (verification_status IN ('unverified', 'verified', 'suspended')),
    CONSTRAINT providers_method CHECK (verification_method = 'http_well_known'),
    CONSTRAINT providers_state_timestamps CHECK ((verification_status = 'unverified' AND verified_at IS NULL) OR (verification_status = 'verified' AND verified_at IS NOT NULL) OR verification_status = 'suspended')
);

ALTER TABLE catalog.agent_identities
    ADD COLUMN provider_id varchar(128) COLLATE "C",
    ADD CONSTRAINT agent_identities_provider_fk FOREIGN KEY (provider_id) REFERENCES catalog.providers(provider_id);

CREATE INDEX agent_identities_provider_idx ON catalog.agent_identities (provider_id);

CREATE TABLE catalog.endpoint_bindings (
    binding_id varchar(128) COLLATE "C" PRIMARY KEY,
    provider_id varchar(128) COLLATE "C" NOT NULL,
    agent_id varchar(128) COLLATE "C" NOT NULL,
    agent_card_version text COLLATE "C" NOT NULL,
    endpoint text NOT NULL,
    endpoint_origin text NOT NULL,
    endpoint_path text NOT NULL,
    verification_method varchar(64) NOT NULL,
    verification_status varchar(16) NOT NULL,
    verification_evidence_digest bytea,
    verification_failure_code varchar(64),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    verified_at timestamptz,
    revoked_at timestamptz,
    CONSTRAINT endpoint_bindings_provider_fk FOREIGN KEY (provider_id) REFERENCES catalog.providers(provider_id),
    CONSTRAINT endpoint_bindings_agent_version_fk FOREIGN KEY (agent_id, agent_card_version) REFERENCES catalog.agent_versions(agent_id, version),
    CONSTRAINT endpoint_bindings_identifier_format CHECK (binding_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' AND agent_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    CONSTRAINT endpoint_bindings_evidence_digest_length CHECK (verification_evidence_digest IS NULL OR octet_length(verification_evidence_digest) = 32),
    CONSTRAINT endpoint_bindings_status CHECK (verification_status IN ('pending', 'verified', 'failed', 'revoked')),
    CONSTRAINT endpoint_bindings_method CHECK (verification_method = 'http_well_known'),
    CONSTRAINT endpoint_bindings_state_timestamps CHECK (
        (verification_status = 'pending' AND verification_evidence_digest IS NULL AND verification_failure_code IS NULL AND verified_at IS NULL AND revoked_at IS NULL)
        OR (verification_status = 'verified' AND verification_evidence_digest IS NOT NULL AND verification_failure_code IS NULL AND verified_at IS NOT NULL AND revoked_at IS NULL)
        OR (verification_status = 'failed' AND verification_evidence_digest IS NULL AND verification_failure_code IS NOT NULL AND verified_at IS NULL AND revoked_at IS NULL)
        OR (verification_status = 'revoked' AND verification_evidence_digest IS NULL AND verification_failure_code IS NULL AND verified_at IS NULL AND revoked_at IS NOT NULL)
    )
);

CREATE INDEX endpoint_bindings_provider_idx ON catalog.endpoint_bindings (provider_id, agent_id);

CREATE TABLE catalog.verification_challenges (
    challenge_id varchar(128) COLLATE "C" PRIMARY KEY,
    binding_id varchar(128) COLLATE "C" NOT NULL,
    proof_digest bytea NOT NULL,
    expires_at timestamptz NOT NULL,
    used_at timestamptz,
    created_at timestamptz NOT NULL,
    CONSTRAINT verification_challenges_binding_fk FOREIGN KEY (binding_id) REFERENCES catalog.endpoint_bindings(binding_id),
    CONSTRAINT verification_challenges_identifier_format CHECK (challenge_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    CONSTRAINT verification_challenges_proof_digest_length CHECK (octet_length(proof_digest) = 32)
);

CREATE INDEX verification_challenges_binding_idx ON catalog.verification_challenges (binding_id, created_at DESC);

---- create above / drop below ----

DROP TABLE catalog.verification_challenges;
DROP TABLE catalog.endpoint_bindings;
DROP INDEX catalog.agent_identities_provider_idx;
ALTER TABLE catalog.agent_identities DROP CONSTRAINT agent_identities_provider_fk, DROP COLUMN provider_id;
DROP TABLE catalog.providers;
