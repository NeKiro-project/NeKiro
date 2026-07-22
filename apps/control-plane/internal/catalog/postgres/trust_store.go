package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/Nene7ko/NeKiro/apps/control-plane/internal/catalog"
	"github.com/jackc/pgx/v5"
)

func (store *Store) AgentOwner(ctx context.Context, agentID string) (string, error) {
	var owner string
	err := store.pool.QueryRow(ctx, `SELECT owner_id FROM catalog.agent_identities WHERE agent_id = $1`, agentID).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", catalog.ErrNotFound
	}
	if err != nil {
		return "", dependencyError("read Agent owner", err)
	}
	return owner, nil
}

func (store *Store) CreateBinding(ctx context.Context, provider catalog.Provider, binding catalog.EndpointBinding) (result catalog.EndpointBinding, returnErr error) {
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return catalog.EndpointBinding{}, trustDependency("begin endpoint binding", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			returnErr = errors.Join(returnErr, trustDependency("rollback endpoint binding", rollbackErr))
		}
	}()
	if _, err := tx.Exec(ctx, `
INSERT INTO catalog.providers (
    provider_id, owner_identity, verification_status, verification_method,
    verified_at, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (provider_id) DO NOTHING`, provider.ProviderID, provider.OwnerIdentity, provider.VerificationStatus, provider.VerificationMethod, provider.VerifiedAt, provider.CreatedAt, provider.UpdatedAt); err != nil {
		return catalog.EndpointBinding{}, trustDependency("create provider", err)
	}
	var ownerIdentity string
	var providerStatus catalog.VerificationStatus
	if err := tx.QueryRow(ctx, `SELECT owner_identity, verification_status FROM catalog.providers WHERE provider_id = $1 FOR UPDATE`, provider.ProviderID).Scan(&ownerIdentity, &providerStatus); err != nil {
		return catalog.EndpointBinding{}, trustDependency("read provider", err)
	}
	if ownerIdentity != provider.OwnerIdentity {
		return catalog.EndpointBinding{}, catalog.ErrForbidden
	}
	if providerStatus == catalog.VerificationSuspended {
		return catalog.EndpointBinding{}, catalog.ErrForbidden
	}
	var claimedProvider *string
	if err := tx.QueryRow(ctx, `SELECT provider_id FROM catalog.agent_identities WHERE agent_id = $1 FOR UPDATE`, binding.AgentID).Scan(&claimedProvider); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return catalog.EndpointBinding{}, catalog.ErrNotFound
		}
		return catalog.EndpointBinding{}, trustDependency("read Agent ownership", err)
	}
	if claimedProvider != nil && *claimedProvider != provider.ProviderID {
		return catalog.EndpointBinding{}, catalog.ErrTrustConflict
	}
	if claimedProvider == nil {
		if _, err := tx.Exec(ctx, `UPDATE catalog.agent_identities SET provider_id = $2 WHERE agent_id = $1`, binding.AgentID, provider.ProviderID); err != nil {
			return catalog.EndpointBinding{}, trustDependency("claim Agent provider", err)
		}
	}
	row := tx.QueryRow(ctx, `
INSERT INTO catalog.endpoint_bindings (
    binding_id, provider_id, agent_id, agent_card_version, endpoint, endpoint_origin, endpoint_path,
    verification_method, verification_status, verification_evidence_digest,
    verification_failure_code, created_at, updated_at, verified_at, revoked_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NULL,NULL,$10,$11,$12,$13)
RETURNING binding_id, provider_id, agent_id, agent_card_version, endpoint, endpoint_origin,
          endpoint_path, verification_method, verification_status,
          verification_evidence_digest, verification_failure_code, created_at,
	          updated_at, verified_at, revoked_at`, binding.BindingID, binding.ProviderID, binding.AgentID, binding.AgentCardVersion, binding.Endpoint, binding.Origin, binding.Path, binding.VerificationMethod, binding.VerificationStatus, binding.CreatedAt, binding.UpdatedAt, binding.VerifiedAt, binding.RevokedAt)
	result, err = scanBinding(row)
	if err != nil {
		if constraintName(err) != "" {
			return catalog.EndpointBinding{}, catalog.ErrTrustConflict
		}
		return catalog.EndpointBinding{}, trustDependency("create endpoint binding", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return catalog.EndpointBinding{}, trustDependency("commit endpoint binding", err)
	}
	return result, nil
}

