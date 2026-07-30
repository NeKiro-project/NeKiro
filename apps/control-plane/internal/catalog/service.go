package catalog

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	semver "github.com/Masterminds/semver/v3"
	"github.com/Nene7ko/NeKiro/contracts"
)

type Clock func() time.Time
type PublicAgentIDGenerator func() (string, error)

type Service struct {
	store            Store
	validator        *contracts.Validator
	clock            Clock
	publicOrigin     string
	newPublicAgentID PublicAgentIDGenerator
}

func NewService(store Store, validator *contracts.Validator, clock Clock) (*Service, error) {
	return NewServiceWithPublicConfig(store, validator, clock, "", NewPublicAgentID)
}

func NewServiceWithPublicConfig(store Store, validator *contracts.Validator, clock Clock, publicOrigin string, generator PublicAgentIDGenerator) (*Service, error) {
	if store == nil || validator == nil || clock == nil || generator == nil {
		return nil, errors.New("catalog service dependencies are required")
	}
	return &Service{store: store, validator: validator, clock: clock, publicOrigin: publicOrigin, newPublicAgentID: generator}, nil
}

func NewPublicAgentID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate public Agent ID: %w", ErrDependency)
	}
	return "agt_" + hex.EncodeToString(raw[:]), nil
}

func (service *Service) Register(ctx context.Context, caller AuthenticatedCaller, requestJSON []byte) (contracts.CatalogEntry, error) {
	request, err := service.validator.DecodeRegisterAgentRequest(requestJSON)
	if err != nil {
		return contracts.CatalogEntry{}, ErrInvalid
	}
	if caller.ID != request.Card.Owner.ID {
		return contracts.CatalogEntry{}, ErrForbidden
	}
	cardJSON, err := json.Marshal(request.Card)
	if err != nil {
		return contracts.CatalogEntry{}, fmt.Errorf("canonicalize Agent Card: %w", ErrDependency)
	}
	digest := sha256.Sum256(cardJSON)
	publicAgentID, err := service.newPublicAgentID()
	if err != nil || !ValidPublicAgentID(publicAgentID) {
		if err != nil {
			return contracts.CatalogEntry{}, err
		}
		return contracts.CatalogEntry{}, fmt.Errorf("generated public Agent ID is invalid: %w", ErrDependency)
	}
	registered, err := service.store.Register(ctx, AgentVersion{
		PublicAgentID: publicAgentID,
		Card:          request.Card,
		CardJSON:      cardJSON,
		CardDigest:    digest,
		Status:        PublicationDraft,
		RegisteredAt:  service.clock().UTC(),
	})
	if err != nil {
		return contracts.CatalogEntry{}, err
	}
	return service.catalogEntry(registered), nil
}

func (service *Service) Get(ctx context.Context, caller AuthenticatedCaller, agentID, version string) (contracts.CatalogEntry, error) {
	if !validAgentVersionIdentity(agentID, version) {
		return contracts.CatalogEntry{}, ErrInvalid
	}
	entry, err := service.store.Get(ctx, agentID, version)
	if err != nil {
		return contracts.CatalogEntry{}, err
	}
	if entry.Status != PublicationPublished && caller.ID != entry.Card.Owner.ID {
		return contracts.CatalogEntry{}, ErrForbidden
	}
	return service.catalogEntry(entry), nil
}

// GetVersion is the controlled Catalog read used by Workspace and future
// dispatch code. It returns the current exact Registry state without applying
// Northbound owner visibility rules.
func (service *Service) GetVersion(ctx context.Context, agentID, version string) (AgentVersion, error) {
	if !validAgentVersionIdentity(agentID, version) {
		return AgentVersion{}, ErrInvalid
	}
	return service.store.Get(ctx, agentID, version)
}

// SelectInstallable applies Catalog-owned SemVer selection and trust
// eligibility, then returns the exact Card/Release facts selected at this
// call's linearization point. It never silently selects an alternate version
// after the highest matching version fails the release gate.
func (service *Service) SelectInstallable(ctx context.Context, agentID, constraint string) (AgentVersion, error) {
	if !ValidIdentifier(agentID) {
		return AgentVersion{}, ErrInvalid
	}
	parsedConstraint, err := semver.NewConstraint(constraint)
	if err != nil || strings.TrimSpace(constraint) == "" {
		return AgentVersion{}, ErrInvalid
	}
	candidates, err := service.store.InstallationCandidates(ctx, agentID)
	if err != nil {
		return AgentVersion{}, err
	}
	var selected AgentVersion
	var selectedVersion *semver.Version
	for _, candidate := range candidates {
		parsedVersion, err := semver.StrictNewVersion(candidate.Card.Version)
		if err != nil {
			return AgentVersion{}, fmt.Errorf("parse stored Agent version: %w", ErrDependency)
		}
		if !parsedConstraint.Check(parsedVersion) || !constraintBranchAllowsVersion(constraint, parsedVersion) {
			continue
		}
		if selectedVersion == nil || parsedVersion.GreaterThan(selectedVersion) ||
			(parsedVersion.Equal(selectedVersion) && candidate.Card.Version > selected.Card.Version) {
			selected = candidate
			selectedVersion = parsedVersion
		}
	}
	if selectedVersion == nil {
		return AgentVersion{}, ErrNotFound
	}
	if err := installabilityError(selected); err != nil {
		return AgentVersion{}, err
	}
	return selected, nil
}

