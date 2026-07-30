package contracts

import "time"

const (
	PublicAgentAvailabilityInstallable    = "installable"
	PublicAgentAvailabilityNotInstallable = "not_installable"
)

type PublicAgentRelease struct {
	ReleaseID          string                  `json:"releaseId"`
	AgentID            string                  `json:"agentId"`
	Name               string                  `json:"name"`
	Description        string                  `json:"description"`
	Owner              AgentOwner              `json:"owner"`
	AgentCardVersion   string                  `json:"agentCardVersion"`
	CardDigest         string                  `json:"cardDigest"`
	PublishedAt        time.Time               `json:"publishedAt"`
	AuthenticationType string                  `json:"authenticationType"`
	Skills             []AgentSkill            `json:"skills"`
	Permissions        []PermissionDeclaration `json:"permissions"`
	Limits             AgentLimits             `json:"limits"`
}

type PublicAgentShare struct {
	SchemaVersion string               `json:"schemaVersion"`
	PublicAgentID string               `json:"publicAgentId"`
	PublicURL     string               `json:"publicUrl"`
	RegisteredAt  time.Time            `json:"registeredAt"`
	Availability  string               `json:"availability"`
	Releases      []PublicAgentRelease `json:"releases"`
}
