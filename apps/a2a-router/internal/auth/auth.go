package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
)

var (
	ErrUnauthenticated = errors.New("router service authentication is required")
	ErrForbidden       = errors.New("router service principal is forbidden")
)

type Principal struct {
	ID          string
	TokenSHA256 string
}

type Caller struct {
	ID string
}

type StaticAuthenticator struct {
	digests map[string]string
}

func NewStaticAuthenticator(principals []Principal) (*StaticAuthenticator, error) {
	if len(principals) == 0 {
		return nil, errors.New("at least one Router principal is required")
	}
	digests := make(map[string]string, len(principals))
	ids := make(map[string]struct{}, len(principals))
	for _, principal := range principals {
		if !validIdentifier(principal.ID) {
			return nil, errors.New("Router principal id is invalid")
		}
		decoded, err := hex.DecodeString(principal.TokenSHA256)
		if err != nil || len(decoded) != sha256.Size || principal.TokenSHA256 != strings.ToLower(principal.TokenSHA256) {
			return nil, errors.New("Router principal tokenSha256 is invalid")
		}
		if _, exists := ids[principal.ID]; exists {
			return nil, errors.New("Router principal id is duplicated")
		}
		if _, exists := digests[principal.TokenSHA256]; exists {
			return nil, errors.New("Router principal tokenSha256 is duplicated")
		}
		ids[principal.ID] = struct{}{}
		digests[principal.TokenSHA256] = principal.ID
	}
	return &StaticAuthenticator{digests: digests}, nil
}

func (authenticator *StaticAuthenticator) Authenticate(request *http.Request) (Caller, error) {
	values := request.Header.Values("Authorization")
	if len(values) != 1 {
		return Caller{}, ErrUnauthenticated
	}
	value := values[0]
	if value == "" {
		return Caller{}, ErrUnauthenticated
	}
	token, ok := strings.CutPrefix(value, "Bearer ")
	if !ok || token == "" || strings.TrimSpace(token) != token {
		return Caller{}, ErrUnauthenticated
	}
	sum := sha256.Sum256([]byte(token))
	digest := hex.EncodeToString(sum[:])
	for configured, id := range authenticator.digests {
		if subtle.ConstantTimeCompare([]byte(configured), []byte(digest)) == 1 {
			return Caller{ID: id}, nil
		}
	}
	return Caller{}, ErrForbidden
}

func validIdentifier(value string) bool {
	if len(value) < 1 || len(value) > 128 {
		return false
	}
	for index, character := range []byte(value) {
		if character >= 'A' && character <= 'Z' || character >= 'a' && character <= 'z' || character >= '0' && character <= '9' || character == '.' || character == '_' || character == ':' || character == '-' {
			if index > 0 || character != '.' && character != '_' && character != ':' && character != '-' {
				continue
			}
		}
		return false
	}
	return true
}
