package catalog

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/NeKiro-project/NeKiro/contracts"
)

type publicShareStoreStub struct {
	view PublicAgentView
	err  error
}

func (store publicShareStoreStub) GetPublicShare(context.Context, string) (PublicAgentView, error) {
	return store.view, store.err
}

func TestPublicShareServiceHidesDraftsAndMapsExactRelease(t *testing.T) {
	registeredAt := time.Date(2026, 7, 30, 1, 0, 0, 0, time.UTC)
	service, err := NewPublicShareService(publicShareStoreStub{view: PublicAgentView{
		PublicAgentID: "agt_0123456789abcdef0123456789abcdef", RegisteredAt: registeredAt,
	}}, "https://agents.nekiro.test")
	if err != nil {
		t.Fatal(err)
	}
	view, err := service.Resolve(context.Background(), "agt_0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	if view.Availability != contracts.PublicAgentAvailabilityNotInstallable || len(view.Releases) != 0 || view.PublicURL != "https://agents.nekiro.test/a/agt_0123456789abcdef0123456789abcdef" {
		t.Fatalf("unpublished view = %#v", view)
	}

	publishedAt := registeredAt.Add(time.Minute)
	var digest [32]byte
	digest[0] = 1
	service, _ = NewPublicShareService(publicShareStoreStub{view: PublicAgentView{
		PublicAgentID: "agt_0123456789abcdef0123456789abcdef", RegisteredAt: registeredAt,
		Releases: []PublicAgentRelease{{ReleaseID: "release-a", AgentID: "agent-a", Name: "Agent A", Description: "Public", Owner: contracts.AgentOwner{ID: "owner-a", DisplayName: "Owner A"}, AgentCardVersion: "1.0.0", CardDigest: digest, PublishedAt: publishedAt, AuthenticationType: "none", Skills: []contracts.AgentSkill{}, Permissions: []contracts.PermissionDeclaration{}}},
	}}, "https://agents.nekiro.test")
	view, err = service.Resolve(context.Background(), "agt_0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	if view.Availability != contracts.PublicAgentAvailabilityInstallable || len(view.Releases) != 1 || view.Releases[0].ReleaseID != "release-a" || view.Releases[0].CardDigest == "" {
		t.Fatalf("published view = %#v", view)
	}
}

func TestPublicShareServicePreservesValidationNotFoundAndDependency(t *testing.T) {
	service, _ := NewPublicShareService(publicShareStoreStub{err: ErrNotFound}, "https://agents.nekiro.test")
	if _, err := service.Resolve(context.Background(), "agt_invalid"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid error = %v", err)
	}
	if _, err := service.Resolve(context.Background(), "agt_0123456789abcdef0123456789abcdef"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("not found error = %v", err)
	}
	service, _ = NewPublicShareService(publicShareStoreStub{err: ErrDependency}, "https://agents.nekiro.test")
	if _, err := service.Resolve(context.Background(), "agt_0123456789abcdef0123456789abcdef"); !errors.Is(err, ErrDependency) {
		t.Fatalf("dependency error = %v", err)
	}
}
