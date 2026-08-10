package api

import (
	"errors"
	"net/http"

	"github.com/NeKiro-project/NeKiro/apps/a2a-router/internal/auth"
	"github.com/NeKiro-project/NeKiro/contracts"
)

const RouterTopologyStatusPath = "/internal/v1/instance-topology/status"

type TopologyStatusReader interface {
	TopologyStatus() contracts.RouterTopologyStatusV1
}

type TopologyStatusHandler struct {
	reader TopologyStatusReader
}

func NewTopologyStatusHandler(reader TopologyStatusReader) (*TopologyStatusHandler, error) {
	if reader == nil {
		return nil, errors.New("Router topology status reader is required")
	}
	return &TopologyStatusHandler{reader: reader}, nil
}

func (handler *TopologyStatusHandler) RegisterRoutes(mux *http.ServeMux, authenticator Authenticator) error {
	if mux == nil {
		return errors.New("Router topology status mux is required")
	}
	if authenticator == nil {
		return errors.New("Router topology status authenticator is required")
	}
	mux.HandleFunc("GET "+RouterTopologyStatusPath, func(writer http.ResponseWriter, request *http.Request) {
		handler.serve(writer, request, authenticator)
	})
	return nil
}

func (handler *TopologyStatusHandler) serve(writer http.ResponseWriter, request *http.Request, authenticator Authenticator) {
	writer.Header().Set("Cache-Control", "no-store")
	traceID, err := newTraceID()
	if err != nil {
		http.Error(writer, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	writer.Header().Set(TraceHeader, string(traceID))
	if _, err := authenticator.Authenticate(request); err != nil {
		writeTopologyStatusError(writer, traceID, err)
		return
	}
	status := handler.reader.TopologyStatus()
	if err := contracts.ValidateRouterTopologyStatusV1(status); err != nil {
		writeTopologyStatusError(writer, traceID, err)
		return
	}
	_ = writeLedgerJSON(writer, http.StatusOK, status)
}

func writeTopologyStatusError(writer http.ResponseWriter, traceID contracts.TraceID, cause error) {
	status := http.StatusServiceUnavailable
	code := contracts.ErrorCodeDependency
	switch {
	case errors.Is(cause, auth.ErrUnauthenticated):
		status = http.StatusUnauthorized
		code = contracts.ErrorCodeUnauthenticated
	case errors.Is(cause, auth.ErrForbidden):
		status = http.StatusForbidden
		code = contracts.ErrorCodeForbidden
	}
	platformError, err := contracts.NewPreCorrelationPlatformErrorV4(code, traceID)
	if err != nil {
		writer.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	_ = writeLedgerJSON(writer, status, platformError)
}
