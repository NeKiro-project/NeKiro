package a2a

import (
	"errors"
	"net/url"

	"github.com/Nene7ko/NeKiro/contracts"
)

type Target struct {
	AgentID    string
	Version    string
	Capability string
	Endpoint   string
	Protocol   string
	Transport  string
	AuthType   string
}

func NewTarget(resolved contracts.ResolveAgentResponse, capability string) (Target, error) {
	if capability == "" {
		return Target{}, errors.New("dispatch capability is required")
	}
	card := resolved.Card
	if card.AgentID == "" || card.Version == "" {
		return Target{}, errors.New("resolved Agent identity is required")
	}
	if card.Protocol.Type != "a2a" || card.Protocol.Version != contracts.A2AProtocolVersion || card.Protocol.Transport != "JSONRPC" {
		return Target{}, errors.New("resolved Agent A2A profile is unsupported")
	}
	if card.Protocol.Endpoint == "" {
		return Target{}, errors.New("resolved Agent endpoint is required")
	}
	endpoint, err := url.Parse(card.Protocol.Endpoint)
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" || endpoint.User != nil {
		return Target{}, errors.New("resolved Agent endpoint is invalid")
	}
	if endpoint.Scheme != "http" && endpoint.Scheme != "https" {
		return Target{}, errors.New("resolved Agent endpoint scheme is unsupported")
	}
	if card.Authentication.Type != "none" {
		return Target{}, errors.New("resolved Agent authentication is unsupported")
	}
	if !declaresCapability(card, capability) {
		return Target{}, errors.New("resolved Agent capability is missing")
	}
	return Target{
		AgentID: card.AgentID, Version: card.Version, Capability: capability,
		Endpoint: card.Protocol.Endpoint, Protocol: card.Protocol.Type,
		Transport: card.Protocol.Transport, AuthType: card.Authentication.Type,
	}, nil
}

func declaresCapability(card contracts.AgentCard, capability string) bool {
	for _, skill := range card.Skills {
		if skill.ID == capability {
			return true
		}
	}
	return false
}
