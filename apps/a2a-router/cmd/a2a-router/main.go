package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"

	"github.com/Nene7ko/NeKiro/apps/a2a-router/internal/api"
	"github.com/Nene7ko/NeKiro/apps/a2a-router/internal/auth"
	"github.com/Nene7ko/NeKiro/apps/a2a-router/internal/config"
	"github.com/Nene7ko/NeKiro/apps/a2a-router/internal/resolution"
	a2atransport "github.com/Nene7ko/NeKiro/apps/a2a-router/internal/transport/a2a"
)

func main() {
	if err := run(); err != nil {
		slog.Error("a2a-router failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	handler, err := newHandler(cfg, http.DefaultClient, http.DefaultClient)
	if err != nil {
		return err
	}
	server := &http.Server{Addr: cfg.ListenAddress, Handler: handler}
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func newHandler(cfg config.Config, doer resolution.HTTPDoer, agentHTTPClient *http.Client) (http.Handler, error) {
	authenticator, err := auth.NewStaticAuthenticator(cfg.RouterPrincipals)
	if err != nil {
		return nil, err
	}
	resolver, err := resolution.NewClient(doer, cfg.ControlPlaneResolveURL, cfg.ControlPlaneServiceToken, cfg.ControlPlaneResponseLimitBytes)
	if err != nil {
		return nil, err
	}
	transport, err := a2atransport.NewClient(agentHTTPClient)
	if err != nil {
		return nil, err
	}
	dispatch, err := api.NewDispatchHandlerWithTransport(authenticator, resolver, transport, cfg.InternalRequestLimitBytes, cfg.ResolutionDeadline)
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.Handle("GET /readyz", api.NewReadinessHandler())
	dispatch.RegisterRoutes(mux)
	return mux, nil
}
