package a2a

import (
	"testing"

	"github.com/Nene7ko/NeKiro/contracts"
)

func TestNewTargetAcceptsExactA2ANoneAuthEndpoint(t *testing.T) {
	resolved := contracts.ResolveAgentResponse{Card: targetCard("http://127.0.0.1:4101/a2a", "none", "capability-a")}
	target, err := NewTarget(resolved, "capability-a")
	if err != nil {
		t.Fatalf("NewTarget = %v", err)
	}
	if target.AgentID != "agent-a" || target.Version != "1.0.0" || target.Endpoint != "http://127.0.0.1:4101/a2a" || target.AuthType != "none" {
		t.Fatalf("target = %#v", target)
	}
}

func TestNewTargetRejectsUnsupportedStates(t *testing.T) {
	tests := []struct {
		name string
		card contracts.AgentCard
		cap  string
	}{
		{name: "missing capability", card: targetCard("http://127.0.0.1:4101/a2a", "none", "capability-a"), cap: ""},
		{name: "unsupported scheme", card: targetCard("ftp://agent.example/a2a", "none", "capability-a"), cap: "capability-a"},
		{name: "userinfo", card: targetCard("http://user@example.test/a2a", "none", "capability-a"), cap: "capability-a"},
		{name: "unsupported auth", card: targetCard("http://127.0.0.1:4101/a2a", "bearer", "capability-a"), cap: "capability-a"},
		{name: "missing skill", card: targetCard("http://127.0.0.1:4101/a2a", "none", "capability-b"), cap: "capability-a"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewTarget(contracts.ResolveAgentResponse{Card: test.card}, test.cap); err == nil {
				t.Fatal("NewTarget succeeded, want error")
			}
		})
	}
}

func targetCard(endpoint, authType, capability string) contracts.AgentCard {
	return contracts.AgentCard{
		AgentID: "agent-a", Version: "1.0.0",
		Protocol:       contracts.AgentProtocol{Type: "a2a", Version: contracts.A2AProtocolVersion, Transport: "JSONRPC", Endpoint: endpoint},
		Authentication: contracts.AgentAuthentication{Type: authType},
		Skills:         []contracts.AgentSkill{{ID: capability}},
	}
}
