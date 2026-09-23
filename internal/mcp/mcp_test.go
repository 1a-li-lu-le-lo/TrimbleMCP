package mcp_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/audit"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/authz"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/gateway"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/mcp"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/trimble/mock"
)

func server(t testing.TB) *mcp.Server {
	t.Helper()
	reg := gateway.NewRegistry()
	reg.Register("t1", mock.New())
	g, err := gateway.New(gateway.Options{Registry: reg, Audit: &audit.Memory{}, CallsPerSecond: 1000, Burst: 1000})
	if err != nil {
		t.Fatal(err)
	}
	return &mcp.Server{Info: mcp.Implementation{Name: "trimble-mcp-bridge", Version: "test"}, Provider: g}
}

var alice = &authz.Principal{Subject: "alice", Tenant: "t1", Scopes: []authz.Scope{authz.ScopeCapabilitiesRead, authz.ScopeProjectsRead, authz.ScopeFilesRead}}

// stdio runs the lines through ServeStdio and returns responses keyed by
// request ID (JSON text), plus responses that carry no ID. Requests may be
// answered out of order, as JSON-RPC permits.
func stdio(t *testing.T, lines ...string) (map[string]map[string]any, []map[string]any) {
	t.Helper()
	var out bytes.Buffer
	in := strings.NewReader(strings.Join(lines, "\n") + "\n")
	if err := server(t).ServeStdio(context.Background(), &mcp.Session{Principal: alice}, in, &out); err != nil {
		t.Fatal(err)
	}
	byID := map[string]map[string]any{}
	var noID []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if l == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("stdout carried a non-JSON line: %q", l)
		}
		id, ok := m["id"]
		if !ok {
			noID = append(noID, m)
			continue
		}
		if id == nil {
			t.Fatalf("response carries a null id: %s", l)
		}
		k, _ := json.Marshal(id)
		byID[string(k)] = m
	}
	return byID, noID
}

func errCode(t *testing.T, m map[string]any) float64 {
	t.Helper()
	e, ok := m["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected an error response, got %v", m)
	}
	return e["code"].(float64)
}

func TestStdioLegacyLifecycle(t *testing.T) {
	res, noID := stdio(t,
		`{"jsonrpc":"2.0","id":0,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"c","version":"1"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"trimble_list_projects","arguments":{"product":"mock","page_size":2}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"trimble_nope","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":"five","method":"ping"}`,
		`not json`,
		`[{"jsonrpc":"2.0","id":6,"method":"ping"}]`,
	)
	if len(res) != 6 || len(noID) != 2 {
		t.Fatalf("got %d id responses and %d id-less: %v %v", len(res), len(noID), res, noID)
	}
	if errCode(t, res["0"]) != mcp.CodeInvalidRequest {
		t.Fatal("requests before initialize must fail")
	}
	init := res["1"]["result"].(map[string]any)
	if init["protocolVersion"] != "2025-06-18" || init["instructions"] == "" {
		t.Fatalf("initialize: %v", init)
	}
	tools := res["2"]["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 5 {
		t.Fatalf("tools: %d", len(tools))
	}
	for _, tl := range tools {
		ann := tl.(map[string]any)["annotations"].(map[string]any)
		if ann["readOnlyHint"] != true || ann["destructiveHint"] != false {
			t.Fatalf("annotations: %v", ann)
		}
	}
	call := res["3"]["result"].(map[string]any)
	if call["isError"] != false || call["structuredContent"].(map[string]any)["audit_id"] == "" {
		t.Fatalf("call: %v", call)
	}
	if errCode(t, res["4"]) != mcp.CodeInvalidParams {
		t.Fatalf("unknown tool: %v", res["4"])
	}
	if _, ok := res[`"five"`]["result"]; !ok {
		t.Fatal("legacy ping with a string id must succeed")
	}
	if errCode(t, noID[0]) != mcp.CodeParseError || errCode(t, noID[1]) != mcp.CodeInvalidRequest {
		t.Fatalf("parse error / batch: %v", noID)
	}
}

