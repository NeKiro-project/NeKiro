package catalog

import (
	"context"
	"crypto/sha256"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Nene7ko/NeKiro/contracts"
)

func TestParseEndpointRejectsCredentialsQueryAndFragment(t *testing.T) {
	for _, value := range []string{
		"https://user:pass@example.test/a2a",
		"https://example.test/a2a?token=secret",
		"https://example.test/a2a#fragment",
		"ftp://example.test/a2a",
	} {
		if _, err := ParseEndpoint(value); !errors.Is(err, ErrEndpointInvalid) {
			t.Fatalf("ParseEndpoint(%q) error=%v, want ErrEndpointInvalid", value, err)
		}
	}
}

func TestParseEndpointCanonicalizesAuthorityAndRejectsAliases(t *testing.T) {
	tests := map[string]string{
		"https://AGENT.EXAMPLE.:443/a2a": "https://agent.example/a2a",
		"http://AGENT.EXAMPLE.:80/a2a":   "http://agent.example/a2a",
		"http://agent.example:8080/a2a":  "http://agent.example:8080/a2a",
		"https://[2001:DB8::1]:443/a2a":  "https://[2001:db8::1]/a2a",
	}
	for raw, want := range tests {
		endpoint, err := ParseEndpoint(raw)
		if err != nil || endpoint.Canonical != want {
			t.Fatalf("ParseEndpoint(%q) = %#v, %v; want %q", raw, endpoint, err, want)
		}
	}
	for _, raw := range []string{
		"https://agent.example:0/a2a",
		"https://agent.example:65536/a2a",
		"https://agent.example/a%2Fa",
		"https://agent.example/a/./b",
		"https://agent.example/a/../b",
	} {
		if _, err := ParseEndpoint(raw); !errors.Is(err, ErrEndpointInvalid) {
			t.Fatalf("ParseEndpoint(%q) error=%v, want ErrEndpointInvalid", raw, err)
		}
	}
}

func TestEndpointPolicyRejectsPrivateDestinationUnlessExplicitlyAllowed(t *testing.T) {
	endpoint, err := ParseEndpoint("http://runtime-a:8080/a2a")
	if err != nil {
		t.Fatal(err)
	}
	lookup := func(context.Context, string) ([]net.IP, error) { return []net.IP{net.ParseIP("10.0.0.5")}, nil }
	policy := EndpointPolicy{LookupIP: lookup}
	if err := policy.ValidateDestination(context.Background(), endpoint); !errors.Is(err, ErrDisallowedNetwork) {
		t.Fatalf("private destination error=%v, want ErrDisallowedNetwork", err)
	}
	policy.AllowedPrivateHosts = map[string]struct{}{"runtime-a": {}}
	if err := policy.ValidateDestination(context.Background(), endpoint); err != nil {
		t.Fatalf("explicit private host allowlist error=%v", err)
	}
}

func TestEndpointPolicyRejectsReservedAndSpecialDestinations(t *testing.T) {
	for _, address := range []string{
		"0.1.2.3", "127.0.0.1", "10.0.0.1", "169.254.1.1", "224.0.0.1",
		"100.64.0.1", "192.0.2.1", "198.18.0.1", "203.0.113.1", "240.0.0.1",
		"::1", "fe80::1", "ff02::1", "100::1", "2001:1::1", "2001:20::1", "2001:db8::1",
	} {
		endpoint, err := ParseEndpoint("http://runtime.test/a2a")
		if err != nil {
			t.Fatal(err)
		}
		policy := EndpointPolicy{LookupIP: func(context.Context, string) ([]net.IP, error) { return []net.IP{net.ParseIP(address)}, nil }}
		if err := policy.ValidateDestination(context.Background(), endpoint); !errors.Is(err, ErrDisallowedNetwork) {
			t.Fatalf("destination %s error=%v, want ErrDisallowedNetwork", address, err)
		}
	}
	endpoint, err := ParseEndpoint("http://runtime.test/a2a")
	if err != nil {
		t.Fatal(err)
	}
	policy := EndpointPolicy{LookupIP: func(context.Context, string) ([]net.IP, error) { return []net.IP{net.ParseIP("127.0.0.1")}, nil }, AllowedPrivateHosts: map[string]struct{}{"runtime.test": {}}}
	if err := policy.ValidateDestination(context.Background(), endpoint); err != nil {
		t.Fatalf("explicit loopback allowlist error=%v", err)
	}
	policy.LookupIP = func(context.Context, string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("8.8.8.8")}, nil
	}
	if err := policy.ValidateDestination(context.Background(), endpoint); !errors.Is(err, ErrDisallowedNetwork) {
		t.Fatalf("mixed private/public destination error=%v, want ErrDisallowedNetwork", err)
	}
	policy.LookupIP = nil
	if err := policy.ValidateDestination(context.Background(), endpoint); !errors.Is(err, ErrTrustDependency) {
		t.Fatalf("missing resolver error=%v, want ErrTrustDependency", err)
	}
}

