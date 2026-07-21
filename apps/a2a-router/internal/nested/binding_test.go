package nested

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http/httptest"
	"strings"
	"testing"
)

func testTokenDigest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func TestNewAgentBindingValidation(t *testing.T) {
	validDigest := testTokenDigest("test-token")

	tests := []struct {
		name       string
		principals []AgentPrincipal
		wantErr    bool
	}{
		{
			"valid single principal",
			[]AgentPrincipal{{WorkspaceID: "workspace01", AgentID: "agent01", TokenSHA256: validDigest}},
			false,
		},
		{
			"empty principals",
			[]AgentPrincipal{},
			true,
		},
		{
			"nil principals",
			nil,
			true,
		},
		{
			"invalid agent id",
			[]AgentPrincipal{{WorkspaceID: "workspace01", AgentID: "agent 01", TokenSHA256: validDigest}},
			true,
		},
		{
			"empty agent id",
			[]AgentPrincipal{{WorkspaceID: "workspace01", AgentID: "", TokenSHA256: validDigest}},
			true,
		},
		{
			"empty workspace id",
			[]AgentPrincipal{{WorkspaceID: "", AgentID: "agent01", TokenSHA256: validDigest}},
			true,
		},
		{
			"invalid token digest",
			[]AgentPrincipal{{WorkspaceID: "workspace01", AgentID: "agent01", TokenSHA256: "not-hex"}},
			true,
		},
		{
			"short token digest",
			[]AgentPrincipal{{WorkspaceID: "workspace01", AgentID: "agent01", TokenSHA256: "abcd"}},
			true,
		},
		{
			"uppercase token digest",
			[]AgentPrincipal{{WorkspaceID: "workspace01", AgentID: "agent01", TokenSHA256: strings.ToUpper(validDigest)}},
			true,
		},
		{
			"duplicate workspace and agent",
			[]AgentPrincipal{
				{WorkspaceID: "workspace01", AgentID: "agent01", TokenSHA256: validDigest},
				{WorkspaceID: "workspace01", AgentID: "agent01", TokenSHA256: testTokenDigest("other-token")},
			},
			true,
		},
		{
			"same agent in distinct workspaces",
			[]AgentPrincipal{
				{WorkspaceID: "workspace01", AgentID: "agent01", TokenSHA256: validDigest},
				{WorkspaceID: "workspace02", AgentID: "agent01", TokenSHA256: testTokenDigest("other-token")},
			},
			false,
		},
		{
			"duplicate token digest",
			[]AgentPrincipal{
				{WorkspaceID: "workspace01", AgentID: "agent01", TokenSHA256: validDigest},
				{WorkspaceID: "workspace02", AgentID: "agent02", TokenSHA256: validDigest},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAgentBinding(tt.principals)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAgentBinding() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAgentBindingAuthenticate(t *testing.T) {
	token := "my-secret-token"
	digest := testTokenDigest(token)
	binding, err := NewAgentBinding([]AgentPrincipal{{WorkspaceID: "workspace01", AgentID: "agent01", TokenSHA256: digest}})
	if err != nil {
		t.Fatalf("NewAgentBinding() error = %v", err)
	}

	tests := []struct {
		name          string
		authHeader    string
		headerCount   int
		wantPrincipal AuthenticatedAgent
		wantErr       bool
		wantForbidden bool
	}{
		{
			"valid token",
			"Bearer " + token,
			1,
			AuthenticatedAgent{WorkspaceID: "workspace01", AgentID: "agent01"},
			false,
			false,
		},
		{
			"missing header",
			"",
			0,
			AuthenticatedAgent{},
			true,
			false,
		},
		{
			"empty header",
			"",
			1,
			AuthenticatedAgent{},
			true,
			false,
		},
		{
			"no bearer prefix",
			token,
			1,
			AuthenticatedAgent{},
			true,
			false,
		},
		{
			"empty bearer token",
			"Bearer ",
			1,
			AuthenticatedAgent{},
			true,
			false,
		},
		{
			"whitespace token",
			"Bearer  token ",
			1,
			AuthenticatedAgent{},
			true,
			false,
		},
		{
			"unknown token",
			"Bearer unknown-token",
			1,
			AuthenticatedAgent{},
			true,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/agent/v1/invocations", nil)
			if tt.headerCount > 0 {
				req.Header.Set("Authorization", tt.authHeader)
			}
			principal, err := binding.Authenticate(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Authenticate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if principal != tt.wantPrincipal {
				t.Errorf("Authenticate() principal = %v, want %v", principal, tt.wantPrincipal)
			}
			if tt.wantForbidden && err != ErrForbidden {
				t.Errorf("Authenticate() expected ErrForbidden, got %v", err)
			}
		})
	}
}

func TestAgentBindingMultiplePrincipals(t *testing.T) {
	token1 := "token-agent-1"
	token2 := "token-agent-2"
	binding, err := NewAgentBinding([]AgentPrincipal{
		{WorkspaceID: "workspace01", AgentID: "agent01", TokenSHA256: testTokenDigest(token1)},
		{WorkspaceID: "workspace02", AgentID: "agent02", TokenSHA256: testTokenDigest(token2)},
	})
	if err != nil {
		t.Fatalf("NewAgentBinding() error = %v", err)
	}

	req1 := httptest.NewRequest("POST", "/agent/v1/invocations", nil)
	req1.Header.Set("Authorization", "Bearer "+token1)
	principal1, err := binding.Authenticate(req1)
	if err != nil || principal1 != (AuthenticatedAgent{WorkspaceID: "workspace01", AgentID: "agent01"}) {
		t.Errorf("unexpected first principal: %v, err %v", principal1, err)
	}

	req2 := httptest.NewRequest("POST", "/agent/v1/invocations", nil)
	req2.Header.Set("Authorization", "Bearer "+token2)
	principal2, err := binding.Authenticate(req2)
	if err != nil || principal2 != (AuthenticatedAgent{WorkspaceID: "workspace02", AgentID: "agent02"}) {
		t.Errorf("unexpected second principal: %v, err %v", principal2, err)
	}
}
