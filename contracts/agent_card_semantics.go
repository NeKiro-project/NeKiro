package contracts

import "fmt"

type AgentCardSemanticRuleID string

const (
	AgentCardRuleUniqueSkillIDs             AgentCardSemanticRuleID = "AC-SEM-001"
	AgentCardRuleUniquePermissionIDs        AgentCardSemanticRuleID = "AC-SEM-002"
	AgentCardRuleRequiredPermissionDeclared AgentCardSemanticRuleID = "AC-SEM-003"
)

type AgentCardConformanceManifest struct {
	Cases []AgentCardConformanceCase `json:"cases"`
}

type AgentCardConformanceCase struct {
	ID            string                    `json:"id"`
	File          string                    `json:"file"`
	Valid         bool                      `json:"valid"`
	ViolatedRules []AgentCardSemanticRuleID `json:"violatedRules"`
}

type AgentCardSemanticViolation struct {
	RuleID AgentCardSemanticRuleID
	Path   string
}

// EvaluateAgentCardSemantics evaluates rules that are outside JSON Schema's
// portable structural guarantees. Callers remain responsible for structural
// validation before treating an empty result as full Agent Card conformance.
func EvaluateAgentCardSemantics(card AgentCard) []AgentCardSemanticViolation {
	violations := make([]AgentCardSemanticViolation, 0)

	skillIDs := make(map[string]struct{}, len(card.Skills))
	for index, skill := range card.Skills {
		if _, exists := skillIDs[skill.ID]; exists {
			violations = append(violations, AgentCardSemanticViolation{
				RuleID: AgentCardRuleUniqueSkillIDs,
				Path:   fmt.Sprintf("/skills/%d/id", index),
			})
		}
		skillIDs[skill.ID] = struct{}{}
	}

	permissionIDs := make(map[string]struct{}, len(card.Permissions))
	for index, permission := range card.Permissions {
		if _, exists := permissionIDs[permission.ID]; exists {
			violations = append(violations, AgentCardSemanticViolation{
				RuleID: AgentCardRuleUniquePermissionIDs,
				Path:   fmt.Sprintf("/permissions/%d/id", index),
			})
		}
		permissionIDs[permission.ID] = struct{}{}
	}

	for skillIndex, skill := range card.Skills {
		for permissionIndex, permissionID := range skill.RequiredPermissions {
			if _, declared := permissionIDs[permissionID]; !declared {
				violations = append(violations, AgentCardSemanticViolation{
					RuleID: AgentCardRuleRequiredPermissionDeclared,
					Path: fmt.Sprintf(
						"/skills/%d/requiredPermissions/%d",
						skillIndex,
						permissionIndex,
					),
				})
			}
		}
	}

	return violations
}