func TestEndpointPolicyDistinguishesResolverFailureAndEmptyResult(t *testing.T) {
	endpoint, err := ParseEndpoint("https://agent.example/a2a")
	if err != nil {
		t.Fatal(err)
	}
	dependency := EndpointPolicy{LookupIP: func(context.Context, string) ([]net.IP, error) { return nil, errors.New("resolver offline") }}
	if err := dependency.ValidateDestination(context.Background(), endpoint); !errors.Is(err, ErrTrustDependency) {
		t.Fatalf("resolver failure error=%v, want ErrTrustDependency", err)
	}
	empty := EndpointPolicy{LookupIP: func(context.Context, string) ([]net.IP, error) { return nil, nil }}
	if err := empty.ValidateDestination(context.Background(), endpoint); !errors.Is(err, ErrEndpointUnavailable) {
		t.Fatalf("empty resolver result error=%v, want ErrEndpointUnavailable", err)
	}
}

func TestTrustServiceCompletesSingleUseHTTPChallenge(t *testing.T) {
	store := newMemoryTrustStore()
	clockValue := time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC)
	var proof string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !strings.HasPrefix(request.URL.Path, "/.well-known/nekiro/challenges/") {
			http.NotFound(writer, request)
			return
		}
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(proof))
	}))
	defer server.Close()
	service, err := NewTrustService(store, ownerReader{agentID: "agent-a", ownerID: "owner-a", endpoint: server.URL + "/a2a"}, func() time.Time { return clockValue }, EndpointPolicy{LookupIP: func(context.Context, string) ([]net.IP, error) { return []net.IP{net.ParseIP("127.0.0.1")}, nil }, AllowedPrivateHosts: map[string]struct{}{"127.0.0.1": {}}}, server.Client(), time.Minute, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := service.CreateBindingForCaller(context.Background(), AuthenticatedCaller{ID: "owner-a"}, "provider-a", "agent-a", "1.0.0", server.URL+"/a2a", VerificationMethodHTTPWellKnown)
	if err != nil {
		t.Fatal(err)
	}
	challenge, err := service.CreateChallengeForCaller(context.Background(), AuthenticatedCaller{ID: "owner-a"}, "provider-a", binding.BindingID)
	if err != nil {
		t.Fatal(err)
	}
	if challenge.Proof == "" || strings.Contains(challenge.ChallengeURL, challenge.Proof) {
		t.Fatalf("challenge response leaked or omitted proof: %#v", challenge)
	}
	proof = challenge.Proof
	verified, err := service.CompleteChallengeForCaller(context.Background(), AuthenticatedCaller{ID: "owner-a"}, "provider-a", binding.BindingID, challenge.ChallengeID)
	if err != nil {
		t.Fatal(err)
	}
	wantDigest := sha256.Sum256([]byte(proof))
	if verified.VerificationStatus != VerificationVerified || verified.VerificationEvidenceDigest == nil || *verified.VerificationEvidenceDigest != wantDigest {
		t.Fatalf("verified binding=%#v", verified)
	}
	if _, err := service.GetBindingForCaller(context.Background(), AuthenticatedCaller{ID: "other-owner"}, "provider-a", binding.BindingID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-owner provider read error=%v, want ErrForbidden", err)
	}
	if _, err := service.CompleteChallengeForCaller(context.Background(), AuthenticatedCaller{ID: "owner-a"}, "provider-a", binding.BindingID, challenge.ChallengeID); !errors.Is(err, ErrTrustConflict) {
		t.Fatalf("verified binding completion error=%v, want ErrTrustConflict", err)
	}
}

