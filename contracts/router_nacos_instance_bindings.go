package contracts

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const RouterNacosInstanceBindingsSchemaVersion = "1"

const routerNacosBindingLimit = 1024

type RouterNacosInstanceBindingsV1 struct {
	SchemaVersion string                               `json:"schemaVersion"`
	Revision      string                               `json:"revision"`
	Targets       []RouterNacosInstanceBindingTargetV1 `json:"targets"`
}

type RouterNacosInstanceBindingTargetV1 struct {
	AgentID           string `json:"agentId"`
	AgentCardVersion  string `json:"agentCardVersion"`
	ReleaseID         string `json:"releaseId"`
	CardDigest        string `json:"cardDigest"`
	CanonicalEndpoint string `json:"canonicalEndpoint"`
	Audience          string `json:"audience"`
	ServiceName       string `json:"serviceName"`
	GroupName         string `json:"groupName"`
	ClusterName       string `json:"clusterName"`
}

func DecodeRouterNacosInstanceBindingsV1(data []byte) (RouterNacosInstanceBindingsV1, error) {
	if err := rejectDuplicateJSONMemberNames(data); err != nil {
		return RouterNacosInstanceBindingsV1{}, fmt.Errorf("decode Router Nacos instance bindings: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var document RouterNacosInstanceBindingsV1
	if err := decoder.Decode(&document); err != nil {
		return RouterNacosInstanceBindingsV1{}, fmt.Errorf("decode Router Nacos instance bindings: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return RouterNacosInstanceBindingsV1{}, errors.New("decode Router Nacos instance bindings: unexpected trailing JSON value")
		}
		return RouterNacosInstanceBindingsV1{}, fmt.Errorf("decode Router Nacos instance bindings: %w", err)
	}
	if document.SchemaVersion != RouterNacosInstanceBindingsSchemaVersion || !safeIdentifierPattern.MatchString(document.Revision) || document.Targets == nil || len(document.Targets) > routerNacosBindingLimit {
		return RouterNacosInstanceBindingsV1{}, errors.New("decode Router Nacos instance bindings: invalid document")
	}
	for _, target := range document.Targets {
		if !safeIdentifierPattern.MatchString(target.AgentID) || target.AgentCardVersion == "" ||
			!safeIdentifierPattern.MatchString(target.ReleaseID) || !routerDirectoryDigestPattern.MatchString(target.CardDigest) ||
			!validRouterDirectoryHTTPURI(target.CanonicalEndpoint) || !validRouterDirectoryHTTPURI(target.Audience) ||
			!safeIdentifierPattern.MatchString(target.ServiceName) || !safeIdentifierPattern.MatchString(target.GroupName) ||
			!safeIdentifierPattern.MatchString(target.ClusterName) {
			return RouterNacosInstanceBindingsV1{}, errors.New("decode Router Nacos instance bindings: invalid target")
		}
	}
	return document, nil
}
