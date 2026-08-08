package contracts

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

const RouterInstanceDirectorySchemaVersion = "1"

const (
	routerDirectoryTargetLimit          = 1024
	routerDirectoryInstanceLimit        = 4096
	routerDirectoryEndpointLimit        = 32
	routerDirectoryAddressLengthLimit   = 253
	routerDirectoryHTTPValueLengthLimit = 2048
	routerDirectoryVersionLengthLimit   = 128
)

var routerDirectoryDigestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// RouterInstanceDirectoryV1 is the Router-owned dynamic routing document.
// Every entry is scoped to one exact, already-authorized Agent Release.
type RouterInstanceDirectoryV1 struct {
	SchemaVersion string                            `json:"schemaVersion"`
	Revision      string                            `json:"revision"`
	Targets       []RouterInstanceDirectoryTargetV1 `json:"targets"`
}

type RouterInstanceDirectoryTargetV1 struct {
	AgentID           string             `json:"agentId"`
	AgentCardVersion  string             `json:"agentCardVersion"`
	ReleaseID         string             `json:"releaseId"`
	CardDigest        string             `json:"cardDigest"`
	CanonicalEndpoint string             `json:"canonicalEndpoint"`
	Audience          string             `json:"audience"`
	Instances         []RouterInstanceV1 `json:"instances"`
}

type RouterInstanceV1 struct {
	InstanceID  string                    `json:"instanceId"`
	Ready       bool                      `json:"ready"`
	Serving     bool                      `json:"serving"`
	Terminating bool                      `json:"terminating"`
	Endpoints   []RouterNetworkEndpointV1 `json:"endpoints"`
}

type RouterNetworkEndpointV1 struct {
	AddressType string `json:"addressType"`
	Address     string `json:"address"`
	PortName    string `json:"portName"`
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"`
}

// DecodeRouterInstanceDirectoryV1 rejects unknown and duplicate members. The
// registry package performs the provider-neutral semantic validation when it
// turns these wire values into immutable topology values.
func DecodeRouterInstanceDirectoryV1(data []byte) (RouterInstanceDirectoryV1, error) {
	if err := rejectDuplicateJSONMemberNames(data); err != nil {
		return RouterInstanceDirectoryV1{}, fmt.Errorf("decode Router instance directory: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var document RouterInstanceDirectoryV1
	if err := decoder.Decode(&document); err != nil {
		return RouterInstanceDirectoryV1{}, fmt.Errorf("decode Router instance directory: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return RouterInstanceDirectoryV1{}, errors.New("decode Router instance directory: unexpected trailing JSON value")
		}
		return RouterInstanceDirectoryV1{}, fmt.Errorf("decode Router instance directory: %w", err)
	}
	if document.SchemaVersion != RouterInstanceDirectorySchemaVersion {
		return RouterInstanceDirectoryV1{}, errors.New("decode Router instance directory: unsupported schemaVersion")
	}
	if !safeIdentifierPattern.MatchString(document.Revision) {
		return RouterInstanceDirectoryV1{}, errors.New("decode Router instance directory: invalid revision")
	}
	if document.Targets == nil {
		return RouterInstanceDirectoryV1{}, errors.New("decode Router instance directory: targets must be an array")
	}
	if err := validateRouterInstanceDirectoryV1(document); err != nil {
		return RouterInstanceDirectoryV1{}, fmt.Errorf("decode Router instance directory: %w", err)
	}
	return document, nil
}

func validateRouterInstanceDirectoryV1(document RouterInstanceDirectoryV1) error {
	if len(document.Targets) > routerDirectoryTargetLimit {
		return errors.New("too many targets")
	}
	for _, target := range document.Targets {
		if !safeIdentifierPattern.MatchString(target.AgentID) ||
			utf8.RuneCountInString(target.AgentCardVersion) < 1 || utf8.RuneCountInString(target.AgentCardVersion) > routerDirectoryVersionLengthLimit ||
			!safeIdentifierPattern.MatchString(target.ReleaseID) || !routerDirectoryDigestPattern.MatchString(target.CardDigest) ||
			!validRouterDirectoryHTTPURI(target.CanonicalEndpoint) || !validRouterDirectoryHTTPURI(target.Audience) ||
			target.Instances == nil || len(target.Instances) > routerDirectoryInstanceLimit {
			return errors.New("invalid target")
		}
		for _, instance := range target.Instances {
			if !safeIdentifierPattern.MatchString(instance.InstanceID) || len(instance.Endpoints) < 1 || len(instance.Endpoints) > routerDirectoryEndpointLimit {
				return errors.New("invalid instance")
			}
			for _, endpoint := range instance.Endpoints {
				if endpoint.AddressType != "IPv4" && endpoint.AddressType != "IPv6" && endpoint.AddressType != "DNS" ||
					utf8.RuneCountInString(endpoint.Address) < 1 || utf8.RuneCountInString(endpoint.Address) > routerDirectoryAddressLengthLimit ||
					!safeIdentifierPattern.MatchString(endpoint.PortName) || endpoint.Port < 1 || endpoint.Port > 65535 || endpoint.Protocol != "TCP" {
					return errors.New("invalid endpoint")
				}
			}
		}
	}
	return nil
}

func validRouterDirectoryHTTPURI(value string) bool {
	if utf8.RuneCountInString(value) > routerDirectoryHTTPValueLengthLimit || !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		return false
	}
	parsed, err := url.Parse(value)
	return err == nil && parsed.IsAbs() && parsed.Host != ""
}
