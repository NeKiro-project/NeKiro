package contracts

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func TestPublicAgentShareOpenAPIAndSchemaMapping(t *testing.T) {
	document := loadOpenAPIDocument(t, filepath.Join("openapi", "public-agent-share.v1.yaml"))
	operation := document.Paths.Find("/v4/public/agents/{publicAgentId}").Get
	if operation == nil || operation.Security == nil || len(*operation.Security) != 0 {
		t.Fatal("public resolution must be explicitly anonymous")
	}
	for _, status := range []int{200, 400, 404, 503, 500} {
		response := operation.Responses.Status(status)
		if response == nil || response.Value == nil || response.Value.Headers["x-nek-trace-id"] == nil {
			t.Fatalf("response %d is missing trace correlation", status)
		}
	}
	now := time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC)
	share := PublicAgentShare{
		SchemaVersion: PublicAgentShareSchemaVersion,
		PublicAgentID: "agt_0123456789abcdef0123456789abcdef",
		PublicURL:     "https://agents.nekiro.dev/a/agt_0123456789abcdef0123456789abcdef",
		RegisteredAt:  now, Availability: PublicAgentAvailabilityInstallable,
		Releases: []PublicAgentRelease{{
			ReleaseID: "rel_runtime_a_1", AgentID: "runtime-a", Name: "Runtime A", Description: "Public sample",
			Owner: AgentOwner{ID: "provider-a", DisplayName: "Provider A"}, AgentCardVersion: "1.0.0",
			CardDigest: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", PublishedAt: now,
			AuthenticationType: "none", Skills: []AgentSkill{{ID: "runtime.echo", Name: "Echo", Description: "Echoes input", InputSchema: JSONSchema{"type": "object"}, OutputSchema: JSONSchema{"type": "object"}, RequiredPermissions: []string{}}},
			Permissions: []PermissionDeclaration{}, Limits: AgentLimits{TimeoutMS: 1000, MaxInputBytes: json.Number("1024"), MaxOutputBytes: json.Number("1024")},
		}},
	}
	validateOpenAPIValue(t, operation.Responses.Status(200).Value.Content["application/json"].Schema, share)
}

func TestCatalogEntryPublicIdentityFieldsArePairedInOpenAPI(t *testing.T) {
	document := loadOpenAPIDocument(t, filepath.Join("openapi", "control-plane.v3.yaml"))
	schema := document.Components.Schemas["CatalogEntry"].Value
	if len(schema.DependentRequired) != 2 {
		t.Fatal("CatalogEntry public identity fields must be paired")
	}
}
