package contracts

import (
	"errors"
	"fmt"
	"time"

	semver "github.com/Masterminds/semver/v3"
)

const (
	RouterTopologyStatusSchemaVersion      = "1"
	RouterTopologyStatusObservationMaximum = 65536
)

type RouterTopologyObservationState string

const (
	RouterTopologyStateInitializing RouterTopologyObservationState = "initializing"
	RouterTopologyStateMissing      RouterTopologyObservationState = "missing"
	RouterTopologyStateEmpty        RouterTopologyObservationState = "empty"
	RouterTopologyStatePopulated    RouterTopologyObservationState = "populated"
	RouterTopologyStateUnavailable  RouterTopologyObservationState = "unavailable"
)

// RouterTopologyStatusV1 is a secrecy-safe projection of the Router's local
// watched topology. It intentionally excludes endpoints and provider tokens.
type RouterTopologyStatusV1 struct {
	SchemaVersion string                              `json:"schemaVersion"`
	Provider      string                              `json:"provider"`
	Observations  []RouterTopologyStatusObservationV1 `json:"observations"`
}

type RouterTopologyStatusObservationV1 struct {
	AgentID          string                         `json:"agentId"`
	AgentCardVersion string                         `json:"agentCardVersion"`
	ReleaseID        string                         `json:"releaseId"`
	State            RouterTopologyObservationState `json:"state"`
	LocalRevision    uint64                         `json:"localRevision"`
	ObservedAt       time.Time                      `json:"observedAt"`
}

func ValidateRouterTopologyStatusV1(status RouterTopologyStatusV1) error {
	if status.SchemaVersion != RouterTopologyStatusSchemaVersion {
		return errors.New("invalid Router topology status schemaVersion")
	}
	if !safeIdentifierPattern.MatchString(status.Provider) {
		return errors.New("invalid Router topology status provider")
	}
	if status.Observations == nil || len(status.Observations) > RouterTopologyStatusObservationMaximum {
		return errors.New("invalid Router topology status observations")
	}
	seen := make(map[string]struct{}, len(status.Observations))
	for _, observation := range status.Observations {
		if !safeIdentifierPattern.MatchString(observation.AgentID) ||
			!safeIdentifierPattern.MatchString(observation.ReleaseID) {
			return errors.New("invalid Router topology status target")
		}
		if _, err := semver.StrictNewVersion(observation.AgentCardVersion); err != nil {
			return errors.New("invalid Router topology status Agent Card version")
		}
		switch observation.State {
		case RouterTopologyStateInitializing, RouterTopologyStateMissing, RouterTopologyStateEmpty,
			RouterTopologyStatePopulated, RouterTopologyStateUnavailable:
		default:
			return errors.New("invalid Router topology observation state")
		}
		if _, err := observation.ObservedAt.MarshalText(); observation.ObservedAt.IsZero() || err != nil {
			return errors.New("invalid Router topology observation timestamp")
		}
		key := fmt.Sprintf("%s\x00%s\x00%s", observation.AgentID, observation.AgentCardVersion, observation.ReleaseID)
		if _, exists := seen[key]; exists {
			return errors.New("duplicate Router topology observation target")
		}
		seen[key] = struct{}{}
	}
	return nil
}
