// Package config loads bridge configuration from environment variables and
// an optional JSON file. Secrets are never read from either directly: only
// paths to private files holding them.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/authz"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/domain"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/errs"
)

// Config is the resolved configuration.
type Config struct {
	Tenant    domain.TenantID
	AuditPath string

	// Local (stdio) principal.
	LocalSubject  string
	LocalScopes   []authz.Scope
	LocalProjects []domain.ProjectID

	EnableMock bool

	// Trimble Connect for Windows command line (trimbleconnect: URL scheme).
	DesktopEnabled bool
	DesktopLaunch  bool

	Connect ConnectConfig

	// Remote HTTP.
	HTTPAddr            string
	HTTPTokensFile      string
	AllowedOrigins      []string
	AllowedHosts        []string
	ResourceMetadataURL string
	TLSCertFile         string
	TLSKeyFile          string
	AllowPlainHTTP      bool
}

// ConnectConfig configures the Trimble Connect adapter.
type ConnectConfig struct {
	Enabled          bool
	Environment      string // stage | production
	Region           string
	Issuer           string
	ClientID         string
	ClientSecretFile string
	RedirectURI      string
	Scope            string
	TokenStore       string
	TokenKeyFile     string
	AccessTokenFile  string
}

func env(k, def string) string {
	if v, ok := os.LookupEnv(k); ok {
		return strings.TrimSpace(v)
	}
	return def
}

func envBool(k string, def bool) (bool, error) {
	v := env(k, "")
	if v == "" {
		return def, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, errs.Newf(errs.Configuration, "%s must be true or false", k)
	}
	return b, nil
}

func list(v string) []string {
	var out []string
	for _, s := range strings.Split(v, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// Load reads configuration from the environment. Defaults are the safe ones:
// mock enabled, Connect disabled, staging environment, read-only scopes.
func Load() (*Config, error) {
	c := &Config{
		AuditPath:           env("TRIMBLE_MCP_AUDIT_LOG", "trimble-mcp-audit.jsonl"),
		LocalSubject:        env("TRIMBLE_MCP_LOCAL_SUBJECT", "local-operator"),
		HTTPAddr:            env("TRIMBLE_MCP_HTTP_ADDR", "127.0.0.1:8787"),
		HTTPTokensFile:      env("TRIMBLE_MCP_HTTP_TOKENS_FILE", ""),
		AllowedOrigins:      list(env("TRIMBLE_MCP_ALLOWED_ORIGINS", "")),
		AllowedHosts:        list(env("TRIMBLE_MCP_ALLOWED_HOSTS", "")),
		ResourceMetadataURL: env("TRIMBLE_MCP_RESOURCE_METADATA_URL", ""),
		TLSCertFile:         env("TRIMBLE_MCP_TLS_CERT_FILE", ""),
		TLSKeyFile:          env("TRIMBLE_MCP_TLS_KEY_FILE", ""),
		Connect: ConnectConfig{
			Environment:      env("TRIMBLE_CONNECT_ENV", "stage"),
			Region:           env("TRIMBLE_CONNECT_REGION", "us"),
			Issuer:           env("TRIMBLE_IDENTITY_ISSUER", "https://id.trimble.com"),
			ClientID:         env("TRIMBLE_CLIENT_ID", ""),
			ClientSecretFile: env("TRIMBLE_CLIENT_SECRET_FILE", ""),
			RedirectURI:      env("TRIMBLE_REDIRECT_URI", "http://127.0.0.1:8765/callback"),
			Scope:            env("TRIMBLE_SCOPE", ""),
			TokenStore:       env("TRIMBLE_TOKEN_STORE", ""),
			TokenKeyFile:     env("TRIMBLE_TOKEN_KEY_FILE", ""),
			AccessTokenFile:  env("TRIMBLE_ACCESS_TOKEN_FILE", ""),
		},
	}
	tenant, err := domain.ParseTenantID(env("TRIMBLE_MCP_TENANT", "local"))
	if err != nil {
		return nil, err
	}
	c.Tenant = tenant
	if c.EnableMock, err = envBool("TRIMBLE_MCP_ENABLE_MOCK", true); err != nil {
		return nil, err
	}
	if c.Connect.Enabled, err = envBool("TRIMBLE_CONNECT_ENABLED", false); err != nil {
		return nil, err
	}
	if c.DesktopEnabled, err = envBool("TRIMBLE_CONNECT_DESKTOP_ENABLED", false); err != nil {
		return nil, err
	}
	if c.DesktopLaunch, err = envBool("TRIMBLE_CONNECT_DESKTOP_LAUNCH", false); err != nil {
		return nil, err
	}
	if c.DesktopLaunch && !c.DesktopEnabled {
		return nil, errs.Newf(errs.Configuration, "TRIMBLE_CONNECT_DESKTOP_LAUNCH requires TRIMBLE_CONNECT_DESKTOP_ENABLED=true")
	}
	if c.AllowPlainHTTP, err = envBool("TRIMBLE_MCP_ALLOW_PLAIN_HTTP", false); err != nil {
		return nil, err
	}
	scopes := list(env("TRIMBLE_MCP_LOCAL_SCOPES", ""))
	if len(scopes) == 0 {
		c.LocalScopes = append([]authz.Scope{}, authz.ReadOnlyDefault...)
		c.LocalScopes = append(c.LocalScopes, authz.ScopeFilesRead)
	} else if c.LocalScopes, err = authz.ValidateScopes(scopes); err != nil {
		return nil, err
	}
	for _, s := range c.LocalScopes {
		if s == authz.ScopeFilesWrite || s == authz.ScopeFilesDelete {
			return nil, errs.Newf(errs.Configuration, "write and delete scopes are not available in this release")
		}
	}
	for _, p := range list(env("TRIMBLE_MCP_LOCAL_PROJECTS", "")) {
		id, err := domain.ParseProjectID(p)
		if err != nil {
			return nil, err
		}
		c.LocalProjects = append(c.LocalProjects, id)
	}
	if c.Connect.Enabled {
		if c.Connect.Environment != "stage" && c.Connect.Environment != "production" {
			return nil, errs.Newf(errs.Configuration, "TRIMBLE_CONNECT_ENV must be stage or production")
		}
		if c.Connect.AccessTokenFile == "" && (c.Connect.TokenStore == "" || c.Connect.TokenKeyFile == "") {
			return nil, errs.Newf(errs.Configuration, "Trimble Connect needs TRIMBLE_TOKEN_STORE and TRIMBLE_TOKEN_KEY_FILE (or TRIMBLE_ACCESS_TOKEN_FILE for sandbox use)")
		}
	}
	return c, nil
}

// TokenEntry is one line of the HTTP tokens file. Only the SHA-256 hash of
// each bearer token is stored.
type TokenEntry struct {
	SHA256   string   `json:"sha256"`
	Subject  string   `json:"subject"`
	Client   string   `json:"client"`
	Scopes   []string `json:"scopes"`
	Projects []string `json:"projects,omitempty"`
	Products []string `json:"products,omitempty"`
}

// LoadTokenEntries reads the HTTP tokens file (a JSON array).
func LoadTokenEntries(path string) ([]TokenEntry, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("tokens file must not be readable by group or others (chmod 600)")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []TokenEntry
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("tokens file is not a JSON array of entries: %w", err)
	}
	return out, nil
}