func TestTrustServiceRecordsWrongProofWithoutReturningSuccess(t *testing.T) {
	store := newMemoryTrustStore()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) { _, _ = writer.Write([]byte("wrong")) }))
	defer server.Close()
	service, err := NewTrustService(store, ownerReader{agentID: "agent-a", ownerID: "provider-a", endpoint: server.URL + "/a2a"}, time.Now, EndpointPolicy{LookupIP: func(context.Context, string) ([]net.IP, error) { return []net.IP{net.ParseIP("127.0.0.1")}, nil }, AllowedPrivateHosts: map[string]struct{}{"127.0.0.1": {}}}, server.Client(), time.Minute, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := service.CreateBindingForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", "agent-a", "1.0.0", server.URL+"/a2a", VerificationMethodHTTPWellKnown)
	if err != nil {
		t.Fatal(err)
	}
	challenge, err := service.CreateChallengeForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", binding.BindingID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CompleteChallengeForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", binding.BindingID, challenge.ChallengeID); !errors.Is(err, ErrWrongProof) {
		t.Fatalf("wrong proof error=%v, want ErrWrongProof", err)
	}
	failed, err := service.GetBindingForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", binding.BindingID)
	if err != nil {
		t.Fatal(err)
	}
	if failed.VerificationStatus != VerificationFailed || failed.VerificationFailureCode == nil || *failed.VerificationFailureCode != "wrong_proof" {
		t.Fatalf("failed binding=%#v", failed)
	}
	if _, err := service.CompleteChallengeForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", binding.BindingID, challenge.ChallengeID); !errors.Is(err, ErrChallengeReused) {
		t.Fatalf("second failed completion error=%v, want ErrChallengeReused", err)
	}
}

func TestTrustServiceCannotCreateNewChallengeForVerifiedBinding(t *testing.T) {
	store := newMemoryTrustStore()
	service, err := NewTrustService(store, ownerReader{agentID: "agent-a", ownerID: "provider-a", endpoint: "https://agent.example/a2a"}, time.Now, EndpointPolicy{}, http.DefaultClient, time.Minute, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := service.CreateBindingForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", "agent-a", "1.0.0", "https://agent.example/a2a", VerificationMethodHTTPWellKnown)
	if err != nil {
		t.Fatal(err)
	}
	binding.VerificationStatus = VerificationVerified
	store.bindings[binding.BindingID] = binding
	if _, err := service.CreateChallengeForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", binding.BindingID); !errors.Is(err, ErrTrustConflict) {
		t.Fatalf("verified binding challenge error=%v, want ErrTrustConflict", err)
	}
}

func TestTrustServiceRejectsCardEndpointMismatchAndSuspendedProvider(t *testing.T) {
	store := newMemoryTrustStore()
	service, err := NewTrustService(store, ownerReader{agentID: "agent-a", ownerID: "display-owner", endpoint: "https://agent.example/declared"}, time.Now, EndpointPolicy{LookupIP: func(context.Context, string) ([]net.IP, error) { return []net.IP{net.ParseIP("8.8.8.8")}, nil }}, http.DefaultClient, time.Minute, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateBindingForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", "agent-a", "1.0.0", "https://agent.example/other", VerificationMethodHTTPWellKnown); !errors.Is(err, ErrEndpointInvalid) {
		t.Fatalf("Card endpoint mismatch error=%v, want ErrEndpointInvalid", err)
	}
	store.providers["provider-a"] = Provider{ProviderID: "provider-a", OwnerIdentity: "provider-a", VerificationStatus: VerificationSuspended}
	store.bindings["binding-1"] = EndpointBinding{BindingID: "binding-1", ProviderID: "provider-a", AgentID: "agent-a", VerificationStatus: VerificationPending}
	if _, err := service.CreateChallengeForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", "binding-1"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("suspended provider challenge error=%v, want ErrForbidden", err)
	}
}

