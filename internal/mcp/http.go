package mcp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/audit"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/authz"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/ratelimit"
)

// Authenticator resolves the caller of an HTTP request. It returns an error
// when credentials are missing or invalid; it must never log the credential.
type Authenticator interface {
	Authenticate(r *http.Request) (*authz.Principal, error)
}

// ErrUnauthenticated is returned by Authenticators for missing or bad tokens.
var ErrUnauthenticated = errors.New("unauthenticated")

// StaticTokenAuthenticator maps SHA-256 hashes of opaque bearer tokens to
// principals. Only hashes are held in memory or configuration. It is intended
// for local development and a controlled pilot; production deployments use an
// OAuth 2.1 authorization server (see docs/security/remote-auth.md).
type StaticTokenAuthenticator struct {
	entries []tokenEntry
}

type tokenEntry struct {
	hash      [32]byte
	principal authz.Principal
}

// AddTokenHash registers a hex-encoded SHA-256 token hash for principal.
func (a *StaticTokenAuthenticator) AddTokenHash(hexHash string, p authz.Principal) error {
	b, err := hex.DecodeString(hexHash)
	if err != nil || len(b) != 32 {
		return errors.New("token hash must be 64 hex characters (SHA-256)")
	}
	var h [32]byte
	copy(h[:], b)
	a.entries = append(a.entries, tokenEntry{hash: h, principal: p})
	return nil
}

// Authenticate implements Authenticator.
func (a *StaticTokenAuthenticator) Authenticate(r *http.Request) (*authz.Principal, error) {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(h) <= len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return nil, ErrUnauthenticated
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(h[len(prefix):])))
	var match *authz.Principal
	for i := range a.entries {
		if subtle.ConstantTimeCompare(sum[:], a.entries[i].hash[:]) == 1 {
			p := a.entries[i].principal
			match = &p
		}
	}
	if match == nil {
		return nil, ErrUnauthenticated
	}
	return match, nil
}

// HTTPOptions configures the Streamable HTTP transport.
type HTTPOptions struct {
	Auth Authenticator
	// AllowedOrigins lists exact Origin values accepted. Requests carrying any
	// other Origin are rejected (DNS-rebinding and cross-site protection).
	AllowedOrigins []string
	// AllowedHosts, when non-empty, restricts the Host header.
	AllowedHosts []string
	// ResourceMetadataURL, when set, is advertised in WWW-Authenticate per
	// RFC 9728 so OAuth-capable clients can discover the authorization server.
	ResourceMetadataURL string
	SessionIdleTTL      time.Duration
	MaxSessions         int
	// MaxSessionsPerSubject stops one caller from exhausting MaxSessions.
	MaxSessionsPerSubject int
	// RatePerSecond and Burst bound requests per authenticated subject.
	RatePerSecond float64
	Burst         int
	Logf          func(format string, args ...any)
}

type httpSession struct {
	sess     *Session
	subject  string
	tenant   string
	lastSeen time.Time
}

// HTTPHandler implements the MCP Streamable HTTP transport using JSON
// responses only. It does not offer a server-initiated SSE stream; GET
// returns 405 as the specification permits.
type HTTPHandler struct {
	srv     *Server
	opts    HTTPOptions
	limiter *ratelimit.Keyed

	mu       sync.Mutex
	sessions map[string]*httpSession
	now      func() time.Time
}

// NewHTTPHandler returns a handler for the MCP endpoint.
func NewHTTPHandler(srv *Server, opts HTTPOptions) *HTTPHandler {
	if opts.SessionIdleTTL == 0 {
		opts.SessionIdleTTL = 30 * time.Minute
	}
	if opts.MaxSessions == 0 {
		opts.MaxSessions = 1000
	}
	if opts.MaxSessionsPerSubject == 0 {
		opts.MaxSessionsPerSubject = 20
	}
	if opts.RatePerSecond == 0 {
		opts.RatePerSecond = 5
	}
	if opts.Burst == 0 {
		opts.Burst = 20
	}
	if opts.Logf == nil {
		opts.Logf = func(string, ...any) {}
	}
	return &HTTPHandler{
		srv: srv, opts: opts,
		limiter:  ratelimit.NewKeyed(opts.RatePerSecond, opts.Burst),
		sessions: map[string]*httpSession{},
		now:      time.Now,
	}
}