func (store *Store) GetBinding(ctx context.Context, providerID, bindingID string) (catalog.EndpointBinding, error) {
	row := store.pool.QueryRow(ctx, bindingSelect+` WHERE provider_id = $1 AND binding_id = $2`, providerID, bindingID)
	binding, err := scanBinding(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return catalog.EndpointBinding{}, catalog.ErrBindingNotFound
	}
	if err != nil {
		return catalog.EndpointBinding{}, trustDependency("read endpoint binding", err)
	}
	return binding, nil
}

func (store *Store) GetProvider(ctx context.Context, providerID string) (catalog.Provider, error) {
	var provider catalog.Provider
	err := store.pool.QueryRow(ctx, `
SELECT provider_id, owner_identity, verification_status, verification_method,
       verified_at, created_at, updated_at
FROM catalog.providers
WHERE provider_id = $1`, providerID).Scan(&provider.ProviderID, &provider.OwnerIdentity, &provider.VerificationStatus, &provider.VerificationMethod, &provider.VerifiedAt, &provider.CreatedAt, &provider.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return catalog.Provider{}, catalog.ErrProviderNotFound
	}
	if err != nil {
		return catalog.Provider{}, trustDependency("read provider", err)
	}
	return provider, nil
}

func (store *Store) CreateChallenge(ctx context.Context, challenge catalog.VerificationChallenge) error {
	_, err := store.pool.Exec(ctx, `
INSERT INTO catalog.verification_challenges (
    challenge_id, binding_id, proof_digest, expires_at, used_at, created_at
) VALUES ($1,$2,$3,$4,$5,$6)`, challenge.ChallengeID, challenge.BindingID, challenge.ProofDigest[:], challenge.ExpiresAt, challenge.UsedAt, challenge.CreatedAt)
	if err != nil {
		if constraintName(err) != "" {
			return catalog.ErrTrustConflict
		}
		return trustDependency("create verification challenge", err)
	}
	return nil
}

func (store *Store) ReserveChallenge(ctx context.Context, bindingID, challengeID string, now time.Time) (result catalog.VerificationChallenge, binding catalog.EndpointBinding, returnErr error) {
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return result, binding, trustDependency("begin challenge reservation", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			returnErr = errors.Join(returnErr, trustDependency("rollback challenge reservation", rollbackErr))
		}
	}()
	var digest []byte
	err = tx.QueryRow(ctx, `
SELECT challenge_id, binding_id, proof_digest, expires_at, used_at,
	   created_at
FROM catalog.verification_challenges
WHERE binding_id = $1 AND challenge_id = $2
	FOR UPDATE`, bindingID, challengeID).Scan(&result.ChallengeID, &result.BindingID, &digest, &result.ExpiresAt, &result.UsedAt, &result.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, binding, catalog.ErrChallengeNotFound
	}
	if err != nil {
		return result, binding, trustDependency("lock verification challenge", err)
	}
	if len(digest) != 32 {
		return result, binding, trustDependency("read verification challenge", errors.New("proof digest length is invalid"))
	}
	copy(result.ProofDigest[:], digest)
	if result.UsedAt != nil {
		return result, binding, catalog.ErrChallengeReused
	}
	if !now.Before(result.ExpiresAt) {
		return result, binding, catalog.ErrChallengeExpired
	}
	binding, err = scanBinding(tx.QueryRow(ctx, bindingSelect+` WHERE binding_id = $1 FOR UPDATE`, bindingID))
	if err != nil {
		return result, binding, trustDependency("read challenge binding", err)
	}
	if binding.VerificationStatus == catalog.VerificationVerified || binding.VerificationStatus == catalog.VerificationRevoked {
		return result, binding, catalog.ErrTrustConflict
	}
	if _, err := tx.Exec(ctx, `UPDATE catalog.verification_challenges SET used_at = $3 WHERE binding_id = $1 AND challenge_id = $2`, bindingID, challengeID, now); err != nil {
		return result, binding, trustDependency("reserve verification challenge", err)
	}
	result.UsedAt = &now
	if err := tx.Commit(ctx); err != nil {
		return result, binding, trustDependency("commit challenge reservation", err)
	}
	return result, binding, nil
}

