// Command trimblectl is the operator CLI for Trimble MCP Bridge. Data
// commands go through the same gateway as MCP clients, so authorization,
// validation, and audit are identical. Defaults: read-only, staging, JSON.
package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/app"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/audit"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/catalog"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/config"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/fsperm"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/gateway"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/identity"
)

const usage = `trimblectl — operator CLI for Trimble MCP Bridge

Usage:
  trimblectl capabilities
  trimblectl projects list   --product P [--page-size N] [--page-token T]
  trimblectl projects get    --product P --project ID
  trimblectl files list      --product P --project ID --folder ID [--page-size N] [--page-token T]
  trimblectl files metadata  --product P --project ID --file ID
  trimblectl desktop link    --product trimble-connect-desktop --project ID [--view V] [--panel P]
  trimblectl desktop open    --product trimble-connect-desktop --project ID [--view V] [--panel P] --reason R [--launch]
                             (Trimble Connect for Windows command line; dry run unless --launch)
  trimblectl api operations  [--api A] [--family F] [--disposition read|plan|reference|variant|excluded] [--query Q] [--key K] [--page-size N] [--page-token T]
  trimblectl api read        --product trimble-connect --key K [--path name=value]... [--query name=value]... [--header name=value]...
  trimblectl api plan        [--product trimble-connect] --key K [--path ...] [--query ...] [--body-file F] --reason R   (dry run; never sent; no product for reference keys)
  trimblectl auth login      (Trimble Identity authorization code + PKCE, loopback redirect)
  trimblectl auth logout     (revokes the refresh token and deletes the local store)
  trimblectl audit verify    --file PATH
  trimblectl token new       (prints a new random bearer token once, and its SHA-256)
  trimblectl diagnostics

Exit codes: 0 success, 1 operation error, 2 usage error.
Configuration is read from environment variables; see docs/operations/configuration.md.
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	cmd := args[0]
	sub := ""
	rest := args[1:]
	if len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
		sub, rest = rest[0], rest[1:]
	}
	switch cmd + " " + sub {
	case "capabilities ":
		return tool(stdout, stderr, gateway.ToolGetCapabilities, rest, nil)
	case "projects list":
		return tool(stdout, stderr, gateway.ToolListProjects, rest, []string{"product", "page-size", "page-token"})
	case "projects get":
		return tool(stdout, stderr, gateway.ToolGetProject, rest, []string{"product", "project"})
	case "files list":
		return tool(stdout, stderr, gateway.ToolListFolderItems, rest, []string{"product", "project", "folder", "page-size", "page-token"})
	case "files metadata":
		return tool(stdout, stderr, gateway.ToolGetFileMetadata, rest, []string{"product", "project", "file"})
	case "desktop link":
		return tool(stdout, stderr, gateway.ToolBuildDesktop, rest, []string{"product", "project", "view", "panel"})
	case "desktop open":
		return tool(stdout, stderr, gateway.ToolOpenDesktop, rest, []string{"product", "project", "view", "panel", "reason", "launch"})
	case "api operations":
		return tool(stdout, stderr, gateway.ToolAPIOperations, rest, []string{"api", "family", "disposition", "query", "key", "page-size", "page-token"})
	case "api read":
		return apiCall(stdout, stderr, gateway.ToolAPIRead, rest)
	case "api plan":
		return apiCall(stdout, stderr, gateway.ToolAPIPlan, rest)
	case "auth login":
		return fail(stderr, authLogin())
	case "auth logout":
		return fail(stderr, authLogout())
	case "audit verify":
		return auditVerify(stdout, stderr, rest)
	case "token new":
		return tokenNew(stdout, stderr)
	case "diagnostics ":
		return diagnostics(stdout, stderr)
	case "help ", "-h ", "--help ":
		fmt.Fprint(stdout, usage)
		return 0
	}
	fmt.Fprint(stderr, usage)
	return 2
}

func fail(stderr io.Writer, err error) int {
	if err != nil {
		fmt.Fprintln(stderr, "trimblectl:", err)
		return 1
	}
	return 0
}

var flagToArg = map[string]string{
	"product": "product", "project": "project_id", "folder": "folder_id", "file": "file_id",
	"page-size": "page_size", "page-token": "page_token",
	"view": "view", "panel": "panel", "reason": "reason",
	"api": "api", "family": "family", "disposition": "disposition", "query": "query", "key": "key",
}

func tool(stdout, stderr io.Writer, name string, args []string, allowed []string) int {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	vals := map[string]*string{}
	var launch *bool
	for _, f := range allowed {
		if f == "launch" {
			launch = fs.Bool("launch", false, "actually open the application (default: dry run)")
			continue
		}
		vals[f] = fs.String(f, "", "")
	}
	if err := fs.Parse(args); err != nil || fs.NArg() > 0 {
		return 2
	}
	in := map[string]any{}
	if launch != nil {
		in["dry_run"] = !*launch
	}
	for f, v := range vals {
		if *v == "" {
			continue
		}
		if f == "page-size" {
			var n int
			if _, err := fmt.Sscanf(*v, "%d", &n); err != nil {
				fmt.Fprintln(stderr, "trimblectl: --page-size must be an integer")
				return 2
			}
			in[flagToArg[f]] = n
			continue
		}
		in[flagToArg[f]] = *v
	}
	return callTool(stdout, stderr, name, in)
}

// callTool runs one gateway tool as the local operator and prints the
// structured result. Exit 1 when the tool reports an error.
func callTool(stdout, stderr io.Writer, name string, in map[string]any) int {
	cfg, err := config.Load()
	if err != nil {
		return fail(stderr, err)
	}
	a, err := app.New(cfg)
	if err != nil {
		return fail(stderr, err)
	}
	defer a.Close()
	raw, _ := json.Marshal(in)
	res, rpcErr := a.Gateway.CallTool(context.Background(), a.LocalPrincipal(), name, raw)
	if rpcErr != nil {
		fmt.Fprintln(stderr, "trimblectl:", rpcErr.Message)
		return 1
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(res.StructuredContent)
	if res.IsError {
		return 1
	}
	return 0
}

// pairs collects repeatable name=value flags.
type pairs map[string]string

func (p pairs) String() string { return "" }
func (p pairs) Set(v string) error {
	k, val, ok := strings.Cut(v, "=")
	if !ok || k == "" {
		return errors.New("expected name=value")
	}
	p[k] = val
	return nil
}

// apiCall implements `trimblectl api read|plan`.
func apiCall(stdout, stderr io.Writer, name string, args []string) int {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	product := fs.String("product", "", "configured product (default trimble-connect; omitted for reference keys)")
	key := fs.String("key", "", "operation key from `trimblectl api operations`")
	reason := fs.String("reason", "", "why (plan only)")
	bodyFile := fs.String("body-file", "", "JSON request body file (plan only)")
	path, query, header := pairs{}, pairs{}, pairs{}
	fs.Var(path, "path", "path parameter name=value (repeatable)")
	fs.Var(query, "query", "query parameter name=value (repeatable)")
	fs.Var(header, "header", "header parameter name=value (repeatable)")
	if err := fs.Parse(args); err != nil || fs.NArg() > 0 || *key == "" {
		return 2
	}
	in := map[string]any{"key": *key}
	switch op, ok := catalog.Lookup(*key); {
	case *product != "":
		in["product"] = *product
	case ok && op.Disposition == catalog.Reference:
		// Reference operations name their product in the key.
	default:
		in["product"] = "trimble-connect"
	}
	if len(path) > 0 {
		in["path_params"] = map[string]string(path)
	}
	if len(query) > 0 {
		q := map[string]any{}
		for k, v := range query {
			q[k] = v
		}
		in["query_params"] = q
	}
	if len(header) > 0 {
		in["header_params"] = map[string]string(header)
	}
	if name == gateway.ToolAPIPlan {
		in["reason"] = *reason
		if *bodyFile != "" {
			b, err := os.ReadFile(*bodyFile)
			if err != nil {
				return fail(stderr, err)
			}
			if !json.Valid(b) {
				return fail(stderr, errors.New("body file is not valid JSON"))
			}
			in["body"] = json.RawMessage(b)
		}
	} else if *reason != "" || *bodyFile != "" {
		fmt.Fprintln(stderr, "trimblectl: --reason and --body-file apply to `api plan` only")
		return 2
	}
	return callTool(stdout, stderr, name, in)
}

func auditVerify(stdout, stderr io.Writer, args []string) int {
	fs := flag.NewFlagSet("audit verify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	path := fs.String("file", "", "audit log path")
	if err := fs.Parse(args); err != nil || *path == "" {
		return 2
	}
	f, err := os.Open(*path)
	if err != nil {
		return fail(stderr, err)
	}
	defer f.Close()
	n, err := audit.Verify(f)
	if err != nil {
		return fail(stderr, err)
	}
	fmt.Fprintf(stdout, "{\"records\": %d, \"chain\": \"intact\"}\n", n)
	return 0
}

func tokenNew(stdout, stderr io.Writer) int {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return fail(stderr, err)
	}
	tok := "tmb_" + hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(tok))
	fmt.Fprintf(stderr, "Store the token in the client's secret store now; it is not shown again.\n")
	fmt.Fprintf(stdout, "{\"token\": %q, \"sha256\": %q}\n", tok, hex.EncodeToString(sum[:]))
	return 0
}

func diagnostics(stdout, stderr io.Writer) int {
	cfg, err := config.Load()
	if err != nil {
		return fail(stderr, err)
	}
	type check struct {
		Name   string `json:"name"`
		Status string `json:"status"`
		Detail string `json:"detail,omitempty"`
	}
	var checks []check
	add := func(n, s, d string) { checks = append(checks, check{n, s, d}) }
	add("tenant", "ok", string(cfg.Tenant))
	add("mock_adapter", map[bool]string{true: "enabled", false: "disabled"}[cfg.EnableMock], "")
	if cfg.Connect.Enabled {
		add("trimble_connect", "enabled", cfg.Connect.Environment+"/"+cfg.Connect.Region)
		for _, p := range []struct{ name, path string }{
			{"token_store", cfg.Connect.TokenStore}, {"token_key_file", cfg.Connect.TokenKeyFile},
			{"access_token_file", cfg.Connect.AccessTokenFile}, {"client_secret_file", cfg.Connect.ClientSecretFile},
		} {
			if p.path == "" {
				continue
			}
			if fi, err := os.Stat(p.path); err != nil {
				add(p.name, "missing", "")
			} else if !fsperm.Private(runtime.GOOS, fi.Mode()) {
				add(p.name, "insecure_permissions", "chmod 600")
			} else {
				add(p.name, "ok", "")
			}
		}
	} else {
		add("trimble_connect", "disabled", "")
	}
	scopes := make([]string, 0, len(cfg.LocalScopes))
	for _, s := range cfg.LocalScopes {
		scopes = append(scopes, string(s))
	}
	add("local_scopes", "ok", strings.Join(scopes, " "))
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]any{"checks": checks})
	return 0
}

// authLogin runs the authorization code + PKCE flow with a loopback redirect.
// The redirect URI must be registered for the application with Trimble.
func authLogin() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	c := cfg.Connect
	if c.TokenStore == "" || c.TokenKeyFile == "" {
		return errors.New("set TRIMBLE_TOKEN_STORE and TRIMBLE_TOKEN_KEY_FILE first")
	}
	cl, err := app.IdentityClient(c)
	if err != nil {
		return err
	}
	ru, err := url.Parse(c.RedirectURI)
	if err != nil || ru.Scheme != "http" || (ru.Hostname() != "127.0.0.1" && ru.Hostname() != "localhost") {
		return errors.New("auth login requires a loopback http redirect URI (http://127.0.0.1:PORT/PATH)")
	}
	pkce, err := identity.NewPKCE()
	if err != nil {
		return err
	}
	state, err := identity.NewState()
	if err != nil {
		return err
	}
	ln, err := net.Listen("tcp", ru.Host)
	if err != nil {
		return err
	}
	type result struct {
		code string
		err  error
	}
	done := make(chan result, 1)
	srv := &http.Server{ReadHeaderTimeout: 10 * time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != ru.Path {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		if q.Get("state") != state {
			// Ignore stray or forged callbacks instead of aborting the
			// login; only the matching state completes it.
			http.Error(w, "state mismatch", http.StatusBadRequest)
			return
		}
		var res result
		switch {
		case q.Get("error") != "":
			res = result{err: fmt.Errorf("authorization denied: %.64s", q.Get("error"))}
		case q.Get("code") == "":
			res = result{err: errors.New("no authorization code returned")}
		default:
			res = result{code: q.Get("code")}
		}
		select {
		case done <- res: // first result wins; repeats never block
		default:
		}
		fmt.Fprintln(w, "Trimble MCP Bridge: you can close this window.")
	})}
	go func() { _ = srv.Serve(ln) }()
	defer srv.Close()

	fmt.Fprintln(os.Stderr, "Open this URL in a browser to sign in with Trimble Identity:")
	fmt.Fprintln(os.Stderr, cl.AuthorizeURL(state, pkce))
	var res result
	select {
	case res = <-done:
	case <-time.After(10 * time.Minute): // authorization codes are valid for 10 minutes
		return errors.New("timed out waiting for the authorization callback")
	}
	if res.err != nil {
		return res.err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tok, err := cl.Exchange(ctx, res.code, pkce)
	if err != nil {
		return err
	}
	if err := (&identity.FileStore{Path: c.TokenStore, KeyPath: c.TokenKeyFile}).Save(tok); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "Signed in. Tokens are stored encrypted; refresh at least every 9 days (Trimble Connect guidance).")
	return nil
}

func authLogout() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	c := cfg.Connect
	store := &identity.FileStore{Path: c.TokenStore, KeyPath: c.TokenKeyFile}
	tok, err := store.Load()
	if err != nil {
		return err
	}
	cl, err := app.IdentityClient(c)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	revokeErr := cl.Revoke(ctx, tok.RefreshToken)
	if err := os.Remove(c.TokenStore); err != nil {
		return err
	}
	if revokeErr != nil {
		return fmt.Errorf("local session deleted, but upstream revocation failed: %w", revokeErr)
	}
	fmt.Fprintln(os.Stderr, "Signed out: refresh token revoked and local store deleted.")
	return nil
}