func installabilityError(candidate AgentVersion) error {
	if candidate.Release == nil {
		if candidate.LegacyUnverified && candidate.Status == PublicationPublished {
			return nil
		}
		return ErrReleaseUnpublished
	}
	release := candidate.Release
	if release.ReleaseID == "" || release.AgentID != candidate.Card.AgentID ||
		release.AgentCardVersion != candidate.Card.Version || release.CardDigest != candidate.CardDigest {
		return ErrDependency
	}
	switch release.State {
	case ReleasePublished:
		if candidate.Status != PublicationPublished {
			return ErrDependency
		}
		return nil
	case ReleaseSuspended:
		return ErrReleaseSuspended
	case ReleaseRevoked:
		return ErrReleaseRevoked
	default:
		return ErrReleaseUnpublished
	}
}

var prereleaseComparator = regexp.MustCompile(`(?:^|[<>=~^*xX\s])v?\d+\.\d+\.\d+-[0-9A-Za-z-]+`)

func constraintBranchAllowsVersion(constraint string, version *semver.Version) bool {
	if version.Prerelease() == "" {
		return true
	}
	for _, branch := range strings.Split(constraint, "||") {
		branch = strings.TrimSpace(branch)
		parsed, err := semver.NewConstraint(branch)
		if err == nil && parsed.Check(version) && prereleaseComparator.MatchString(branch) {
			return true
		}
	}
	return false
}

func (service *Service) Publish(ctx context.Context, caller AuthenticatedCaller, agentID, version string) (contracts.CatalogEntry, error) {
	if !validAgentVersionIdentity(agentID, version) {
		return contracts.CatalogEntry{}, ErrInvalid
	}
	entry, err := service.store.Publish(ctx, agentID, version, caller.ID, service.clock().UTC())
	if err != nil {
		return contracts.CatalogEntry{}, err
	}
	return service.catalogEntry(entry), nil
}

func (service *Service) Disable(ctx context.Context, caller AuthenticatedCaller, agentID, version string) (contracts.CatalogEntry, error) {
	if !validAgentVersionIdentity(agentID, version) {
		return contracts.CatalogEntry{}, ErrInvalid
	}
	entry, err := service.store.Disable(ctx, agentID, version, caller.ID, service.clock().UTC())
	if err != nil {
		return contracts.CatalogEntry{}, err
	}
	return service.catalogEntry(entry), nil
}

func (service *Service) Search(ctx context.Context, query contracts.SearchAgentsQuery) (SearchResult, error) {
	filter, err := NormalizeDiscoveryFilter(query)
	if err != nil {
		return SearchResult{}, err
	}
	var snapshot int64
	var after *DiscoveryPosition
	if query.Cursor == nil {
		var firstPage DiscoveryResult
		snapshot, firstPage, err = service.store.DiscoverFirstPage(ctx, filter)
		if err != nil {
			return SearchResult{}, err
		}
		return service.buildSearchResult(filter, snapshot, firstPage)
	} else {
		var position DiscoveryPosition
		snapshot, position, err = DecodeCursor(*query.Cursor, filter)
		if err != nil {
			return SearchResult{}, err
		}
		after = &position
	}
	result, err := service.store.Discover(ctx, DiscoveryQuery{
		Filter: filter, SnapshotPublicationSequence: snapshot, After: after,
	})
	if err != nil {
		return SearchResult{}, err
	}
	return service.buildSearchResult(filter, snapshot, result)
}

func (service *Service) buildSearchResult(filter DiscoveryFilter, snapshot int64, result DiscoveryResult) (SearchResult, error) {
	entries := make([]contracts.CatalogEntry, 0, len(result.Versions))
	for _, version := range result.Versions {
		entries = append(entries, service.catalogEntry(version))
	}
	response := SearchResult{Entries: entries}
	if result.HasMore && len(result.Versions) > 0 {
		last := result.Versions[len(result.Versions)-1]
		if last.PublishedAt == nil {
			return SearchResult{}, fmt.Errorf("published discovery row has no timestamp: %w", ErrDependency)
		}
		cursor, err := EncodeCursor(filter, snapshot, DiscoveryPosition{
			PublishedAt: *last.PublishedAt,
			AgentID:     last.Card.AgentID,
			Version:     last.Card.Version,
		})
		if err != nil {
			return SearchResult{}, fmt.Errorf("encode next cursor: %w", ErrDependency)
		}
		response.NextCursor = &cursor
	}
	return response, nil
}

func (service *Service) catalogEntry(version AgentVersion) contracts.CatalogEntry {
	entry := version.CatalogEntry()
	if version.PublicAgentID != "" && service.publicOrigin != "" {
		entry.PublicURL = service.publicOrigin + "/a/" + version.PublicAgentID
	}
	return entry
}

func validAgentVersionIdentity(agentID, version string) bool {
	if !ValidIdentifier(agentID) {
		return false
	}
	_, err := semver.StrictNewVersion(version)
	return err == nil
}
