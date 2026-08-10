package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const nacosTLSMaterialLimit = 1 << 20

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
	directory, err := openInstanceDirectory(cfg)
	if err != nil {
		return err
	}
	if directory != nil {
		defer directory.Close()
		if cfg.NacosObserveEnabled {
			watchSelector, watchErr := routing.NewWatchSelector(directory, cfg.InstanceRoutingMode, cfg.InstancePortName, cfg.NacosMaxObservations)
			if watchErr != nil {
				return fmt.Errorf("initialize Router watch selector: %w", watchErr)
			}
			defer watchSelector.Close()
			selector = watchSelector
		} else {
			selector, err = routing.NewSnapshotSelector(directory, cfg.InstancePortName)
		}
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

type instanceDirectory interface {
	registry.InstanceDirectory
	api.ReadinessChecker
}

func openInstanceDirectory(cfg config.Config) (instanceDirectory, error) {
	switch cfg.InstanceRoutingMode {
	case config.InstanceRoutingConfigCenterFile:
		reader, openErr := configfile.OpenReader(configfile.ReaderConfig{
			Root: cfg.ConfigCenterFileRoot, MaxPayloadBytes: cfg.ConfigCenterMaxPayloadBytes, SubscriptionBuffer: 1,
		})
		if openErr != nil {
			return nil, fmt.Errorf("open Router Config Center reader: %w", openErr)
		}
		directory, err := configdirectory.New(reader, cfg.InstanceDirectoryKey)
		if err != nil {
			_ = reader.Close()
			return nil, fmt.Errorf("initialize Router instance directory: %w", err)
		}
		return directory, nil
	case config.InstanceRoutingNacos:
		nacosClient, clientErr := newNacosHTTPClient(cfg)
		if clientErr != nil {
			return nil, fmt.Errorf("initialize Router Nacos HTTP transport security: %w", clientErr)
		}
		reader, readerErr := confignacos.NewReader(confignacos.ReaderConfig{
			APIOrigin: cfg.NacosAPIOrigin, NamespaceID: cfg.NacosNamespaceID, GroupName: cfg.NacosConfigGroup,
			MaxPayloadBytes: cfg.NacosResponseLimitBytes, AuthMode: cfg.NacosAuthMode,
			AccessToken: cfg.NacosAccessToken, Executor: nacosClient,
		})
		if readerErr != nil {
			return nil, fmt.Errorf("open Router Nacos Config Center reader: %w", readerErr)
		}
		var subscriber registrynacos.NamingSubscriptionExecutor
		if cfg.NacosObserveEnabled {
			var headers map[string]string
			if cfg.NacosAuthMode == config.NacosAuthAccessToken {
				headers = map[string]string{"accessToken": cfg.NacosAccessToken}
			}
			transportCredentials, credentialsErr := newNacosGRPCTransportCredentials(cfg)
			if credentialsErr != nil {
				_ = reader.Close()
				return nil, fmt.Errorf("initialize Router Nacos gRPC transport security: %w", credentialsErr)
			}
			grpcExecutor, grpcErr := registrynacos.NewGRPCExecutor(registrynacos.GRPCExecutorConfig{
				Target: cfg.NacosGRPCTarget, ClientIP: cfg.NacosGRPCClientIP,
				RequestTimeout: cfg.NacosGRPCRequestTimeout, TransportCredentials: transportCredentials,
				RequestHeaders: headers,
			})
			if grpcErr != nil {
				_ = reader.Close()
				return nil, fmt.Errorf("initialize Router Nacos gRPC executor: %w", grpcErr)
			}
			subscriber = grpcExecutor
		}
		directory, err := nacosdirectory.New(reader, cfg.InstanceDirectoryKey, registrynacos.DirectoryConfig{
			APIOrigin: cfg.NacosAPIOrigin, NamespaceID: cfg.NacosNamespaceID, PortName: cfg.InstancePortName,
			MaxResponseBytes: cfg.NacosResponseLimitBytes, AuthMode: cfg.NacosAuthMode,
			AccessToken: cfg.NacosAccessToken, Executor: nacosClient,
			Subscriber: subscriber, PendingChanges: cfg.NacosPendingChanges,
		})
		if err != nil {
			_ = reader.Close()
			return nil, fmt.Errorf("initialize Router Nacos instance directory: %w", err)
		}
		return directory, nil
	}
	return nil, nil
}

func newNacosGRPCTransportCredentials(cfg config.Config) (credentials.TransportCredentials, error) {
	switch cfg.NacosGRPCTransportSecurity {
	case config.NacosGRPCSecurityInsecure:
		if cfg.NacosGRPCTLSCAFile != "" || cfg.NacosGRPCTLSServerName != "" ||
			cfg.NacosGRPCTLSClientCertFile != "" || cfg.NacosGRPCTLSClientKeyFile != "" {
			return nil, errors.New("Nacos gRPC insecure mode cannot include TLS material")
		}
		return insecure.NewCredentials(), nil
	case config.NacosGRPCSecurityTLS:
		if cfg.NacosGRPCTLSClientCertFile != "" || cfg.NacosGRPCTLSClientKeyFile != "" {
			return nil, errors.New("Nacos gRPC TLS mode cannot include a client certificate")
		}
	case config.NacosGRPCSecurityMTLS:
		if cfg.NacosGRPCTLSClientCertFile == "" || cfg.NacosGRPCTLSClientKeyFile == "" {
			return nil, errors.New("Nacos gRPC mTLS client certificate and key are required")
		}
	default:
		return nil, errors.New("Nacos gRPC transport security mode is unsupported")
	}
	if cfg.NacosGRPCTLSServerName == "" {
		return nil, errors.New("Nacos gRPC TLS server name is required")
	}
	tlsConfig, err := newNacosTLSConfig(
		cfg.NacosGRPCTLSCAFile,
		cfg.NacosGRPCTLSServerName,
		cfg.NacosGRPCTLSClientCertFile,
		cfg.NacosGRPCTLSClientKeyFile,
	)
	if err != nil {
		return nil, err
	}
	return credentials.NewTLS(tlsConfig), nil
}

func newNacosTLSConfig(caFile, serverName, clientCertFile, clientKeyFile string) (*tls.Config, error) {
	if serverName == "" {
		return nil, errors.New("Nacos TLS server name is required")
	}
	if (clientCertFile == "") != (clientKeyFile == "") {
		return nil, errors.New("Nacos TLS client certificate and key must be configured together")
	}
	caPEM, err := readNacosTLSMaterial(caFile, "CA bundle")
	if err != nil {
		return nil, err
	}
	defer clear(caPEM)
	roots, err := newNacosTLSCertPool(caPEM)
	if err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    roots,
		ServerName: serverName,
	}
	if clientCertFile == "" {
		return tlsConfig, nil
	}
	certificatePEM, err := readNacosTLSMaterial(clientCertFile, "client certificate")
	if err != nil {
		return nil, err
	}
	defer clear(certificatePEM)
	keyPEM, err := readNacosTLSMaterial(clientKeyFile, "client key")
	if err != nil {
		return nil, err
	}
	defer clear(keyPEM)
	clientCertificate, err := tls.X509KeyPair(certificatePEM, keyPEM)
	if err != nil {
		return nil, errors.New("Nacos TLS client certificate or key is invalid")
	}
	tlsConfig.Certificates = []tls.Certificate{clientCertificate}
	return tlsConfig, nil
}