func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if !h.hostAllowed(r.Host) {
		http.Error(w, "host not allowed", http.StatusForbidden)
		return
	}
	if o := r.Header.Get("Origin"); o != "" && !slices.Contains(h.opts.AllowedOrigins, o) {
		http.Error(w, "origin not allowed", http.StatusForbidden)
		return
	}
	switch r.Method {
	case http.MethodPost, http.MethodDelete:
	default:
		w.Header().Set("Allow", "POST, DELETE")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.opts.Auth == nil {
		http.Error(w, "server has no authenticator configured", http.StatusServiceUnavailable)
		return
	}
	p, err := h.opts.Auth.Authenticate(r)
	if err != nil {
		h.challenge(w, r)
		return
	}
	if ok, wait := h.limiter.Allow(string(p.Tenant) + "\x00" + p.Subject); !ok {
		w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
		http.Error(w, "rate limited", http.StatusTooManyRequests)
		return
	}
	if r.Method == http.MethodDelete {
		h.deleteSession(w, r, p)
		return
	}
	h.post(w, r, p)
}

func (h *HTTPHandler) hostAllowed(host string) bool {
	if len(h.opts.AllowedHosts) == 0 {
		return true
	}
	if hh, _, err := net.SplitHostPort(host); err == nil {
		host = hh
	}
	return slices.Contains(h.opts.AllowedHosts, host)
}

func (h *HTTPHandler) challenge(w http.ResponseWriter, r *http.Request) {
	v := `Bearer realm="trimble-mcp"`
	if r.Header.Get("Authorization") != "" {
		v += `, error="invalid_token"`
	}
	if h.opts.ResourceMetadataURL != "" {
		v += `, resource_metadata="` + h.opts.ResourceMetadataURL + `"`
	}
	w.Header().Set("WWW-Authenticate", v)
	http.Error(w, "unauthorized", http.StatusUnauthorized)
}