func TestTrustServiceBindsOneAgentToOneProvider(t *testing.T) {
	store := newMemoryTrustStore()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) { _, _ = writer.Write([]byte("proof")) }))
	defer server.Close()
	service, err := NewTrustService(store, ownerReader{agentID: "agent-a", ownerID: "display-owner", endpoint: server.URL + "/a2a"}, time.Now, EndpointPolicy{LookupIP: func(context.Context, string) ([]net.IP, error) { return []net.IP{net.ParseIP("127.0.0.1")}, nil }, AllowedPrivateHosts: map[string]struct{}{"127.0.0.1": {}}}, server.Client(), time.Minute, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateBindingForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", "agent-a", "1.0.0", server.URL+"/a2a", VerificationMethodHTTPWellKnown); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateBindingForCaller(context.Background(), AuthenticatedCaller{ID: "provider-b"}, "provider-b", "agent-a", "1.0.0", server.URL+"/a2a", VerificationMethodHTTPWellKnown); !errors.Is(err, ErrTrustConflict) {
		t.Fatalf("second provider claim error=%v, want ErrTrustConflict", err)
	}
}

func TestTrustServiceRecordsEndpointUnavailable(t *testing.T) {
	store := newMemoryTrustStore()
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	endpoint := server.URL + "/a2a"
	service, err := NewTrustService(store, ownerReader{agentID: "agent-a", ownerID: "provider-a", endpoint: endpoint}, time.Now, EndpointPolicy{LookupIP: func(context.Context, string) ([]net.IP, error) { return []net.IP{net.ParseIP("127.0.0.1")}, nil }, AllowedPrivateHosts: map[string]struct{}{"127.0.0.1": {}}}, server.Client(), time.Minute, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := service.CreateBindingForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", "agent-a", "1.0.0", endpoint, VerificationMethodHTTPWellKnown)
	if err != nil {
		t.Fatal(err)
	}
	challenge, err := service.CreateChallengeForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", binding.BindingID)
	if err != nil {
		t.Fatal(err)
	}
	server.Close()
	if _, err := service.CompleteChallengeForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", binding.BindingID, challenge.ChallengeID); !errors.Is(err, ErrEndpointUnavailable) {
		t.Fatalf("unavailable endpoint error=%v, want ErrEndpointUnavailable", err)
	}
	failed, err := service.GetBindingForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", binding.BindingID)
	if err != nil {
		t.Fatal(err)
	}
	if failed.VerificationStatus != VerificationFailed || failed.VerificationFailureCode == nil || *failed.VerificationFailureCode != "endpoint_unavailable" {
		t.Fatalf("unavailable binding=%#v", failed)
	}
}

func TestTrustServiceRecordsExpiryAfterVerificationStarts(t *testing.T) {
	store := newMemoryTrustStore()
	now := time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC)
	current := now
	proofReady := make(chan string, 1)
	requestStarted := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		close(requestStarted)
		proof := <-proofReady
		<-release
		_, _ = writer.Write([]byte(proof))
	}))
	defer server.Close()
	service, err := NewTrustService(store, ownerReader{agentID: "agent-a", ownerID: "provider-a", endpoint: server.URL + "/a2a"}, func() time.Time { return current }, EndpointPolicy{LookupIP: func(context.Context, string) ([]net.IP, error) { return []net.IP{net.ParseIP("127.0.0.1")}, nil }, AllowedPrivateHosts: map[string]struct{}{"127.0.0.1": {}}}, server.Client(), time.Minute, 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := service.CreateBindingForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", "agent-a", "1.0.0", server.URL+"/a2a", VerificationMethodHTTPWellKnown)
	if err != nil {
		t.Fatal(err)
	}
	challenge, err := service.CreateChallengeForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", binding.BindingID)
	if err != nil {
		t.Fatal(err)
	}
	completion := make(chan error, 1)
	go func() {
		_, completionErr := service.CompleteChallengeForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", binding.BindingID, challenge.ChallengeID)
		completion <- completionErr
	}()
	<-requestStarted
	current = now.Add(2 * time.Minute)
	proofReady <- challenge.Proof
	close(release)
	if err := <-completion; !errors.Is(err, ErrChallengeExpired) {
		t.Fatalf("finish-after-expiry error=%v, want ErrChallengeExpired", err)
	}
	failed, err := service.GetBindingForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", binding.BindingID)
	if err != nil {
		t.Fatal(err)
	}
	if failed.VerificationStatus != VerificationFailed || failed.VerificationFailureCode == nil || *failed.VerificationFailureCode != "challenge_expired" {
		t.Fatalf("expired binding=%#v", failed)
	}
}

