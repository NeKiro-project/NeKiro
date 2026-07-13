package contracts

import (
	"path/filepath"
	"testing"
	"time"
)

func TestCatalogV2OperationsDeclareSecurityTraceAndExactErrors(t *testing.T) {
	document := loadOpenAPIDocument(t, filepath.Join("openapi", "control-plane.v2.yaml"))
	tests := []struct {
		path     string
		method   string
		success  int
		failures []int
	}{
		{path: "/v2/agents", method: "POST", success: 201, failures: []int{400, 401, 403, 409, 503}},
		{path: "/v2/agents", method: "GET", success: 200, failures: []int{400, 401, 503}},
		{path: "/v2/agents/{agentId}/versions/{version}", method: "GET", success: 200, failures: []int{400, 401, 403, 404, 503}},
		{path: "/v2/agents/{agentId}/versions/{version}/publish", method: "POST", success: 200, failures: []int{400, 401, 403, 404, 409, 503}},
		{path: "/v2/agents/{agentId}/versions/{version}/disable", method: "POST", success: 200, failures: []int{400, 401, 403, 404, 503}},
	}
	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			path := document.Paths.Find(test.path)
			operation := path.Get
			if test.method == "POST" {
				operation = path.Post
			}
			if operation == nil || operation.Security == nil || len(*operation.Security) != 1 {
				t.Fatal("Bearer security requirement is missing")
			}
			security := (*operation.Security)[0]
			if _, exists := security["bearerAuth"]; !exists {
				t.Fatalf("security = %#v, want bearerAuth", security)
			}
			statuses := append([]int{test.success}, test.failures...)
			for _, status := range statuses {
				response := operation.Responses.Status(status)
				if response == nil || response.Value == nil {
					t.Errorf("response %d is missing", status)
					continue
				}
				if response.Value.Headers["x-nek-trace-id"] == nil {
					t.Errorf("response %d trace header is missing", status)
				}
			}
		})
	}
}

func TestCatalogV2GoMappingsAndDiscoveryPolicy(t *testing.T) {
	document := loadOpenAPIDocument(t, filepath.Join("openapi", "control-plane.v2.yaml"))
	card := validAgentCard()
	entry := CatalogEntry{Card: card, PublicationStatus: "published", RegisteredAt: time.Now().UTC()}
	register := document.Paths.Find("/v2/agents").Post
	validateOpenAPIValue(t, register.RequestBody.Value.Content["application/json"].Schema, RegisterAgentRequest{Card: card})
	validateOpenAPIValue(t, register.Responses.Status(201).Value.Content["application/json"].Schema, entry)
	search := document.Paths.Find("/v2/agents").Get
	validateOpenAPIValue(t, search.Responses.Status(200).Value.Content["application/json"].Schema, SearchAgentsResponse{Items: []CatalogEntry{entry}})

	var foundLimit bool
	for _, parameter := range search.Parameters {
		if parameter.Value.Name == "limit" {
			foundLimit = true
			if parameter.Value.Schema.Value.Default != float64(DiscoveryDefaultLimit) {
				t.Fatalf("limit default = %#v, want %d", parameter.Value.Schema.Value.Default, DiscoveryDefaultLimit)
			}
			if parameter.Value.Schema.Value.Min == nil || *parameter.Value.Schema.Value.Min != float64(DiscoveryMinimumLimit) || parameter.Value.Schema.Value.Max == nil || *parameter.Value.Schema.Value.Max != float64(DiscoveryMaximumLimit) {
				t.Fatalf("limit bounds = %#v", parameter.Value.Schema.Value)
			}
		}
	}
	if !foundLimit {
		t.Fatal("discovery limit parameter is missing")
	}
}

func TestHistoricalCatalogV1RemainsReadable(t *testing.T) {
	document := loadOpenAPIDocument(t, filepath.Join("openapi", "control-plane.v1.yaml"))
	if document.Info.Version != "1.0.0" {
		t.Fatalf("historical Northbound version = %q", document.Info.Version)
	}
}
