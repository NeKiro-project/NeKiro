package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestRuntimeContractOpenAPIDirectionsAndVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path       string
		version    string
		route      string
		security   string
		serverPart string
	}{
		{"openapi/control-plane-invocation.v4.yaml", "4.0.0", "/v4/workspaces/{workspaceId}/invocations", "bearerAuth", "api.nekiro.dev"},
		{"openapi/router-internal.v3.yaml", "3.0.0", "/internal/v3/invocations", "serviceBearerAuth", "a2a-router.internal"},
		{"openapi/router-agent.v1.yaml", "1.0.0", "/agent/v1/invocations", "agentBearerAuth", "a2a-router.agent"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.path, func(t *testing.T) {
			document := loadOpenAPIDocument(t, filepath.FromSlash(test.path))
			if document.Info.Version != test.version {
				t.Fatalf("version = %q, want %q", document.Info.Version, test.version)
			}
			operation := document.Paths.Find(test.route).Post
			if operation == nil {
				t.Fatalf("missing POST %s", test.route)
			}
			if operation.Security == nil || len(*operation.Security) != 1 {
				t.Fatal("operation must declare exactly one security alternative")
			}
			if _, exists := (*operation.Security)[0][test.security]; !exists {
				t.Fatalf("operation security does not use %s", test.security)
			}
			if len(document.Servers) != 1 || !strings.Contains(document.Servers[0].URL, test.serverPart) {
				t.Fatalf("unexpected explicit destination: %#v", document.Servers)
			}
		})
	}
}

func TestRuntimeContractNestedRequestContainsOnlyUntrustedWork(t *testing.T) {
	t.Parallel()

	document := loadOpenAPIDocument(t, filepath.FromSlash("openapi/router-agent.v1.yaml"))
	schema := document.Components.Schemas["NestedInvocationRequest"].Value
	want := []string{"capability", "input", "parentInvocationId", "stream", "targetAgentId"}
	got := make([]string, 0, len(schema.Properties))
	for name := range schema.Properties {
		got = append(got, name)
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("nested request fields = %v, want %v", got, want)
	}
	if schema.AdditionalProperties.Has == nil || *schema.AdditionalProperties.Has {
		t.Fatal("nested request must reject additional trusted fields")
	}

	encoded, err := json.Marshal(NestedInvocationRequestV1{
		ParentInvocationID: "parent-1",
		TargetAgentID:      "agent-b",
		Capability:         "summarize",
		Input:              json.RawMessage(`{"text":"safe"}`),
		Stream:             true,
	})
	if err != nil {
		t.Fatalf("marshal nested request: %v", err)
	}
	text := strings.ToLower(string(encoded))
	for _, forbidden := range []string{"caller", "workspace", "roottask", "traceid", "agentcardversion", "endpoint", "credential", "token"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("nested request leaks trusted/secret field %q: %s", forbidden, encoded)
		}
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("decode nested request fields: %v", err)
	}
	if _, exists := fields["invocationId"]; exists {
		t.Fatal("nested request must not supply the child Invocation ID")
	}
}

func TestRuntimeContractExactFailureMappings(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		"openapi/control-plane-invocation.v4.yaml",
		"openapi/router-internal.v3.yaml",
		"openapi/router-agent.v1.yaml",
	} {
		document := loadOpenAPIDocument(t, filepath.FromSlash(path))
		var route string
		switch path {
		case "openapi/control-plane-invocation.v4.yaml":
			route = "/v4/workspaces/{workspaceId}/invocations"
		case "openapi/router-internal.v3.yaml":
			route = "/internal/v3/invocations"
		default:
			route = "/agent/v1/invocations"
		}
		responses := document.Paths.Find(route).Post.Responses
		assertExtensionStringSliceContains(t, responses.Status(413).Value.Extensions, "x-platform-error-codes", "PAYLOAD_TOO_LARGE")
		assertExtensionStringSliceContains(t, responses.Status(502).Value.Extensions, "x-platform-error-codes", "AGENT_AUTH_UNSUPPORTED")
	}

	if ErrorCodeAgentAuthUnsupported == ErrorCodeRouteNotFound ||
		ErrorCodeAgentAuthUnsupported == ErrorCodeAgentUnavailable ||
		ErrorCodeAgentAuthUnsupported == ErrorCodeDependency {
		t.Fatal("unsupported Agent auth must remain a distinct outcome")
	}
}