func TestTrustServiceRecordsAlreadyExpiredChallengeAsBindingFailure(t *testing.T) {
	store := newMemoryTrustStore()
	now := time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC)
	current := now
	service, err := NewTrustService(store, ownerReader{agentID: "agent-a", ownerID: "provider-a", endpoint: "https://agent.example/a2a"}, func() time.Time { return current }, EndpointPolicy{LookupIP: func(context.Context, string) ([]net.IP, error) { return []net.IP{net.ParseIP("8.8.8.8")}, nil }}, http.DefaultClient, time.Minute, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := service.CreateBindingForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", "agent-a", "1.0.0", "https://agent.example/a2a", VerificationMethodHTTPWellKnown)
	if err != nil {
		t.Fatal(err)
	}
	challenge, err := service.CreateChallengeForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", binding.BindingID)
	if err != nil {
		t.Fatal(err)
	}
	current = challenge.ExpiresAt
	if _, err := service.CompleteChallengeForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", binding.BindingID, challenge.ChallengeID); !errors.Is(err, ErrChallengeExpired) {
		t.Fatalf("expired challenge error=%v, want ErrChallengeExpired", err)
	}
	failed, err := service.GetBindingForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", binding.BindingID)
	if err != nil {
		t.Fatal(err)
	}
	if failed.VerificationStatus != VerificationFailed || failed.VerificationFailureCode == nil || *failed.VerificationFailureCode != "challenge_expired" {
		t.Fatalf("already expired binding=%#v", failed)
	}
}

func TestTrustServiceRejectsRedirectAsDistinctFailure(t *testing.T) {
	store := newMemoryTrustStore()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, "https://other.example/proof", http.StatusFound)
	}))
	defer server.Close()
	service, err := NewTrustService(store, ownerReader{agentID: "agent-a", ownerID: "provider-a", endpoint: server.URL + "/a2a"}, time.Now, EndpointPolicy{LookupIP: func(context.Context, string) ([]net.IP, error) { return []net.IP{net.ParseIP("127.0.0.1")}, nil }, AllowedPrivateHosts: map[string]struct{}{"127.0.0.1": {}}}, server.Client(), time.Minute, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := service.CreateBindingForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", "agent-a", "1.0.0", server.URL+"/a2a", VerificationMethodHTTPWellKnown)
	if err != nil {
		t.Fatal(err)
	}
	challenge, err := service.CreateChallengeForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", binding.BindingID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CompleteChallengeForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", binding.BindingID, challenge.ChallengeID); !errors.Is(err, ErrRedirectNotAllowed) {
		t.Fatalf("redirect error=%v, want ErrRedirectNotAllowed", err)
	}
}

func TestTrustServicePinsHTTPSDestinationWithoutCustomTLSDialBypass(t *testing.T) {
	store := newMemoryTrustStore()
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) { _, _ = writer.Write([]byte("proof")) }))
	defer server.Close()
	service, err := NewTrustService(store, ownerReader{agentID: "agent-a", ownerID: "provider-a", endpoint: server.URL + "/a2a"}, time.Now, EndpointPolicy{LookupIP: func(context.Context, string) ([]net.IP, error) { return []net.IP{net.ParseIP("127.0.0.1")}, nil }, AllowedPrivateHosts: map[string]struct{}{"127.0.0.1": {}}}, server.Client(), time.Minute, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := service.CreateBindingForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", "agent-a", "1.0.0", server.URL+"/a2a", VerificationMethodHTTPWellKnown)
	if err != nil {
		t.Fatal(err)
	}
	challenge, err := service.CreateChallengeForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", binding.BindingID)
	if err != nil {
		t.Fatal(err)
	}
	server.Config.Handler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) { _, _ = writer.Write([]byte(challenge.Proof)) })
	verified, err := service.CompleteChallengeForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", binding.BindingID, challenge.ChallengeID)
	if err != nil || verified.VerificationStatus != VerificationVerified {
		t.Fatalf("HTTPS verification = %#v, %v", verified, err)
	}
}

