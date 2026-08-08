package api

import (
	"context"
	"encoding/json"
	"net/http"
)

type ReadinessChecker interface {
	Check(context.Context) error
}

func NewReadinessHandler(checkers ...ReadinessChecker) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		for _, checker := range checkers {
			if checker == nil || checker.Check(request.Context()) != nil {
				writer.WriteHeader(http.StatusServiceUnavailable)
				_ = json.NewEncoder(writer).Encode(map[string]string{"status": "not_ready"})
				return
			}
		}
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(map[string]string{"status": "ok"})
	})
}
