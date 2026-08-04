package catalog

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/Nene7ko/NeKiro/contracts"
)

type PublicShareStore interface {
	GetPublicShare(context.Context, string) (PublicAgentView, error)
}

type PublicAgentView struct {
	PublicAgentID string
	RegisteredAt  time.Time
	Releases      []PublicAgentRelease
}

type PublicAgentRelease struct {
	ReleaseID          string
	AgentID            string
	Name               string
	Description        string
	Owner              contracts.AgentOwner
	AgentCardVersion   string
	CardDigest         [32]byte
	PublishedAt        time.Time
	AuthenticationType string
	Skills             []contracts.AgentSkill
	Permissions        []contracts.PermissionDeclaration
	Limits             contracts.AgentLimits
}

type PublicShareService struct {
	store  PublicShareStore
	origin string
}

func NewPublicShareService(store PublicShareStore, origin string) (*PublicShareService, error) {
	if store == nil || origin == "" {
		return nil, errors.New("public share dependencies are required")
	}
	return &PublicShareService{store: store, origin: origin}, nil
}

func (service *PublicShareService) Resolve(ctx context.Context, publicAgentID string) (contracts.PublicAgentShare, error) {
	if !ValidPublicAgentID(publicAgentID) {
		return contracts.PublicAgentShare{}, ErrInvalid
	}
	view, err := service.store.GetPublicShare(ctx, publicAgentID)
	if err != nil {
		return contracts.PublicAgentShare{}, err
	}
	if view.PublicAgentID != publicAgentID || !ValidPublicAgentID(view.PublicAgentID) {
		return contracts.PublicAgentShare{}, fmt.Errorf("public identity projection mismatch: %w", ErrDependency)
	}
	result := contracts.PublicAgentShare{
		SchemaVersion: contracts.PublicAgentShareSchemaVersion,
		PublicAgentID: view.PublicAgentID,
		PublicURL:     service.origin + "/a/" + view.PublicAgentID,
		RegisteredAt:  view.RegisteredAt.UTC(),
		Availability:  contracts.PublicAgentAvailabilityNotInstallable,
		Releases:      make([]contracts.PublicAgentRelease, 0, len(view.Releases)),
	}
	for _, release := range view.Releases {
		if release.ReleaseID == "" || release.AgentID == "" || !ValidIdentifier(release.ReleaseID) || !ValidIdentifier(release.AgentID) || release.AgentCardVersion == "" || release.PublishedAt.IsZero() || release.CardDigest == ([32]byte{}) {
			return contracts.PublicAgentShare{}, fmt.Errorf("public Release projection is invalid: %w", ErrDependency)
		}
		result.Releases = append(result.Releases, contracts.PublicAgentRelease{
			ReleaseID: release.ReleaseID, AgentID: release.AgentID, Name: release.Name, Description: release.Description,
			Owner: release.Owner, AgentCardVersion: release.AgentCardVersion, CardDigest: hex.EncodeToString(release.CardDigest[:]),
			PublishedAt: release.PublishedAt.UTC(), AuthenticationType: release.AuthenticationType, Skills: release.Skills,
			Permissions: release.Permissions, Limits: release.Limits,
		})
	}
	if len(result.Releases) > 0 {
		result.Availability = contracts.PublicAgentAvailabilityInstallable
	}
	return result, nil
}