func TestTrustServiceConcurrentChallengesCannotOverwriteVerifiedBinding(t *testing.T) {
	store := newMemoryTrustStore()
	proofs := make(map[string]string)
	var lock sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		lock.Lock()
		proof := proofs[strings.TrimPrefix(request.URL.Path, "/.well-known/nekiro/challenges/")]
		lock.Unlock()
		_, _ = writer.Write([]byte(proof))
	}))
	defer server.Close()
	service, err := NewTrustService(store, ownerReader{agentID: "agent-a", ownerID: "provider-a", endpoint: server.URL + "/a2a"}, time.Now, EndpointPolicy{LookupIP: func(context.Context, string) ([]net.IP, error) { return []net.IP{net.ParseIP("127.0.0.1")}, nil }, AllowedPrivateHosts: map[string]struct{}{"127.0.0.1": {}}}, server.Client(), time.Minute, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := service.CreateBindingForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", "agent-a", "1.0.0", server.URL+"/a2a", VerificationMethodHTTPWellKnown)
	if err != nil {
		t.Fatal(err)
	}
	challenges := make([]contracts.VerificationChallengeResponse, 2)
	for index := range challenges {
		challenges[index], err = service.CreateChallengeForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", binding.BindingID)
		if err != nil {
			t.Fatal(err)
		}
		lock.Lock()
		proofs[challenges[index].ChallengeID] = challenges[index].Proof
		lock.Unlock()
	}
	results := make(chan error, len(challenges))
	var wait sync.WaitGroup
	for _, challenge := range challenges {
		wait.Add(1)
		go func(challenge contracts.VerificationChallengeResponse) {
			defer wait.Done()
			_, completionErr := service.CompleteChallengeForCaller(context.Background(), AuthenticatedCaller{ID: "provider-a"}, "provider-a", binding.BindingID, challenge.ChallengeID)
			results <- completionErr
		}(challenge)
	}
	wait.Wait()
	close(results)
	var successes, conflicts int
	for completionErr := range results {
		switch {
		case completionErr == nil:
			successes++
		case errors.Is(completionErr, ErrTrustConflict):
			conflicts++
		default:
			t.Fatalf("concurrent completion error=%v", completionErr)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent completion outcomes success=%d conflict=%d", successes, conflicts)
	}
}

func TestBindingVerificationCannotOverwriteVerifiedFact(t *testing.T) {
	store := newMemoryTrustStore()
	now := time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC)
	store.bindings["binding-1"] = EndpointBinding{BindingID: "binding-1", VerificationStatus: VerificationPending}
	digest := sha256.Sum256([]byte("proof"))
	if _, err := store.SetBindingVerification(context.Background(), "binding-1", VerificationVerified, nil, &digest, now); err != nil {
		t.Fatal(err)
	}
	failure := "wrong_proof"
	if _, err := store.SetBindingVerification(context.Background(), "binding-1", VerificationFailed, &failure, nil, now.Add(time.Second)); !errors.Is(err, ErrTrustConflict) {
		t.Fatalf("verified overwrite error=%v, want ErrTrustConflict", err)
	}
}

type ownerReader struct{ agentID, ownerID, endpoint string }

func (reader ownerReader) Get(_ context.Context, agentID, version string) (AgentVersion, error) {
	if agentID != reader.agentID || version != "1.0.0" {
		return AgentVersion{}, ErrNotFound
	}
	return AgentVersion{Card: contracts.AgentCard{AgentID: agentID, Version: version, Owner: contracts.AgentOwner{ID: reader.ownerID}, Protocol: contracts.AgentProtocol{Endpoint: reader.endpoint}}}, nil
}