func (h *HTTPHandler) post(w http.ResponseWriter, r *http.Request, p *authz.Principal) {
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(strings.ToLower(ct), "application/json") {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}
	if acc := r.Header.Get("Accept"); acc != "" && !strings.Contains(acc, "application/json") && !strings.Contains(acc, "*/*") {
		http.Error(w, "Accept must include application/json", http.StatusNotAcceptable)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, MaxMessageBytes+1))
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}
	if len(body) > MaxMessageBytes {
		http.Error(w, "message too large", http.StatusRequestEntityTooLarge)
		return
	}
	body = bytes.TrimSpace(body)
	if len(body) > 0 && body[0] == '[' {
		writeJSON(w, http.StatusBadRequest, errorResponse(nil, CodeInvalidRequest, "batch requests are not supported"))
		return
	}
	mp, bad := DecodeMessage(body)
	if bad != nil {
		writeJSON(w, http.StatusBadRequest, bad)
		return
	}
	m := *mp
	// Checked after parsing so the error can carry the request ID.
	hv := r.Header.Get("MCP-Protocol-Version")
	if hv != "" && !slices.Contains(SupportedProtocolVersions, hv) {
		writeJSON(w, http.StatusBadRequest, unsupportedVersion(m.ID, hv))
		return
	}

	version, modern, metaErr := requestMeta(m.Params)
	if modern && m.Method != "initialize" {
		if metaErr != "" {
			writeJSON(w, http.StatusBadRequest, &Message{JSONRPC: "2.0", ID: m.ID, Error: &RPCError{Code: CodeInvalidParams, Message: metaErr}})
			return
		}
		h.postModern(w, r, p, &m, version)
		return
	}
	if slices.Contains(ModernProtocolVersions, hv) && m.Method != "initialize" {
		// A modern header without modern metadata: answer in JSON so a
		// dual-era client can recognise the modern error and adapt.
		writeJSON(w, http.StatusBadRequest, &Message{JSONRPC: "2.0", ID: m.ID, Error: &RPCError{
			Code: CodeInvalidParams, Message: "params._meta must carry " + MetaProtocolVersion + " and " + MetaClientCapabilities}})
		return
	}

	var hs *httpSession
	if m.Method == "initialize" {
		hs, err = h.newSession(p)
		if err != nil {
			http.Error(w, "too many sessions", http.StatusServiceUnavailable)
			return
		}
	} else {
		id := r.Header.Get("Mcp-Session-Id")
		if id == "" {
			http.Error(w, "Mcp-Session-Id header is required", http.StatusBadRequest)
			return
		}
		hs = h.lookup(id, p)
		if hs == nil {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
	}
	// Refresh the principal so revocations and scope changes take effect on
	// the next request rather than at session end.
	hs.sess.SetPrincipal(p)

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	resp := h.srv.Handle(ctx, hs.sess, &m)
	if m.Method == "initialize" {
		w.Header().Set("Mcp-Session-Id", hs.sess.ID)
	}
	if resp == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// postModern serves a stateless 2026-07-28 request. Routing headers must
// agree with the body so intermediaries that route on headers cannot be
// confused about what is being invoked.
func (h *HTTPHandler) postModern(w http.ResponseWriter, r *http.Request, p *authz.Principal, m *Message, version string) {
	if hv := r.Header.Get("MCP-Protocol-Version"); hv != version {
		writeJSON(w, http.StatusBadRequest, &Message{JSONRPC: "2.0", ID: m.ID,
			Error: &RPCError{Code: CodeHeaderMismatch, Message: "MCP-Protocol-Version header must match params._meta"}})
		return
	}
	if !m.IsResponse() && r.Header.Get("Mcp-Method") != m.Method {
		writeJSON(w, http.StatusBadRequest, &Message{JSONRPC: "2.0", ID: m.ID,
			Error: &RPCError{Code: CodeHeaderMismatch, Message: "Mcp-Method header must match the request method"}})
		return
	}
	if name, ok := routedName(m); ok {
		if decodeHeaderValue(r.Header.Get("Mcp-Name")) != name {
			writeJSON(w, http.StatusBadRequest, &Message{JSONRPC: "2.0", ID: m.ID,
				Error: &RPCError{Code: CodeHeaderMismatch, Message: "Mcp-Name header must match the request"}})
			return
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	resp := h.srv.Handle(ctx, &Session{Principal: p}, m)
	if resp == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	status := http.StatusOK
	if resp.Error != nil {
		switch resp.Error.Code {
		case CodeUnsupportedVersion:
			status = http.StatusBadRequest
		case CodeMethodNotFound:
			status = http.StatusNotFound
		}
	}
	writeJSON(w, status, resp)
}

// routedName returns the name or URI the modern revision requires in the
// Mcp-Name header for tools/call, resources/read, and prompts/get.
func routedName(m *Message) (string, bool) {
	var p struct {
		Name string `json:"name"`
		URI  string `json:"uri"`
	}
	switch m.Method {
	case "tools/call", "prompts/get":
		_ = json.Unmarshal(m.Params, &p)
		return p.Name, true
	case "resources/read":
		_ = json.Unmarshal(m.Params, &p)
		return p.URI, true
	}
	return "", false
}

// decodeHeaderValue decodes the =?base64?...?= form used for header values
// that are not plain ASCII.
func decodeHeaderValue(v string) string {
	const pre, suf = "=?base64?", "?="
	if strings.HasPrefix(v, pre) && strings.HasSuffix(v, suf) && len(v) >= len(pre)+len(suf) {
		b, err := base64.StdEncoding.DecodeString(v[len(pre) : len(v)-len(suf)])
		if err == nil {
			return string(b)
		}
	}
	return v
}

func (h *HTTPHandler) newSession(p *authz.Principal) (*httpSession, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.expireLocked()
	if len(h.sessions) >= h.opts.MaxSessions {
		return nil, errors.New("session limit")
	}
	mine := 0
	for _, s := range h.sessions {
		if s.subject == p.Subject && s.tenant == string(p.Tenant) {
			mine++
		}
	}
	if mine >= h.opts.MaxSessionsPerSubject {
		return nil, errors.New("per-subject session limit")
	}
	s := &Session{ID: audit.NewID("mcps"), Principal: p}
	hs := &httpSession{sess: s, subject: p.Subject, tenant: string(p.Tenant), lastSeen: h.now()}
	h.sessions[s.ID] = hs
	return hs, nil
}

// lookup returns the session only if it belongs to the same subject and
// tenant, so a leaked session ID is useless without the owner's credential.
func (h *HTTPHandler) lookup(id string, p *authz.Principal) *httpSession {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.expireLocked()
	hs, ok := h.sessions[id]
	if !ok || hs.subject != p.Subject || hs.tenant != string(p.Tenant) {
		return nil
	}
	hs.lastSeen = h.now()
	return hs
}

func (h *HTTPHandler) deleteSession(w http.ResponseWriter, r *http.Request, p *authz.Principal) {
	id := r.Header.Get("Mcp-Session-Id")
	if id == "" {
		// Without a session DELETE has no meaning (modern clients have none).
		w.Header().Set("Allow", "POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.lookup(id, p) == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	h.mu.Lock()
	delete(h.sessions, id)
	h.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPHandler) expireLocked() {
	cutoff := h.now().Add(-h.opts.SessionIdleTTL)
	for id, hs := range h.sessions {
		if hs.lastSeen.Before(cutoff) {
			delete(h.sessions, id)
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
