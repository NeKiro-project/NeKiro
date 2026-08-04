package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/NeKiro-project/NeKiro/contracts"
)

type PublicShareResolver interface {
	Resolve(context.Context, string) (contracts.PublicAgentShare, error)
}

type PublicShareHandler struct {
	service PublicShareResolver
	traces  *TraceGenerator
	logger  *slog.Logger
}

func NewPublicShareHandler(service PublicShareResolver, traces *TraceGenerator, logger *slog.Logger) (*PublicShareHandler, error) {
	if service == nil || traces == nil || logger == nil {
		return nil, errors.New("public share gateway dependencies are required")
	}
	return &PublicShareHandler{service: service, traces: traces, logger: logger}, nil
}

func (handler *PublicShareHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v4/public/agents/{publicAgentId}", handler.resolve)
	mux.HandleFunc("GET /v4/public/agents/", handler.resolve)
}

func (handler *PublicShareHandler) resolve(writer http.ResponseWriter, request *http.Request) {
	traceID := handler.traces.Next()
	writer.Header().Set(TraceHeader, string(traceID))
	publicAgentID := request.PathValue("publicAgentId")
	view, err := handler.service.Resolve(request.Context(), publicAgentID)
	if err != nil {
		code := catalogErrorCode(err)
		handler.logger.WarnContext(request.Context(), "public Agent resolution failed", "trace_id", traceID, "code", code)
		if writeErr := writePlatformError(writer, traceID, code); writeErr != nil {
			handler.logger.ErrorContext(request.Context(), "write public Agent error failed", "trace_id", traceID, "error", writeErr)
		}
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(writer).Encode(view); err != nil {
		handler.logger.ErrorContext(request.Context(), "write public Agent response failed", "trace_id", traceID, "error", err)
	}
}
