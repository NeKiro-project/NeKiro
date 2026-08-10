package contracts

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"testing"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestValidateRouterTopologyStatusV1(t *testing.T) {
	valid := validRouterTopologyStatus()
	if err := ValidateRouterTopologyStatusV1(valid); err != nil {
		t.Fatalf("valid status rejected: %v", err)
	}
	for name, mutate := range map[string]func(*RouterTopologyStatusV1){
		"schema":    func(status *RouterTopologyStatusV1) { status.SchemaVersion = "2" },
		"provider":  func(status *RouterTopologyStatusV1) { status.Provider = "not safe" },
		"nil list":  func(status *RouterTopologyStatusV1) { status.Observations = nil },
		"agent":     func(status *RouterTopologyStatusV1) { status.Observations[0].AgentID = "not safe" },
		"version":   func(status *RouterTopologyStatusV1) { status.Observations[0].AgentCardVersion = "latest" },
		"release":   func(status *RouterTopologyStatusV1) { status.Observations[0].ReleaseID = "not safe" },
		"state":     func(status *RouterTopologyStatusV1) { status.Observations[0].State = "stale" },
		"timestamp": func(status *RouterTopologyStatusV1) { status.Observations[0].ObservedAt = time.Time{} },
		"unencodable timestamp": func(status *RouterTopologyStatusV1) {
			status.Observations[0].ObservedAt = time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
		},
		"duplicate": func(status *RouterTopologyStatusV1) {
			status.Observations = append(status.Observations, status.Observations[0])
		},
	} {
		t.Run(name, func(t *testing.T) {
			status := validRouterTopologyStatus()
			mutate(&status)
			if err := ValidateRouterTopologyStatusV1(status); err == nil {
				t.Fatal("invalid status accepted")
			}
		})
	}
}

func TestRouterTopologyStatusSchemaAndOpenAPIMatchGo(t *testing.T) {
	statusData, err := fs.ReadFile(ContractFiles(), "schemas/router-topology-status.v1.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	statusDocument, err := jsonschema.UnmarshalJSON(bytes.NewReader(statusData))
	if err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	if err := compiler.AddResource("https://schemas.nekiro.dev/router-topology-status/v1", statusDocument); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile("https://schemas.nekiro.dev/router-topology-status/v1")
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(validRouterTopologyStatus())
	if err != nil {
		t.Fatal(err)
	}
	var value any
	if err := json.Unmarshal(encoded, &value); err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(value); err != nil {
		t.Fatalf("schema rejected Go mapping: %v", err)
	}

	document := loadOpenAPIDocument(t, "openapi/router-topology-status.v1.yaml")
	operation := document.Paths.Find("/internal/v1/instance-topology/status").Get
	if operation == nil || operation.Security == nil || len(*operation.Security) != 1 || operation.Responses.Status(200) == nil ||
		operation.Responses.Status(401) == nil || operation.Responses.Status(403) == nil || operation.Responses.Status(503) == nil {
		t.Fatal("topology status operation does not preserve authenticated response contract")
	}
	validateOpenAPIValue(t, operation.Responses.Status(200).Value.Content["application/json"].Schema, validRouterTopologyStatus())
}

func TestRouterTopologyStatusContractExcludesSensitiveTopology(t *testing.T) {
	data, err := fs.ReadFile(ContractFiles(), "schemas/router-topology-status.v1.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	encoded := string(data)
	for _, forbidden := range []string{"endpoint", "address", "sourceToken", "instanceId", "metadata", "cardDigest", "audience", "credential", "payload", "input", "output"} {
		if bytes.Contains(bytes.ToLower([]byte(encoded)), bytes.ToLower([]byte(forbidden))) {
			t.Fatalf("status schema contains forbidden topology field %q", forbidden)
		}
	}
}

func validRouterTopologyStatus() RouterTopologyStatusV1 {
	return RouterTopologyStatusV1{
		SchemaVersion: RouterTopologyStatusSchemaVersion,
		Provider:      "nacos",
		Observations: []RouterTopologyStatusObservationV1{{
			AgentID: "runtime-b", AgentCardVersion: "1.0.0", ReleaseID: "release-b",
			State: RouterTopologyStatePopulated, LocalRevision: 3,
			ObservedAt: time.Date(2026, 8, 10, 5, 0, 0, 0, time.UTC),
		}},
	}
}
