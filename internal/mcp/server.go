package mcp

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"sync"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/authz"
)

// Provider supplies tools, resources, and prompts. Every method receives the
// authenticated principal and must enforce authorization itself.
type Provider interface {
	Instructions() string
	ListTools(ctx context.Context, p *authz.Principal) []ToolInfo
	CallTool(ctx context.Context, p *authz.Principal, name string, args json.RawMessage) (*CallToolResult, *RPCError)
	ListResources(ctx context.Context, p *authz.Principal) []ResourceInfo
	ListResourceTemplates(ctx context.Context, p *authz.Principal) []ResourceTemplate
	ReadResource(ctx context.Context, p *authz.Principal, uri string) ([]ResourceContents, *RPCError)
	ListPrompts(ctx context.Context, p *authz.Principal) []PromptInfo
	GetPrompt(ctx context.Context, p *authz.Principal, name string, args map[string]string) (*GetPromptResult, *RPCError)
}

// Server dispatches MCP messages to a Provider.
type Server struct {
	Info     Implementation
	Provider Provider
}

// Session is per-connection state for legacy clients. Modern (stateless)
// requests use a Session only to carry the principal.
type Session struct {
	ID        string
	Principal *authz.Principal

	mu          sync.Mutex
	initialized bool
	version     string
	client      Implementation
}

// ProtocolVersion returns the negotiated legacy protocol revision, if any.
func (s *Session) ProtocolVersion() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.version
}

// SetPrincipal replaces the session's principal, e.g. after a transport
// re-authenticates a request, so revocations and scope changes apply.
func (s *Session) SetPrincipal(p *authz.Principal) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Principal = p
}

func (s *Session) principal() *authz.Principal {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Principal
}

// Initialized reports whether the legacy initialize handshake has completed.
func (s *Session) Initialized() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.initialized
}

// requestMeta inspects params._meta. modern is true when a protocol version
// key is present. bad is non-empty when the modern metadata is malformed:
// the revision requires a string protocolVersion and an object
// clientCapabilities on every request.
func requestMeta(params json.RawMessage) (version string, modern bool, bad string) {
	if len(params) == 0 {
		return "", false, ""
	}
	var p struct {
		Meta map[string]json.RawMessage `json:"_meta"`
	}
	if json.Unmarshal(params, &p) != nil || p.Meta == nil {
		return "", false, ""
	}
	raw, ok := p.Meta[MetaProtocolVersion]
	if !ok {
		return "", false, ""
	}
	if json.Unmarshal(raw, &version) != nil {
		return "", true, MetaProtocolVersion + " must be a string"
	}
	caps := strings.TrimSpace(string(p.Meta[MetaClientCapabilities]))
	if !strings.HasPrefix(caps, "{") {
		return version, true, MetaClientCapabilities + " is required and must be an object"
	}
	return version, true, ""
}

func unsupportedVersion(id json.RawMessage, requested string) *Message {
	return &Message{JSONRPC: "2.0", ID: id, Error: &RPCError{
		Code: CodeUnsupportedVersion, Message: "unsupported protocol version",
		Data: map[string]any{"supported": ModernProtocolVersions, "requested": requested},
	}}
}

// Handle processes one decoded message (see DecodeMessage) and returns the
// response, or nil for notifications and client responses.
func (srv *Server) Handle(ctx context.Context, sess *Session, m *Message) *Message {
	if m.IsResponse() || m.IsNotification() {
		// The server sends no requests, so client responses are ignored.
		// Notifications need no reply; transports handle cancellation.
		return nil
	}
	version, modern, bad := requestMeta(m.Params)
	if modern && m.Method != "initialize" {
		if bad != "" {
			return &Message{JSONRPC: "2.0", ID: m.ID, Error: &RPCError{Code: CodeInvalidParams, Message: bad}}
		}
		if !slices.Contains(ModernProtocolVersions, version) {
			return unsupportedVersion(m.ID, version)
		}
		result, rpcErr := srv.dispatch(ctx, sess, m, true)
		if rpcErr != nil {
			if rpcErr.Code == CodeResourceNotFoundLegacy {
				rpcErr.Code = CodeInvalidParams
			}
			return &Message{JSONRPC: "2.0", ID: m.ID, Error: rpcErr}
		}
		return &Message{JSONRPC: "2.0", ID: m.ID, Result: srv.decorateModern(m.Method, result)}
	}
	result, rpcErr := srv.dispatch(ctx, sess, m, false)
	if rpcErr != nil {
		return &Message{JSONRPC: "2.0", ID: m.ID, Error: rpcErr}
	}
	if m.Method == "server/discover" {
		// A modern-revision result even when probed without _meta.
		result = srv.decorateModern(m.Method, result)
	}
	return &Message{JSONRPC: "2.0", ID: m.ID, Result: result}
}

// needsInit reports whether a legacy request must wait for initialize.
func needsInit(method string) bool {
	return method != "initialize" && method != "ping" && method != "server/discover"
}

