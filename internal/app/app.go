// Package app wires configuration, adapters, audit, and the gateway. It is
// shared by the MCP server and the CLI so both enforce identical policy.
package app

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/audit"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/authz"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/config"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/domain"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/gateway"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/identity"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/mcp"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/trimble/connect"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/trimble/desktop"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/trimble/mock"
)

// App is a wired bridge.
type App struct {
	Config  *config.Config
	Gateway *gateway.Gateway
	Server  *mcp.Server
	Logger  *slog.Logger
	closers []io.Closer
}

// Close releases resources such as the audit file.
func (a *App) Close() error {
	var errsOut []error
	for _, c := range a.closers {
		errsOut = append(errsOut, c.Close())
	}
	return errors.Join(errsOut...)
}

// LocalPrincipal is the operator principal used by stdio and the CLI.
func (a *App) LocalPrincipal() *authz.Principal {
	return &authz.Principal{
		Subject: a.Config.LocalSubject, Client: "local", Tenant: a.Config.Tenant,
		Scopes: a.Config.LocalScopes, Projects: a.Config.LocalProjects, Local: true,
	}
}

// New builds an App. Logs go to stderr so stdout stays reserved for MCP.
func New(cfg *config.Config) (*App, error) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	a := &App{Config: cfg, Logger: logger}

	sink, f, err := audit.OpenFile(cfg.AuditPath)
	if err != nil {
		return nil, err
	}
	a.closers = append(a.closers, f)

	reg := gateway.NewRegistry()
	if cfg.EnableMock {
		reg.Register(cfg.Tenant, mock.New())
	}
	if cfg.Connect.Enabled {
		ts, err := connectTokens(cfg.Connect)
		if err != nil {
			a.Close()
			return nil, err
		}
		region, err := domain.ParseRegion(cfg.Connect.Region)
		if err != nil {
			a.Close()
			return nil, err
		}
		ad, err := connect.New(connect.Config{
			Environment: connect.Environment(cfg.Connect.Environment), Region: region, Tokens: ts,
		})
		if err != nil {
			a.Close()
			return nil, err
		}
		reg.Register(cfg.Tenant, ad)
	}
	if cfg.DesktopEnabled {
		reg.Register(cfg.Tenant, desktop.New(desktop.Config{LaunchEnabled: cfg.DesktopLaunch}))
	}
	gw, err := gateway.New(gateway.Options{Registry: reg, Audit: sink, Logger: logger})
	if err != nil {
		a.Close()
		return nil, err
	}
	a.Gateway = gw
	a.Server = &mcp.Server{
		Info:     mcp.Implementation{Name: "trimble-mcp-bridge", Title: "Trimble MCP Bridge", Version: gateway.Version},
		Provider: gw,
	}
	return a, nil
}

func connectTokens(c config.ConnectConfig) (connect.TokenSource, error) {
	if c.AccessTokenFile != "" {
		return identity.FileTokenSource{Path: c.AccessTokenFile}, nil
	}
	cl, err := IdentityClient(c)
	if err != nil {
		return nil, err
	}
	return &identity.RefreshingSource{Client: cl, Store: &identity.FileStore{Path: c.TokenStore, KeyPath: c.TokenKeyFile},
		Logf: func(format string, args ...any) { fmt.Fprintf(os.Stderr, format+"\n", args...) }}, nil
}

// IdentityClient builds the Trimble Identity client from configuration.
func IdentityClient(c config.ConnectConfig) (*identity.Client, error) {
	var secret identity.Secret
	if c.ClientSecretFile != "" {
		b, err := os.ReadFile(c.ClientSecretFile)
		if err != nil {
			return nil, err
		}
		secret = identity.Secret(strings.TrimSpace(string(b)))
	}
	return identity.NewClient(c.Issuer, c.ClientID, secret, c.RedirectURI, c.Scope)
}
