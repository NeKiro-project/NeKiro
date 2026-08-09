package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/NeKiro-project/NeKiro/apps/a2a-router/internal/api"
	"github.com/NeKiro-project/NeKiro/apps/a2a-router/internal/auth"
	"github.com/NeKiro-project/NeKiro/apps/a2a-router/internal/config"
	"github.com/NeKiro-project/NeKiro/apps/a2a-router/internal/configdirectory"
	"github.com/NeKiro-project/NeKiro/apps/a2a-router/internal/credential"
	"github.com/NeKiro-project/NeKiro/apps/a2a-router/internal/ledger"
	"github.com/NeKiro-project/NeKiro/apps/a2a-router/internal/nacosdirectory"
	"github.com/NeKiro-project/NeKiro/apps/a2a-router/internal/nested"
	"github.com/NeKiro-project/NeKiro/apps/a2a-router/internal/resolution"
	"github.com/NeKiro-project/NeKiro/apps/a2a-router/internal/routing"
	a2atransport "github.com/NeKiro-project/NeKiro/apps/a2a-router/internal/transport/a2a"
	configfile "github.com/NeKiro-project/NeKiro/config_center/file"
	confignacos "github.com/NeKiro-project/NeKiro/config_center/nacos"
	"github.com/NeKiro-project/NeKiro/registry"
	registrynacos "github.com/NeKiro-project/NeKiro/registry/nacos"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	if err := run(context.Background(), os.Args[1:], logger); err != nil {
		logger.Error("a2a-router failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, arguments []string, logger *slog.Logger) error {
	if len(arguments) == 0 {
		return errors.New("command is required: serve or migrate")
	}
	switch arguments[0] {
	case "serve":
		if len(arguments) != 1 {
			return errors.New("serve accepts no arguments")
		}
		return serve(ctx, logger)
	case "migrate":
		if len(arguments) != 2 || arguments[1] != "up" {
			return errors.New("migrate requires exactly one direction: up")
		}
		return migrate(ctx, arguments[1])
	default:
		return fmt.Errorf("unknown command %q", arguments[0])
	}
}

func serve(ctx context.Context, logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open Router Ledger database: %w", err)
	}
	defer pool.Close()
	ledgerStore, err := ledger.NewStore(pool)
	if err != nil {
		return err
	}
	if err := ledgerStore.Check(ctx); err != nil {
		return fmt.Errorf("router Ledger schema is not ready: %w", err)
	}
	var selector a2atransport.TargetSelector
	var directory interface {
		registry.InstanceDirectory
		api.ReadinessChecker
	}
	switch cfg.InstanceRoutingMode {
	case config.InstanceRoutingConfigCenterFile:
		reader, openErr := configfile.OpenReader(configfile.ReaderConfig{
			Root: cfg.ConfigCenterFileRoot, MaxPayloadBytes: cfg.ConfigCenterMaxPayloadBytes, SubscriptionBuffer: 1,
		})
		if openErr != nil {
			return fmt.Errorf("open Router Config Center reader: %w", openErr)
		}
		fileDirectory, directoryErr := configdirectory.New(reader, cfg.InstanceDirectoryKey)
		directory, err = fileDirectory, directoryErr
		if err != nil {
			_ = reader.Close()
			return fmt.Errorf("initialize Router instance directory: %w", err)
		}
	case config.InstanceRoutingNacos:
		nacosClient := newNacosHTTPClient(cfg.NacosRequestTimeout)
		reader, readerErr := confignacos.NewReader(confignacos.ReaderConfig{
			APIOrigin: cfg.NacosAPIOrigin, NamespaceID: cfg.NacosNamespaceID, GroupName: cfg.NacosConfigGroup,
			MaxPayloadBytes: cfg.NacosResponseLimitBytes, AuthMode: cfg.NacosAuthMode,
			AccessToken: cfg.NacosAccessToken, Executor: nacosClient,
		})
		if readerErr != nil {
			return fmt.Errorf("open Router Nacos Config Center reader: %w", readerErr)
		}
		nacosDirectory, directoryErr := nacosdirectory.New(reader, cfg.InstanceDirectoryKey, registrynacos.DirectoryConfig{
			APIOrigin: cfg.NacosAPIOrigin, NamespaceID: cfg.NacosNamespaceID, PortName: cfg.InstancePortName,
			MaxResponseBytes: cfg.NacosResponseLimitBytes, AuthMode: cfg.NacosAuthMode,
			AccessToken: cfg.NacosAccessToken, Executor: nacosClient,
		})
		directory, err = nacosDirectory, directoryErr
		if err != nil {
			_ = reader.Close()
			return fmt.Errorf("initialize Router Nacos instance directory: %w", err)
		}
	}
	if directory != nil {
		defer directory.Close()
		selector, err = routing.NewSnapshotSelector(directory, cfg.InstancePortName)
		if err != nil {
			return fmt.Errorf("initialize Router instance selector: %w", err)
		}
	}
	var readiness []api.ReadinessChecker
	if directory != nil {
		if err := directory.Check(ctx); err != nil {
			return fmt.Errorf("Router instance directory is not ready: %w", err)
		}
		readiness = append(readiness, directory)
	}
	handler, err := newHandlerWithTargetSelector(cfg, http.DefaultClient, http.DefaultClient, ledgerStore, selector, readiness...)
	if err != nil {
		return err
	}
	server := &http.Server{Addr: cfg.ListenAddress, Handler: handler}
	if logger != nil {
		logger.Info("a2a-router listening", "address", cfg.ListenAddress)
	}
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func newNacosHTTPClient(timeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DisableKeepAlives = true
	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("Nacos redirects are disabled")
		},
	}
}