func (store *Store) SetBindingVerification(ctx context.Context, bindingID string, status catalog.VerificationStatus, failureCode *string, digest *[32]byte, at time.Time) (result catalog.EndpointBinding, returnErr error) {
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return result, trustDependency("begin binding verification", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			returnErr = errors.Join(returnErr, trustDependency("rollback binding verification", rollbackErr))
		}
	}()
	var verifiedAt *time.Time
	if status == catalog.VerificationVerified {
		verifiedAt = &at
	}
	var digestBytes []byte
	if digest != nil {
		digestBytes = digest[:]
	}
	result, err = scanBinding(tx.QueryRow(ctx, `
UPDATE catalog.endpoint_bindings
SET verification_status = $2,
    verification_evidence_digest = $3,
    verification_failure_code = $4,
    verified_at = $5,
    updated_at = $6
WHERE binding_id = $1
  AND verification_status IN ('pending', 'failed')
RETURNING binding_id, provider_id, agent_id, agent_card_version, endpoint, endpoint_origin,
          endpoint_path, verification_method, verification_status,
          verification_evidence_digest, verification_failure_code, created_at,
	          updated_at, verified_at, revoked_at`, bindingID, status, digestBytes, failureCode, verifiedAt, at))
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		if existsErr := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM catalog.endpoint_bindings WHERE binding_id = $1)`, bindingID).Scan(&exists); existsErr != nil {
			return result, trustDependency("check endpoint binding transition", existsErr)
		}
		if exists {
			return result, catalog.ErrTrustConflict
		}
		return result, catalog.ErrBindingNotFound
	}
	if err != nil {
		return result, trustDependency("update endpoint verification", err)
	}
	if status == catalog.VerificationVerified {
		commandTag, err := tx.Exec(ctx, `
UPDATE catalog.providers
SET verification_status = 'verified', verified_at = $2, updated_at = $2
WHERE provider_id = $1 AND verification_status <> 'suspended'`, result.ProviderID, at)
		if err != nil {
			return result, trustDependency("verify provider", err)
		}
		if commandTag.RowsAffected() != 1 {
			return result, catalog.ErrForbidden
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return result, trustDependency("commit binding verification", err)
	}
	return result, nil
}

const bindingSelect = `
SELECT binding_id, provider_id, agent_id, agent_card_version, endpoint, endpoint_origin,
       endpoint_path, verification_method, verification_status,
       verification_evidence_digest, verification_failure_code, created_at,
       updated_at, verified_at, revoked_at
FROM catalog.endpoint_bindings`

func scanBinding(row scanner) (catalog.EndpointBinding, error) {
	var result catalog.EndpointBinding
	var digest []byte
	if err := row.Scan(&result.BindingID, &result.ProviderID, &result.AgentID, &result.AgentCardVersion, &result.Endpoint, &result.Origin, &result.Path, &result.VerificationMethod, &result.VerificationStatus, &digest, &result.VerificationFailureCode, &result.CreatedAt, &result.UpdatedAt, &result.VerifiedAt, &result.RevokedAt); err != nil {
		return catalog.EndpointBinding{}, err
	}
	if len(digest) != 0 && len(digest) != 32 {
		return catalog.EndpointBinding{}, errors.New("endpoint verification digest length is invalid")
	}
	if len(digest) == 32 {
		var fixed [32]byte
		copy(fixed[:], digest)
		result.VerificationEvidenceDigest = &fixed
	}
	return result, nil
}

func trustDependency(operation string, err error) error {
	return errors.Join(catalog.ErrTrustDependency, errors.New(operation+": "+err.Error()))
}
