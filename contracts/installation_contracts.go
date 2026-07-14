package contracts

import "fmt"

type InstallationSemanticRuleID string

const (
	InstallationRuleCanonicalPermissions InstallationSemanticRuleID = "INST-SEM-001"
	InstallationRuleMonotonicUpdate      InstallationSemanticRuleID = "INST-SEM-002"
	InstallationRuleTerminalTimestamp    InstallationSemanticRuleID = "INST-SEM-003"
	InstallationRuleImmutablePin         InstallationSemanticRuleID = "INST-SEM-004"
)

type InstallationSemanticValidationError struct {
	RuleID InstallationSemanticRuleID
}

func (validationError *InstallationSemanticValidationError) Error() string {
	return fmt.Sprintf("installation semantic validation failed (%s)", validationError.RuleID)
}

func validateInstallationV2Semantics(installation Installation) error {
	for index := 1; index < len(installation.AcceptedPermissions); index++ {
		if installation.AcceptedPermissions[index-1] >= installation.AcceptedPermissions[index] {
			return &InstallationSemanticValidationError{RuleID: InstallationRuleCanonicalPermissions}
		}
	}
	if installation.InstalledAt.After(installation.UpdatedAt) {
		return &InstallationSemanticValidationError{RuleID: InstallationRuleMonotonicUpdate}
	}
	if installation.Status == "uninstalled" {
		if installation.UninstalledAt == nil || !installation.UninstalledAt.Equal(installation.UpdatedAt) {
			return &InstallationSemanticValidationError{RuleID: InstallationRuleTerminalTimestamp}
		}
	}
	return nil
}

func ValidateInstallationImmutablePin(before, after Installation) error {
	if before.VersionConstraint != after.VersionConstraint ||
		before.InstalledVersion != after.InstalledVersion ||
		len(before.AcceptedPermissions) != len(after.AcceptedPermissions) {
		return &InstallationSemanticValidationError{RuleID: InstallationRuleImmutablePin}
	}
	for index := range before.AcceptedPermissions {
		if before.AcceptedPermissions[index] != after.AcceptedPermissions[index] {
			return &InstallationSemanticValidationError{RuleID: InstallationRuleImmutablePin}
		}
	}
	return nil
}