func TestStdioInvalidIDsAndMethods(t *testing.T) {
	res, noID := stdio(t,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"c","version":"1"}}}`,
		`{"jsonrpc":"2.0","id":null,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":1.5,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":{"a":1},"method":"ping"}`,
		`{"jsonrpc":"2.0","id":true,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":7,"method":5}`,
		`{"jsonrpc":"1.0","id":8,"method":"ping"}`,
		`5`,
		`{"jsonrpc":"2.0","id":9,"method":"tools/list","params":{"cursor":"zzz"}}`,
		`{"jsonrpc":"2.0","id":10,"method":"tools/call","params":{"name":"trimble_list_projects","arguments":"x"}}`,
		`{"jsonrpc":"2.0","id":11,"method":"tools/call","params":{"name":"trimble_list_projects","arguments":{"product":"mock","page_size":0}}}`,
	)
	if len(noID) != 5 {
		t.Fatalf("null/fraction/object/bool ids and a non-object message must be rejected without an id: %v", noID)
	}
	for _, m := range noID {
		if errCode(t, m) != mcp.CodeInvalidRequest {
			t.Fatalf("%v", m)
		}
	}
	if errCode(t, res["7"]) != mcp.CodeInvalidRequest || errCode(t, res["8"]) != mcp.CodeInvalidRequest {
		t.Fatal("non-string method / wrong jsonrpc must be -32600 with the id")
	}
	if errCode(t, res["9"]) != mcp.CodeInvalidParams {
		t.Fatal("an unknown cursor must be rejected")
	}
	if errCode(t, res["10"]) != mcp.CodeInvalidParams {
		t.Fatal("non-object arguments are a protocol error")
	}
	r11 := res["11"]["result"].(map[string]any)
	if r11["isError"] != true || !strings.Contains(r11["content"].([]any)[0].(map[string]any)["text"].(string), "page_size") {
		t.Fatalf("page_size 0 must be a validation error: %v", r11)
	}
}

func TestStdioCancellation(t *testing.T) {
	// A cancelled request gets no response; others still do.
	res, _ := stdio(t,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"c","version":"1"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":99}}`,
	)
	if _, ok := res["2"]; !ok {
		t.Fatal("uncancelled request lost")
	}
}

func TestLegacyVersionNegotiation(t *testing.T) {
	res, _ := stdio(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"1999-01-01","capabilities":{},"clientInfo":{"name":"c","version":"1"}}}`)
	if v := res["1"]["result"].(map[string]any)["protocolVersion"]; v != mcp.LegacyProtocolVersions[0] {
		t.Fatalf("got %v", v)
	}
}

const modernMeta = `"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}`

func TestStdioModernStateless(t *testing.T) {
	res, _ := stdio(t,
		`{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{`+modernMeta+`}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{`+modernMeta+`}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2031-01-01","io.modelcontextprotocol/clientCapabilities":{}}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"resources/read","params":{"uri":"trimble://nope",`+modernMeta+`}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`,
		`{"jsonrpc":"2.0","id":6,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":5,"io.modelcontextprotocol/clientCapabilities":{}}}}`,
		`{"jsonrpc":"2.0","id":7,"method":"ping","params":{`+modernMeta+`}}`,
		`{"jsonrpc":"2.0","id":8,"method":"resources/read","params":{"uri":"trimble://nosuch/projects/x",`+modernMeta+`}}`,
		`{"jsonrpc":"2.0","id":9,"method":"server/discover"}`,
	)
	disc := res["1"]["result"].(map[string]any)
	if disc["resultType"] != "complete" {
		t.Fatalf("discover: %v", disc)
	}
	sv := disc["supportedVersions"].([]any)
	if len(sv) != len(mcp.ModernProtocolVersions) {
		t.Fatalf("discover must list versions usable in _meta: %v", sv)
	}
	list := res["2"]["result"].(map[string]any)
	if list["resultType"] != "complete" || list["cacheScope"] != "private" || list["ttlMs"] == nil {
		t.Fatalf("modern list decoration: %v", list)
	}
	if _, ok := list["_meta"].(map[string]any)[mcp.MetaServerInfo]; !ok {
		t.Fatal("serverInfo missing from _meta")
	}
	e := res["3"]["error"].(map[string]any)
	if e["code"].(float64) != mcp.CodeUnsupportedVersion || len(e["data"].(map[string]any)["supported"].([]any)) != len(mcp.ModernProtocolVersions) {
		t.Fatalf("unsupported version: %v", e)
	}
	if errCode(t, res["4"]) != mcp.CodeInvalidParams || errCode(t, res["8"]) != mcp.CodeInvalidParams {
		t.Fatal("modern resource-not-found must be -32602 and never success content")
	}
	if errCode(t, res["5"]) != mcp.CodeInvalidParams || errCode(t, res["6"]) != mcp.CodeInvalidParams {
		t.Fatal("missing clientCapabilities / non-string version must be -32602")
	}
	if errCode(t, res["7"]) != mcp.CodeMethodNotFound {
		t.Fatal("ping was removed in the modern revision")
	}
	if res["9"]["result"].(map[string]any)["resultType"] != "complete" {
		t.Fatal("discover without _meta still needs resultType")
	}
}

// ---- HTTP ----

const bobToken = "bob-secret-token"
const aliceToken = "alice-secret-token"

