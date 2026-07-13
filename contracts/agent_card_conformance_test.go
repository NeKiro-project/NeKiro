package contracts

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const (
	agentCardV02SchemaID = "https://schemas.nekiro.dev/agent-card/v0.2"
	commonV1SchemaID     = "https://schemas.nekiro.dev/common/v1"
)

func TestAgentCardConformance(t *testing.T) {
	schema := compileAgentCardV02Schema(t)
	manifestPath := filepath.Join("agent-card", "v0.2", "conformance", "manifest.json")
	manifest := loadAgentCardConformanceManifest(t, manifestPath)
	if len(manifest.Cases) == 0 {
		t.Fatal("Agent Card conformance manifest contains no cases")
	}

	knownRuleIDs := map[AgentCardSemanticRuleID]struct{}{
		AgentCardRuleUniqueSkillIDs:             {},
		AgentCardRuleUniquePermissionIDs:        {},
		AgentCardRuleRequiredPermissionDeclared: {},
	}
	requiredCaseIDs := []string{
		"valid-baseline",
		"valid-shared-permission",
		"invalid-duplicate-skill-id",
		"invalid-duplicate-permission-id",
		"invalid-undeclared-permission",
		"invalid-cross-version-permission",
		"invalid-case-mismatched-permission",
		"invalid-structural-missing-name",
	}
	caseIDs := make(map[string]struct{}, len(manifest.Cases))

	for _, testCase := range manifest.Cases {
		t.Run(testCase.ID, func(t *testing.T) {
			if _, exists := caseIDs[testCase.ID]; exists {
				t.Fatalf("duplicate conformance case id %q", testCase.ID)
			}
			caseIDs[testCase.ID] = struct{}{}
			for _, ruleID := range testCase.ViolatedRules {
				if _, exists := knownRuleIDs[ruleID]; !exists {
					t.Fatalf("manifest contains unknown semantic rule id %q", ruleID)
				}
			}

			fixturePath := filepath.Join(filepath.Dir(manifestPath), testCase.File)
			fixture := readConformanceFixture(t, fixturePath)
			document, err := jsonschema.UnmarshalJSON(bytes.NewReader(fixture))
			if err != nil {
				t.Fatalf("decode raw fixture: %v", err)
			}

			structuralErr := schema.Validate(document)
			actualRuleIDs := make([]AgentCardSemanticRuleID, 0)
			if structuralErr == nil {
				var card AgentCard
				decoder := json.NewDecoder(bytes.NewReader(fixture))
				decoder.DisallowUnknownFields()
				if err := decoder.Decode(&card); err != nil {
					t.Fatalf("decode structurally valid fixture into Go mapping: %v", err)
				}
				if err := requireJSONEOF(decoder); err != nil {
					t.Fatalf("decode structurally valid fixture into Go mapping: %v", err)
				}
				actualRuleIDs = uniqueSemanticRuleIDs(EvaluateAgentCardSemantics(card))
			} else if len(testCase.ViolatedRules) > 0 {
				t.Fatalf("semantic fixture failed structural validation: %v", structuralErr)
			}

			actualValid := structuralErr == nil && len(actualRuleIDs) == 0
			if actualValid != testCase.Valid {
				t.Fatalf("combined validity = %t, want %t (structural error: %v, rules: %v)", actualValid, testCase.Valid, structuralErr, actualRuleIDs)
			}

			wantRuleIDs := slices.Clone(testCase.ViolatedRules)
			slices.Sort(wantRuleIDs)
			if !slices.Equal(actualRuleIDs, wantRuleIDs) {
				t.Fatalf("violated rules = %v, want %v", actualRuleIDs, wantRuleIDs)
			}
		})
	}

	for _, caseID := range requiredCaseIDs {
		if _, exists := caseIDs[caseID]; !exists {
			t.Errorf("Agent Card conformance manifest is missing required case %q", caseID)
		}
	}
}

func compileAgentCardV02Schema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()

	commonSchema, err := readJSONDocument("schemas/common.v1.schema.json")
	if err != nil {
		t.Fatalf("read common schema: %v", err)
	}
	if err := compiler.AddResource(commonV1SchemaID, commonSchema); err != nil {
		t.Fatalf("add common schema: %v", err)
	}

	agentCardSchema, err := readJSONDocument("schemas/agent-card.v0.2.schema.json")
	if err != nil {
		t.Fatalf("read Agent Card v0.2 schema: %v", err)
	}
	if err := compiler.AddResource(agentCardV02SchemaID, agentCardSchema); err != nil {
		t.Fatalf("add Agent Card v0.2 schema: %v", err)
	}

	compiled, err := compiler.Compile(agentCardV02SchemaID)
	if err != nil {
		t.Fatalf("compile Agent Card v0.2 schema: %v", err)
	}
	return compiled
}

func loadAgentCardConformanceManifest(t *testing.T, path string) AgentCardConformanceManifest {
	t.Helper()
	data := readConformanceFixture(t, path)
	var manifest AgentCardConformanceManifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		t.Fatalf("decode Agent Card conformance manifest: %v", err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		t.Fatalf("decode Agent Card conformance manifest: %v", err)
	}
	return manifest
}

func readConformanceFixture(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

func uniqueSemanticRuleIDs(violations []AgentCardSemanticViolation) []AgentCardSemanticRuleID {
	seen := make(map[AgentCardSemanticRuleID]struct{}, len(violations))
	ruleIDs := make([]AgentCardSemanticRuleID, 0, len(violations))
	for _, violation := range violations {
		if _, exists := seen[violation.RuleID]; exists {
			continue
		}
		seen[violation.RuleID] = struct{}{}
		ruleIDs = append(ruleIDs, violation.RuleID)
	}
	slices.Sort(ruleIDs)
	return ruleIDs
}