func newNacosTLSCertPool(caPEM []byte) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	remaining := caPEM
	certificates := 0
	for len(bytes.TrimSpace(remaining)) > 0 {
		remaining = bytes.TrimSpace(remaining)
		if !bytes.HasPrefix(remaining, []byte("-----BEGIN CERTIFICATE-----")) {
			return nil, errors.New("Nacos TLS CA bundle is invalid")
		}
		block, rest := pem.Decode(remaining)
		if block == nil || block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			return nil, errors.New("Nacos TLS CA bundle is invalid")
		}
		certificate, err := x509.ParseCertificate(block.Bytes)
		if err != nil || !certificate.IsCA || certificate.KeyUsage&x509.KeyUsageCertSign == 0 {
			return nil, errors.New("Nacos TLS CA bundle is invalid")
		}
		pool.AddCert(certificate)
		certificates++
		remaining = rest
	}
	if certificates == 0 {
		return nil, errors.New("Nacos TLS CA bundle is invalid")
	}
	return pool, nil
}

func readNacosTLSMaterial(path, label string) ([]byte, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, fmt.Errorf("Nacos TLS %s path is invalid", label)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Nacos TLS %s is unreadable", label)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("Nacos TLS %s is not a regular file", label)
	}
	data, err := io.ReadAll(io.LimitReader(file, nacosTLSMaterialLimit+1))
	if err != nil || len(data) == 0 {
		return nil, fmt.Errorf("Nacos TLS %s is unreadable", label)
	}
	if len(data) > nacosTLSMaterialLimit {
		clear(data)
		return nil, fmt.Errorf("Nacos TLS %s exceeds the byte limit", label)
	}
	return data, nil
}

func newNacosHTTPClient(cfg config.Config) (*http.Client, error) {
	parsed, err := url.Parse(cfg.NacosAPIOrigin)
	if err != nil {
		return nil, errors.New("Nacos HTTP API origin is invalid")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DisableKeepAlives = true
	transport.TLSClientConfig = nil
	switch parsed.Scheme {
	case "http":
		if cfg.NacosHTTPTLSCAFile != "" || cfg.NacosHTTPTLSServerName != "" ||
			cfg.NacosHTTPTLSClientCertFile != "" || cfg.NacosHTTPTLSClientKeyFile != "" {
			return nil, errors.New("Nacos HTTP plaintext mode cannot include TLS material")
		}
	case "https":
		tlsConfig, tlsErr := newNacosTLSConfig(
			cfg.NacosHTTPTLSCAFile,
			cfg.NacosHTTPTLSServerName,
			cfg.NacosHTTPTLSClientCertFile,
			cfg.NacosHTTPTLSClientKeyFile,
		)
		if tlsErr != nil {
			return nil, tlsErr
		}
		transport.TLSClientConfig = tlsConfig
	default:
		return nil, errors.New("Nacos HTTP API origin scheme is unsupported")
	}
	return &http.Client{
		Transport: transport,
		Timeout:   cfg.NacosRequestTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("Nacos redirects are disabled")
		},
	}, nil
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
	if topologyReader, ok := selector.(api.TopologyStatusReader); ok {
		topologyHandler, err := api.NewTopologyStatusHandler(topologyReader)
		if err != nil {
			return nil, err
		}
		if err := topologyHandler.RegisterRoutes(mux, authenticator); err != nil {
			return nil, err
		}
	}
	return mux, nil
}
