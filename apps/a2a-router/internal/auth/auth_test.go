package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"testing"
)

func TestStaticAuthenticatorAuthenticatesOnlyConfiguredBearer(t *testing.T) {
	token := "service-secret"
	sum := sha256.Sum256([]byte(token))
	authenticator, err := NewStaticAuthenticator([]Principal{{ID: "control-plane", TokenSHA256: hex.EncodeToString(sum[:])}})
	if err != nil {
		t.Fatal(err)
	}
	request, _ := http.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	caller, err := authenticator.Authenticate(request)
	if err != nil || caller.ID != "control-plane" {
		t.Fatalf("caller=%#v err=%v", caller, err)
	}
	cases := map[string]error{"": ErrUnauthenticated, "Basic abc": ErrUnauthenticated, "Bearer ": ErrUnauthenticated, "Bearer other": ErrForbidden}
	for value, want := range cases {
		request, _ := http.NewRequest(http.MethodPost, "/", nil)
		if value != "" {
			request.Header.Set("Authorization", value)
		}
		_, err := authenticator.Authenticate(request)
		if !errors.Is(err, want) {
			t.Fatalf("auth %q err=%v, want %v", value, err, want)
		}
	}
	request, _ = http.NewRequest(http.MethodPost, "/", nil)
	request.Header.Add("Authorization", "Bearer "+token)
	request.Header.Add("Authorization", "Bearer other")
	if _, err := authenticator.Authenticate(request); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("duplicate Authorization err=%v, want unauthenticated", err)
	}
}

func TestStaticAuthenticatorRejectsUnsafePrincipals(t *testing.T) {
	sum := sha256.Sum256([]byte("token"))
	valid := hex.EncodeToString(sum[:])
	other := sha256.Sum256([]byte("other"))
	tests := [][]Principal{
		nil,
		{{ID: "-bad", TokenSHA256: valid}},
		{{ID: "ok", TokenSHA256: "ABC"}},
		{{ID: "dup", TokenSHA256: valid}, {ID: "dup", TokenSHA256: hex.EncodeToString(other[:])}},
		{{ID: "a", TokenSHA256: valid}, {ID: "b", TokenSHA256: valid}},
	}
	for _, test := range tests {
		if _, err := NewStaticAuthenticator(test); err == nil {
			t.Fatalf("principals accepted: %#v", test)
		}
	}
}
