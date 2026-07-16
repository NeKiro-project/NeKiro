package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunAcceptsOnlyMigrateUpDirection(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, arguments := range [][]string{
		{"migrate"},
		{"migrate", "down"},
		{"migrate", "sideways"},
		{"migrate", "up", "extra"},
	} {
		err := run(context.Background(), arguments, logger)
		if err == nil || !strings.Contains(err.Error(), "migrate requires exactly one direction: up") {
			t.Fatalf("run(%q) error = %v", arguments, err)
		}
	}
}

func TestRouterHTTPClientDoesNotFollowRedirects(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.URL.Path == "/first" {
			http.Redirect(writer, request, "/second", http.StatusFound)
			return
		}
		t.Fatalf("unexpected redirected request to %s", request.URL.Path)
	}))
	defer server.Close()

	request, err := http.NewRequest(http.MethodGet, server.URL+"/first", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := newRouterHTTPClient().Do(request)
	if err != nil {
		t.Fatalf("Router request failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusFound || requests != 1 {
		t.Fatalf("redirect response = %d with %d requests, want 302 with one request", response.StatusCode, requests)
	}
}