type memoryTrustStore struct {
	mu         sync.Mutex
	providers  map[string]Provider
	bindings   map[string]EndpointBinding
	challenges map[string]VerificationChallenge
}

func newMemoryTrustStore() *memoryTrustStore {
	return &memoryTrustStore{providers: make(map[string]Provider), bindings: make(map[string]EndpointBinding), challenges: make(map[string]VerificationChallenge)}
}

func (store *memoryTrustStore) CreateBinding(_ context.Context, provider Provider, binding EndpointBinding) (EndpointBinding, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if existing, exists := store.providers[provider.ProviderID]; exists && existing.OwnerIdentity != provider.OwnerIdentity {
		return EndpointBinding{}, ErrForbidden
	}
	if existing, exists := store.providers[provider.ProviderID]; exists && existing.VerificationStatus == VerificationSuspended {
		return EndpointBinding{}, ErrForbidden
	}
	for _, existing := range store.bindings {
		if existing.AgentID == binding.AgentID && existing.ProviderID != binding.ProviderID {
			return EndpointBinding{}, ErrTrustConflict
		}
	}
	store.providers[provider.ProviderID] = provider
	if _, exists := store.bindings[binding.BindingID]; exists {
		return EndpointBinding{}, ErrTrustConflict
	}
	store.bindings[binding.BindingID] = binding
	return binding, nil
}

func (store *memoryTrustStore) GetProvider(_ context.Context, providerID string) (Provider, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	provider, exists := store.providers[providerID]
	if !exists {
		return Provider{}, ErrProviderNotFound
	}
	return provider, nil
}

func (store *memoryTrustStore) GetBinding(_ context.Context, _, bindingID string) (EndpointBinding, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	binding, exists := store.bindings[bindingID]
	if !exists {
		return EndpointBinding{}, ErrBindingNotFound
	}
	return binding, nil
}

func (store *memoryTrustStore) CreateChallenge(_ context.Context, challenge VerificationChallenge) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.challenges[challenge.ChallengeID] = challenge
	return nil
}

func (store *memoryTrustStore) ReserveChallenge(_ context.Context, bindingID, challengeID string, now time.Time) (VerificationChallenge, EndpointBinding, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	challenge, exists := store.challenges[challengeID]
	if !exists || challenge.BindingID != bindingID {
		return VerificationChallenge{}, EndpointBinding{}, ErrChallengeNotFound
	}
	if challenge.UsedAt != nil {
		return VerificationChallenge{}, EndpointBinding{}, ErrChallengeReused
	}
	if !now.Before(challenge.ExpiresAt) {
		return VerificationChallenge{}, EndpointBinding{}, ErrChallengeExpired
	}
	binding, exists := store.bindings[bindingID]
	if !exists {
		return VerificationChallenge{}, EndpointBinding{}, ErrBindingNotFound
	}
	if binding.VerificationStatus == VerificationVerified || binding.VerificationStatus == VerificationRevoked {
		return VerificationChallenge{}, EndpointBinding{}, ErrTrustConflict
	}
	challenge.UsedAt = &now
	store.challenges[challengeID] = challenge
	return challenge, binding, nil
}

func (store *memoryTrustStore) SetBindingVerification(_ context.Context, bindingID string, status VerificationStatus, failureCode *string, digest *[32]byte, at time.Time) (EndpointBinding, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	binding, exists := store.bindings[bindingID]
	if !exists {
		return EndpointBinding{}, ErrBindingNotFound
	}
	if binding.VerificationStatus == VerificationVerified || binding.VerificationStatus == VerificationRevoked {
		return EndpointBinding{}, ErrTrustConflict
	}
	binding.VerificationStatus = status
	binding.VerificationFailureCode = failureCode
	binding.VerificationEvidenceDigest = digest
	binding.UpdatedAt = at
	if status == VerificationVerified {
		binding.VerifiedAt = &at
	}
	store.bindings[bindingID] = binding
	return binding, nil
}