// decorateModern adds the fields the modern revision requires on results.
// Listings and reads are per-principal, so they are marked private and short
// lived.
func (srv *Server) decorateModern(method string, result any) any {
	b, err := json.Marshal(result)
	if err != nil {
		return result
	}
	var out map[string]any
	if json.Unmarshal(b, &out) != nil || out == nil {
		out = map[string]any{}
	}
	out["resultType"] = "complete"
	meta, _ := out["_meta"].(map[string]any)
	if meta == nil {
		meta = map[string]any{}
	}
	meta[MetaServerInfo] = srv.Info
	out["_meta"] = meta
	switch method {
	case "tools/list", "resources/list", "resources/templates/list", "prompts/list", "resources/read":
		if _, ok := out["ttlMs"]; !ok {
			out["ttlMs"] = 60000
			out["cacheScope"] = "private"
		}
	}
	return out
}

func (srv *Server) capabilities() map[string]any {
	return map[string]any{
		"tools":     map[string]any{"listChanged": false},
		"resources": map[string]any{"listChanged": false, "subscribe": false},
		"prompts":   map[string]any{"listChanged": false},
	}
}

func (srv *Server) dispatch(ctx context.Context, sess *Session, m *Message, modern bool) (any, *RPCError) {
	switch m.Method {
	case "initialize":
		return srv.initialize(sess, m.Params)
	case "ping":
		if modern {
			// ping was removed in the 2026-07-28 revision.
			return nil, &RPCError{Code: CodeMethodNotFound, Message: "method not found"}
		}
		return struct{}{}, nil
	case "server/discover":
		return map[string]any{
			// Versions usable in params._meta. Legacy clients negotiate
			// through initialize instead.
			"supportedVersions": ModernProtocolVersions,
			"capabilities":      srv.capabilities(),
			"instructions":      srv.Provider.Instructions(),
			"ttlMs":             3600000,
			"cacheScope":        "public",
		}, nil
	}
	if !modern && !sess.Initialized() {
		return nil, &RPCError{Code: CodeInvalidRequest, Message: "session is not initialized"}
	}
	switch m.Method {
	case "tools/list", "resources/list", "resources/templates/list", "prompts/list":
		// The server never issues nextCursor, so any cursor is invalid.
		var pg struct {
			Cursor *string `json:"cursor"`
		}
		if json.Unmarshal(m.Params, &pg) == nil && pg.Cursor != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid cursor"}
		}
	}
	p := sess.principal()
	switch m.Method {
	case "tools/list":
		return map[string]any{"tools": nonNil(srv.Provider.ListTools(ctx, p))}, nil
	case "tools/call":
		var params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(m.Params, &params); err != nil || params.Name == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "tools/call requires a tool name"}
		}
		if len(params.Arguments) == 0 || string(params.Arguments) == "null" {
			params.Arguments = json.RawMessage("{}")
		}
		if !strings.HasPrefix(strings.TrimSpace(string(params.Arguments)), "{") {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "tools/call arguments must be an object"}
		}
		return srv.Provider.CallTool(ctx, p, params.Name, params.Arguments)
	case "resources/list":
		return map[string]any{"resources": nonNil(srv.Provider.ListResources(ctx, p))}, nil
	case "resources/templates/list":
		return map[string]any{"resourceTemplates": nonNil(srv.Provider.ListResourceTemplates(ctx, p))}, nil
	case "resources/read":
		var params struct {
			URI string `json:"uri"`
		}
		if err := json.Unmarshal(m.Params, &params); err != nil || params.URI == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "resources/read requires a uri"}
		}
		contents, rpcErr := srv.Provider.ReadResource(ctx, p, params.URI)
		if rpcErr != nil {
			return nil, rpcErr
		}
		return map[string]any{"contents": contents}, nil
	case "prompts/list":
		return map[string]any{"prompts": nonNil(srv.Provider.ListPrompts(ctx, p))}, nil
	case "prompts/get":
		var params struct {
			Name      string            `json:"name"`
			Arguments map[string]string `json:"arguments"`
		}
		if err := json.Unmarshal(m.Params, &params); err != nil || params.Name == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "prompts/get requires a prompt name"}
		}
		return srv.Provider.GetPrompt(ctx, p, params.Name, params.Arguments)
	}
	return nil, &RPCError{Code: CodeMethodNotFound, Message: "method not found"}
}

func (srv *Server) initialize(sess *Session, raw json.RawMessage) (any, *RPCError) {
	var params initializeParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid initialize params"}
	}
	version := LegacyProtocolVersions[0]
	if slices.Contains(LegacyProtocolVersions, params.ProtocolVersion) {
		version = params.ProtocolVersion
	}
	sess.mu.Lock()
	sess.initialized = true
	sess.version = version
	sess.client = params.ClientInfo
	sess.mu.Unlock()
	return initializeResult{
		ProtocolVersion: version,
		Capabilities:    srv.capabilities(),
		ServerInfo:      srv.Info,
		Instructions:    srv.Provider.Instructions(),
	}, nil
}

// errorResponse builds an error response. An ID that could not be read is
// omitted, as the MCP schema makes the error response ID optional.
func errorResponse(id json.RawMessage, code int, msg string) *Message {
	return &Message{JSONRPC: "2.0", ID: id, Error: &RPCError{Code: code, Message: msg}}
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