func TestRuntimeContractLimitsAndSSEHaveNoDefaults(t *testing.T) {
	t.Parallel()

	for _, test := range []struct{ path, route string }{
		{"openapi/control-plane-invocation.v4.yaml", "/v4/workspaces/{workspaceId}/invocations"},
		{"openapi/router-internal.v3.yaml", "/internal/v3/invocations"},
		{"openapi/router-agent.v1.yaml", "/agent/v1/invocations"},
	} {
		document := loadOpenAPIDocument(t, filepath.FromSlash(test.path))
		operation := document.Paths.Find(test.route).Post
		request := operation.RequestBody.Value.Extensions
		if request["x-nekiro-max-body-bytes-source"] == nil || request["x-nekiro-limit-default"] != false {
			t.Fatalf("%s request limit must be required with no default: %#v", test.path, request)
		}
		stream := operation.Responses.Status(200).Value.Content["text/event-stream"]
		if stream.Extensions["x-nekiro-sse-framing"] != "single-data-line-blank-line-flush" ||
			stream.Extensions["x-nekiro-max-event-bytes-source"] == nil || stream.Extensions["x-nekiro-limit-default"] != false {
			t.Fatalf("%s SSE framing/limit is incomplete: %#v", test.path, stream.Extensions)
		}
	}

	if RuntimeDeadlineMinimumMS != 1 || RuntimeDeadlineMaximumMS != 600000 ||
		RuntimeByteLimitMinimum != 1 || RuntimeByteLimitMaximum != 2147483647 {
		t.Fatal("Go runtime limit mapping differs from the language-neutral contract ranges")
	}
}

func TestRuntimeContractSchemasAndContentExclusion(t *testing.T) {
	t.Parallel()

	document := loadOpenAPIDocument(t, filepath.FromSlash("openapi/router-internal.v3.yaml"))
	request := DispatchInvocationRequestV3{
		InvocationID: "inv-1", RootTaskID: "task-1", TraceID: "trace-1",
		Caller: Caller{Type: "user", ID: "user-1"}, WorkspaceID: "workspace-1",
		TargetAgentID: "agent-1", AgentCardVersion: "1.0.0", Capability: "summarize",
		Input: json.RawMessage(`{"text":"value"}`), Stream: false,
	}
	validateOpenAPIValue(t, document.Components.Schemas["DispatchInvocationRequest"], request)

	for _, schemaPath := range []string{
		"schemas/platform-error.v4.schema.json",
		"schemas/invocation-event.v0.3.schema.json",
		"schemas/invocation-result-stream-event.v2.schema.json",
	} {
		data, err := os.ReadFile(schemaPath)
		if err != nil {
			t.Fatalf("read %s: %v", schemaPath, err)
		}
		lower := strings.ToLower(string(data))
		for _, forbidden := range []string{"apikey", "credentiallocator", "rawdependency", "stacktrace", "endpoint\""} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("%s contains forbidden metadata field %q", schemaPath, forbidden)
			}
		}
	}

	eventSchema := readContractJSONObject(t, "schemas/invocation-event.v0.3.schema.json")
	properties := requiredJSONObject(t, eventSchema, "properties")
	assertObjectKeysAbsent(t, "Invocation Event 0.3", properties, "input", "result", "chunk", "output", "payload", "endpoint", "credential")
}

func TestRuntimeContractPolicyFreezesAcceptanceAndInterruption(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.FromSlash("../docs/decisions/0006-invocation-runtime-trust-and-failure-policy.md"))
	if err != nil {
		t.Fatalf("read ADR 0006: %v", err)
	}
	text := string(data)
	for _, required := range []string{
		"`created` append/projection transaction is the\naccepted-Invocation boundary",
		"last committed non-terminal fact",
		"does not fabricate a terminal event/projection",
		"at most one `tasks/cancel` request",
		"first successfully committed terminal Ledger\ntransaction wins",
		"have no defaults",
		"exactly one `data:` line",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("ADR 0006 missing policy evidence %q", required)
		}
	}
}

func TestRuntimeContractHistoricalArtifactsRemainHistorical(t *testing.T) {
	t.Parallel()

	for _, test := range []struct{ path, version string }{
		{"openapi/control-plane.v3.yaml", "3.0.0"},
		{"openapi/router-internal.v2.yaml", "2.0.0"},
	} {
		document := loadOpenAPIDocument(t, filepath.FromSlash(test.path))
		if document.Info.Version != test.version {
			t.Fatalf("historical %s version changed to %s", test.path, document.Info.Version)
		}
	}
	compatibility, err := os.ReadFile(filepath.FromSlash("../docs/contracts/compatibility.md"))
	if err != nil {
		t.Fatalf("read compatibility guide: %v", err)
	}
	for _, required := range []string{"invocation-only", "Catalog, Workspace, and Installation", "not a second fact", "Do not run v3/v4"} {
		if !strings.Contains(string(compatibility), required) {
			t.Fatalf("compatibility guide missing %q", required)
		}
	}
}

func assertExtensionStringSliceContains(t *testing.T, extensions map[string]any, key, want string) {
	t.Helper()
	value, exists := extensions[key]
	if !exists {
		t.Fatalf("missing extension %s", key)
	}
	items, ok := value.([]any)
	if !ok {
		t.Fatalf("extension %s type = %s, value %#v", key, reflect.TypeOf(value), value)
	}
	for _, item := range items {
		if item == want {
			return
		}
	}
	t.Fatalf("extension %s = %#v, missing %s", key, value, want)
}