func migrate(ctx context.Context, direction string) (returnErr error) {
	databaseURL, err := config.LoadDatabaseURL()
	if err != nil {
		return err
	}
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return errors.New("connect Router Ledger migration database")
	}
	defer func() {
		if closeErr := connection.Close(ctx); closeErr != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("close Router Ledger migration database: %w", closeErr))
		}
	}()
	if err := ledger.Migrate(ctx, connection, direction); err != nil {
		return errors.New("router Ledger migration failed")
	}
	return nil
}

func newHandler(cfg config.Config, doer resolution.HTTPDoer, agentHTTPClient *http.Client, ledgerAppender api.InvocationLedgerAppender) (http.Handler, error) {
	return newHandlerWithTargetSelector(cfg, doer, agentHTTPClient, ledgerAppender, nil)
}

func newHandlerWithTargetSelector(cfg config.Config, doer resolution.HTTPDoer, agentHTTPClient *http.Client, ledgerAppender api.InvocationLedgerAppender, selector a2atransport.TargetSelector, readiness ...api.ReadinessChecker) (http.Handler, error) {
	authenticator, err := auth.NewStaticAuthenticator(cfg.RouterPrincipals)
	if err != nil {
		return nil, err
	}
	resolver, err := resolution.NewClientWithVersionURL(doer, cfg.ControlPlaneResolveURL, cfg.ControlPlaneVersionURL, cfg.ControlPlaneServiceToken, cfg.ControlPlaneResponseLimitBytes)
	if err != nil {
		return nil, err
	}
	credentialIssuer, err := credential.NewIssuer(cfg.AgentCredential, time.Now, rand.Reader)
	if err != nil {
		return nil, err
	}
	var transport *a2atransport.Client
	if selector == nil {
		transport, err = a2atransport.NewClient(agentHTTPClient, credentialIssuer, cfg.InternalRequestLimitBytes, cfg.AgentResponseLimitBytes, cfg.A2AEventLimitBytes, cfg.SSEEventLimitBytes)
	} else {
		transport, err = a2atransport.NewClientWithTargetSelector(agentHTTPClient, credentialIssuer, selector, cfg.InternalRequestLimitBytes, cfg.AgentResponseLimitBytes, cfg.A2AEventLimitBytes, cfg.SSEEventLimitBytes)
	}
	if err != nil {
		return nil, err
	}
	var dispatch *api.DispatchHandler
	if ledgerAppender == nil {
		return nil, errors.New("router Ledger appender is required")
	}
	ledgerReader, ok := ledgerAppender.(api.LedgerReader)
	if !ok {
		return nil, errors.New("router Ledger reader is required")
	}
	nestedLedgerReader, ok := ledgerAppender.(api.NestedLedgerReader)
	if !ok {
		return nil, errors.New("router nested Ledger reader is required")
	}
	dispatch, err = api.NewDispatchHandlerWithTransportAndLedgerAndStreaming(authenticator, resolver, transport, ledgerAppender, cfg.SSEEventLimitBytes, cfg.InternalRequestLimitBytes, cfg.ResolutionDeadline)
	if err != nil {
		return nil, err
	}
	ledgerHandler, err := api.NewLedgerHandler(ledgerReader)
	if err != nil {
		return nil, err
	}
	agentBinding, err := nested.NewAgentBinding(cfg.AgentPrincipals)
	if err != nil {
		return nil, err
	}
	agentHandler, err := api.NewAgentInvocationHandler(agentBinding, nestedLedgerReader, resolver, dispatch, cfg.AgentRequestLimitBytes, cfg.AgentDeadline)
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.Handle("GET /readyz", api.NewReadinessHandler(readiness...))
	dispatch.RegisterRoutes(mux)
	agentHandler.RegisterRoutes(mux)
	if err := ledgerHandler.RegisterRoutes(mux, authenticator); err != nil {
		return nil, err
	}
	return mux, nil
}
