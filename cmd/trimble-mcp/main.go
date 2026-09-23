// Command trimble-mcp runs the Trimble MCP Bridge over stdio (local agents)
// or Streamable HTTP (remote connectors).
package main

import (
	"context"
	cryptotls "crypto/tls"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/app"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/authz"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/config"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/domain"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/mcp"
)

func main() {
	transport := flag.String("transport", "stdio", "stdio or http")
	flag.Parse()
	if err := run(*transport); err != nil {
		fmt.Fprintln(os.Stderr, "trimble-mcp:", err)
		os.Exit(1)
	}
}

func run(transport string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	a, err := app.New(cfg)
	if err != nil {
		return err
	}
	defer a.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch transport {
	case "stdio":
		sess := &mcp.Session{ID: "stdio", Principal: a.LocalPrincipal()}
		return a.Server.ServeStdio(ctx, sess, os.Stdin, os.Stdout)
	case "http":
		return serveHTTP(ctx, a)
	}
	return fmt.Errorf("unknown transport %q", transport)
}

func serveHTTP(ctx context.Context, a *app.App) error {
	cfg := a.Config
	if cfg.HTTPTokensFile == "" {
		return errors.New("http transport requires TRIMBLE_MCP_HTTP_TOKENS_FILE")
	}
	tls := cfg.TLSCertFile != "" && cfg.TLSKeyFile != ""
	if !tls && !cfg.AllowPlainHTTP {
		return errors.New("http transport requires TLS (TRIMBLE_MCP_TLS_CERT_FILE/KEY_FILE) or TRIMBLE_MCP_ALLOW_PLAIN_HTTP=true behind a TLS-terminating proxy")
	}
	entries, err := config.LoadTokenEntries(cfg.HTTPTokensFile)
	if err != nil {
		return err
	}
	auth := &mcp.StaticTokenAuthenticator{}
	for _, e := range entries {
		scopes, err := authz.ValidateScopes(e.Scopes)
		if err != nil {
			return err
		}
		for _, s := range scopes {
			if s == authz.ScopeFilesWrite || s == authz.ScopeFilesDelete {
				return errors.New("write and delete scopes are not available in this release")
			}
		}
		p := authz.Principal{Subject: e.Subject, Client: e.Client, Tenant: cfg.Tenant, Scopes: scopes}
		for _, id := range e.Projects {
			pid, err := domain.ParseProjectID(id)
			if err != nil {
				return err
			}
			p.Projects = append(p.Projects, pid)
		}
		for _, pr := range e.Products {
			prod, err := domain.ParseProduct(pr)
			if err != nil {
				return err
			}
			p.Products = append(p.Products, prod)
		}
		if err := auth.AddTokenHash(e.SHA256, p); err != nil {
			return err
		}
	}
	h := mcp.NewHTTPHandler(a.Server, mcp.HTTPOptions{
		Auth: auth, AllowedOrigins: cfg.AllowedOrigins, AllowedHosts: cfg.AllowedHosts,
		ResourceMetadataURL: cfg.ResourceMetadataURL,
	})
	mux := http.NewServeMux()
	mux.Handle("/mcp", h)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	srv := &http.Server{
		Addr: cfg.HTTPAddr, Handler: mux, TLSConfig: &cryptotls.Config{MinVersion: cryptotls.VersionTLS12},
		ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second,
		WriteTimeout: 3 * time.Minute, IdleTimeout: 2 * time.Minute, MaxHeaderBytes: 32 << 10,
	}
	errc := make(chan error, 1)
	go func() {
		a.Logger.Info("listening", "addr", cfg.HTTPAddr, "tls", tls)
		if tls {
			errc <- srv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
		} else {
			errc <- srv.ListenAndServe()
		}
	}()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return srv.Shutdown(shutdown)
}