func httpHandler(t *testing.T) http.Handler {
	t.Helper()
	auth := &mcp.StaticTokenAuthenticator{}
	for tok, sub := range map[string]string{aliceToken: "alice", bobToken: "bob"} {
		sum := sha256.Sum256([]byte(tok))
		if err := auth.AddTokenHash(hex.EncodeToString(sum[:]), authz.Principal{Subject: sub, Tenant: "t1", Scopes: alice.Scopes}); err != nil {
			t.Fatal(err)
		}
	}
	return mcp.NewHTTPHandler(server(t), mcp.HTTPOptions{
		Auth: auth, AllowedOrigins: []string{"https://claude.ai"},
		ResourceMetadataURL: "https://mcp.example.com/.well-known/oauth-protected-resource",
	})
}

func post(h http.Handler, token, session, body string, hdr map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json, text/event-stream")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	if session != "" {
		r.Header.Set("Mcp-Session-Id", session)
	}
	for k, v := range hdr {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

const initBody = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"c","version":"1"}}}`

func TestHTTPAuthChallenge(t *testing.T) {
	h := httpHandler(t)
	w := post(h, "", "", initBody, nil)
	if w.Code != 401 || !strings.Contains(w.Header().Get("WWW-Authenticate"), `resource_metadata="https://mcp.example.com/`) {
		t.Fatalf("code %d hdr %q", w.Code, w.Header().Get("WWW-Authenticate"))
	}
	if w := post(h, "wrong", "", initBody, nil); w.Code != 401 {
		t.Fatalf("bad token: %d", w.Code)
	}
	r := httptest.NewRequest(http.MethodPost, "/mcp?access_token="+aliceToken, strings.NewReader(initBody))
	r.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatal("tokens in the query string must not be accepted")
	}
}

func TestHTTPOriginAndMethod(t *testing.T) {
	h := httpHandler(t)
	if w := post(h, aliceToken, "", initBody, map[string]string{"Origin": "https://evil.example"}); w.Code != 403 {
		t.Fatalf("origin: %d", w.Code)
	}
	if w := post(h, aliceToken, "", initBody, map[string]string{"Origin": "https://claude.ai"}); w.Code != 200 {
		t.Fatalf("allowed origin: %d", w.Code)
	}
	r := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 405 {
		t.Fatalf("GET: %d", w.Code)
	}
}

func TestHTTPLegacySessionBinding(t *testing.T) {
	h := httpHandler(t)
	w := post(h, aliceToken, "", initBody, nil)
	sid := w.Header().Get("Mcp-Session-Id")
	if w.Code != 200 || sid == "" {
		t.Fatalf("init: %d %q", w.Code, sid)
	}
	if w := post(h, aliceToken, sid, `{"jsonrpc":"2.0","method":"notifications/initialized"}`, nil); w.Code != 202 {
		t.Fatalf("notification: %d", w.Code)
	}
	list := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
	if w := post(h, aliceToken, sid, list, nil); w.Code != 200 || !strings.Contains(w.Body.String(), "trimble_list_projects") {
		t.Fatalf("list: %d %s", w.Code, w.Body.String())
	}
	if w := post(h, aliceToken, "", list, nil); w.Code != 400 {
		t.Fatalf("missing session: %d", w.Code)
	}
	if w := post(h, bobToken, sid, list, nil); w.Code != 404 {
		t.Fatalf("session hijack by another subject must fail: %d", w.Code)
	}
	if w := post(h, aliceToken, sid, list, map[string]string{"MCP-Protocol-Version": "1999-01-01"}); w.Code != 400 || !strings.Contains(w.Body.String(), `"id":2`) {
		t.Fatalf("bad version header must keep the request id: %d %s", w.Code, w.Body.String())
	}
	if w := post(h, aliceToken, sid, `[`+list+`]`, nil); w.Code != 400 {
		t.Fatalf("batch: %d", w.Code)
	}
	del := httptest.NewRequest(http.MethodDelete, "/mcp", nil)
	del.Header.Set("Authorization", "Bearer "+aliceToken)
	del.Header.Set("Mcp-Session-Id", sid)
	dw := httptest.NewRecorder()
	h.ServeHTTP(dw, del)
	if dw.Code != 204 {
		t.Fatalf("delete: %d", dw.Code)
	}
	if w := post(h, aliceToken, sid, list, nil); w.Code != 404 {
		t.Fatalf("deleted session reused: %d", w.Code)
	}
}

