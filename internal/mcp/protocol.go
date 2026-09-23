// Package mcp is a small, dependency-free Model Context Protocol server:
// JSON-RPC 2.0 message handling plus stdio and Streamable HTTP transports.
// It knows nothing about Trimble; tools, resources, and prompts are
// registered by the gateway package.
package mcp

import (
	"encoding/json"
	"regexp"
	"strings"
)

// Protocol revisions. The server is dual-era:
//
//   - Legacy revisions (2025-03-26 through 2025-11-25) use the initialize
//     handshake and, over HTTP, Mcp-Session-Id sessions. The server echoes the
//     client's requested revision when supported and otherwise offers the
//     newest legacy revision.
//   - The modern revision (2026-07-28) is stateless: every request carries its
//     protocol version in params._meta and the server implements
//     server/discover. The mode is chosen per request.
var (
	LegacyProtocolVersions    = []string{"2025-11-25", "2025-06-18", "2025-03-26"}
	ModernProtocolVersions    = []string{"2026-07-28"}
	SupportedProtocolVersions = append(append([]string{}, ModernProtocolVersions...), LegacyProtocolVersions...)
)

// Metadata keys defined by the modern revision.
const (
	MetaProtocolVersion    = "io.modelcontextprotocol/protocolVersion"
	MetaClientCapabilities = "io.modelcontextprotocol/clientCapabilities"
	MetaClientInfo         = "io.modelcontextprotocol/clientInfo"
	MetaServerInfo         = "io.modelcontextprotocol/serverInfo"
)

// JSON-RPC 2.0 error codes.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
	// CodeResourceNotFoundLegacy is the pre-2026 code for resources/read
	// misses; the modern revision uses CodeInvalidParams instead.
	CodeResourceNotFoundLegacy = -32002
	// CodeHeaderMismatch: an HTTP routing header disagrees with the body.
	CodeHeaderMismatch = -32020
	// CodeUnsupportedVersion: the requested protocol revision is unsupported.
	CodeUnsupportedVersion = -32022
)

// Message is a JSON-RPC request, notification, or response.
type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// IsNotification reports whether m is a request without an ID.
func (m *Message) IsNotification() bool { return m.Method != "" && len(m.ID) == 0 }

var integerID = regexp.MustCompile(`^-?(0|[1-9][0-9]*)$`)

// validID reports whether raw is an MCP request ID: a JSON string or an
// integer. Unlike base JSON-RPC, MCP forbids null IDs.
func validID(raw json.RawMessage) bool {
	s := strings.TrimSpace(string(raw))
	if integerID.MatchString(s) {
		return true
	}
	var str string
	return strings.HasPrefix(s, `"`) && json.Unmarshal(raw, &str) == nil
}

// DecodeMessage parses one JSON-RPC message. On failure it returns the error
// response to send: -32700 for invalid JSON, -32600 for JSON that is not a
// valid MCP message (non-object, bad jsonrpc, non-string method, or an ID
// that is null or not a string or integer). The ID is echoed only when it
// could be read.
func DecodeMessage(b []byte) (*Message, *Message) {
	if !json.Valid(b) {
		return nil, errorResponse(nil, CodeParseError, "parse error")
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal(b, &raw) != nil || raw == nil {
		return nil, errorResponse(nil, CodeInvalidRequest, "message must be a JSON object")
	}
	m := &Message{Params: raw["params"]}
	idRaw, hasID := raw["id"]
	if hasID {
		if !validID(idRaw) {
			return nil, errorResponse(nil, CodeInvalidRequest, "id must be a string or integer, not null")
		}
		m.ID = idRaw
	}
	if json.Unmarshal(raw["jsonrpc"], &m.JSONRPC) != nil || m.JSONRPC != "2.0" {
		return nil, errorResponse(m.ID, CodeInvalidRequest, "jsonrpc must be \"2.0\"")
	}
	if mr, ok := raw["method"]; ok {
		if json.Unmarshal(mr, &m.Method) != nil || m.Method == "" {
			return nil, errorResponse(m.ID, CodeInvalidRequest, "method must be a non-empty string")
		}
	} else if !hasID {
		return nil, errorResponse(nil, CodeInvalidRequest, "method is required")
	}
	if r, ok := raw["result"]; ok {
		m.Result = r
	}
	return m, nil
}

// IsResponse reports whether m is a response from the client.
func (m *Message) IsResponse() bool { return m.Method == "" && len(m.ID) > 0 }

// RPCError is a JSON-RPC error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Implementation identifies client or server software.
type Implementation struct {
	Name    string `json:"name"`
	Title   string `json:"title,omitempty"`
	Version string `json:"version"`
}

type initializeParams struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ClientInfo      Implementation `json:"clientInfo"`
}

type initializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ServerInfo      Implementation `json:"serverInfo"`
	Instructions    string         `json:"instructions,omitempty"`
}

// ToolAnnotations are advisory hints for clients. They are never relied on
// for enforcement; authorization happens server-side.
type ToolAnnotations struct {
	Title           string `json:"title,omitempty"`
	ReadOnlyHint    bool   `json:"readOnlyHint"`
	DestructiveHint bool   `json:"destructiveHint"`
	IdempotentHint  bool   `json:"idempotentHint"`
	OpenWorldHint   bool   `json:"openWorldHint"`
}

// ToolInfo is the tools/list entry.
type ToolInfo struct {
	Name         string          `json:"name"`
	Title        string          `json:"title,omitempty"`
	Description  string          `json:"description"`
	InputSchema  json.RawMessage `json:"inputSchema"`
	OutputSchema json.RawMessage `json:"outputSchema,omitempty"`
	Annotations  ToolAnnotations `json:"annotations"`
}

// Content is a tool result content block.
type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// CallToolResult is the tools/call result.
type CallToolResult struct {
	Content           []Content `json:"content"`
	StructuredContent any       `json:"structuredContent,omitempty"`
	IsError           bool      `json:"isError"`
}

// ResourceInfo is a resources/list entry.
type ResourceInfo struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ResourceTemplate is a resources/templates/list entry.
type ResourceTemplate struct {
	URITemplate string `json:"uriTemplate"`
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ResourceContents is one item in a resources/read result.
type ResourceContents struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text"`
}

// PromptArgument describes a prompt argument.
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
}

// PromptInfo is a prompts/list entry.
type PromptInfo struct {
	Name        string           `json:"name"`
	Title       string           `json:"title,omitempty"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptMessage is one message returned by prompts/get.
type PromptMessage struct {
	Role    string  `json:"role"`
	Content Content `json:"content"`
}

// GetPromptResult is the prompts/get result.
type GetPromptResult struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}