func TestHTTPModernHeaders(t *testing.T) {
	h := httpHandler(t)
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"trimble_get_capabilities","arguments":{},` + modernMeta + `}}`
	ok := map[string]string{"MCP-Protocol-Version": "2026-07-28", "Mcp-Method": "tools/call", "Mcp-Name": "trimble_get_capabilities"}
	if w := post(h, aliceToken, "", body, ok); w.Code != 200 || !strings.Contains(w.Body.String(), `"resultType":"complete"`) {
		t.Fatalf("modern call: %d %s", w.Code, w.Body.String())
	}
	b64 := map[string]string{"MCP-Protocol-Version": "2026-07-28", "Mcp-Method": "tools/call", "Mcp-Name": "=?base64?dHJpbWJsZV9nZXRfY2FwYWJpbGl0aWVz?="}
	if w := post(h, aliceToken, "", body, b64); w.Code != 200 {
		t.Fatalf("base64 Mcp-Name: %d", w.Code)
	}
	bad := map[string]string{"MCP-Protocol-Version": "2026-07-28", "Mcp-Method": "tools/call", "Mcp-Name": "trimble_list_projects"}
	if w := post(h, aliceToken, "", body, bad); w.Code != 400 || !strings.Contains(w.Body.String(), "-32020") {
		t.Fatalf("name mismatch: %d %s", w.Code, w.Body.String())
	}
	noMethod := map[string]string{"MCP-Protocol-Version": "2026-07-28"}
	if w := post(h, aliceToken, "", body, noMethod); w.Code != 400 {
		t.Fatalf("missing Mcp-Method: %d", w.Code)
	}
}

func TestHTTPRateLimit(t *testing.T) {
	auth := &mcp.StaticTokenAuthenticator{}
	sum := sha256.Sum256([]byte(aliceToken))
	_ = auth.AddTokenHash(hex.EncodeToString(sum[:]), *alice)
	h := mcp.NewHTTPHandler(server(t), mcp.HTTPOptions{Auth: auth, RatePerSecond: 0.001, Burst: 1})
	post(h, aliceToken, "", initBody, nil)
	w := post(h, aliceToken, "", initBody, nil)
	if w.Code != 429 || w.Header().Get("Retry-After") == "" {
		t.Fatalf("rate limit: %d", w.Code)
	}
}

func FuzzStdioNeverPanicsAndEmitsOnlyJSON(f *testing.F) {
	f.Add(initBody)
	f.Add(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"trimble_list_projects","arguments":{"product":"mock","page_token":"!!"},` + modernMeta + `}}`)
	f.Add(`{"jsonrpc":"2.0","id":{"x":1},"method":7}`)
	f.Fuzz(func(t *testing.T, line string) {
		var out bytes.Buffer
		s := server(t)
		sess := &mcp.Session{Principal: alice}
		in := strings.NewReader(initBody + "\n" + strings.ReplaceAll(line, "\n", " ") + "\n")
		_ = s.ServeStdio(context.Background(), sess, in, &out)
		for _, l := range strings.Split(strings.TrimSpace(out.String()), "\n") {
			if l != "" && !json.Valid([]byte(l)) {
				t.Fatalf("invalid JSON on stdout: %q", l)
			}
		}
	})
}

func TestHTTPSpecEdges(t *testing.T) {
	h := httpHandler(t)
	// DELETE without a session: 405.
	del := httptest.NewRequest(http.MethodDelete, "/mcp", nil)
	del.Header.Set("Authorization", "Bearer "+aliceToken)
	dw := httptest.NewRecorder()
	h.ServeHTTP(dw, del)
	if dw.Code != 405 {
		t.Fatalf("DELETE without session: %d", dw.Code)
	}
	// Modern header but no _meta: JSON 400 with -32602, not plain text.
	w := post(h, aliceToken, "", `{"jsonrpc":"2.0","id":3,"method":"tools/list"}`, map[string]string{"MCP-Protocol-Version": "2026-07-28", "Mcp-Method": "tools/list"})
	if w.Code != 400 || !strings.Contains(w.Body.String(), "-32602") {
		t.Fatalf("modern header without meta: %d %s", w.Code, w.Body.String())
	}
	// Missing clientCapabilities over HTTP: 400.
	body := `{"jsonrpc":"2.0","id":4,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`
	w = post(h, aliceToken, "", body, map[string]string{"MCP-Protocol-Version": "2026-07-28", "Mcp-Method": "tools/list"})
	if w.Code != 400 {
		t.Fatalf("missing clientCapabilities: %d", w.Code)
	}
	// Null id over HTTP: rejected.
	body = `{"jsonrpc":"2.0","id":null,"method":"tools/list","params":{` + modernMeta + `}}`
	w = post(h, aliceToken, "", body, map[string]string{"MCP-Protocol-Version": "2026-07-28", "Mcp-Method": "tools/list"})
	if w.Code != 400 || strings.Contains(w.Body.String(), `"tools"`) {
		t.Fatalf("null id: %d %s", w.Code, w.Body.String())
	}
	// Bad bearer token: invalid_token in the challenge.
	w = post(h, "wrong", "", initBody, nil)
	if !strings.Contains(w.Header().Get("WWW-Authenticate"), `error="invalid_token"`) {
		t.Fatalf("challenge: %q", w.Header().Get("WWW-Authenticate"))
	}
}
