package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/audit"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/authz"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/domain"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/errs"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/trimble/connect"
)

type staticTok string

func (s staticTok) Token(context.Context) (string, error) { return string(s), nil }

type apiFixture struct {
	g     *Gateway
	calls *atomic.Int32
	last  *atomic.Value
}

func setupAPI(t *testing.T, h http.HandlerFunc) apiFixture {
	t.Helper()
	var calls atomic.Int32
	var last atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		last.Store(r.URL.RequestURI())
		if r.Method != http.MethodGet {
			t.Errorf("bridge sent %s; only GET is ever executed", r.Method)
		}
		h(w, r)
	}))
	t.Cleanup(srv.Close)
	ad, err := connect.New(connect.Config{
		Environment: connect.Production, Region: "us", Tokens: staticTok("tok"),
		BaseURLOverride: srv.URL + "/tc/api", AllowInsecureOverride: true,
		CatalogBaseOverride: map[string]string{"core": srv.URL + "/tc/api", "topics": srv.URL, "pset": srv.URL + "/v1"},
		RatePerSecond:       1000, Burst: 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	reg := NewRegistry()
	reg.Register("tenant-a", ad)
	g, err := New(Options{Registry: reg, Audit: &audit.Memory{}, CallsPerSecond: 1000, Burst: 1000})
	if err != nil {
		t.Fatal(err)
	}
	return apiFixture{g: g, calls: &calls, last: &last}
}

func apiPrincipal() *authz.Principal {
	return principal(authz.ScopeCapabilitiesRead, authz.ScopeProjectsRead, authz.ScopeAPIRead, authz.ScopeAPIPlan)
}

func TestAPIOperationsSearchAndDetail(t *testing.T) {
	f := setupAPI(t, func(http.ResponseWriter, *http.Request) {})
	env := invoke(t, f.g, apiPrincipal(), ToolAPIOperations, `{"api":"topics","disposition":"read","page_size":5}`)
	if env.Status != "ok" || env.Pagination.Complete || *env.Pagination.Total < 10 {
		t.Fatalf("%+v", env.Pagination)
	}
	env = invoke(t, f.g, apiPrincipal(), ToolAPIOperations, `{"key":"core:GET /files/fs/{fileId}/downloadurl"}`)
	b, _ := json.Marshal(env.Result)
	if !strings.Contains(string(b), `"disposition":"excluded"`) || !strings.Contains(string(b), "presigned") {
		t.Fatalf("%s", b)
	}
	if f.calls.Load() != 0 {
		t.Fatal("catalogue search must not call upstream")
	}
}

func TestAPIReadExecutesWithValidationAndRedaction(t *testing.T) {
	f := setupAPI(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"p1","name":"Bridge","thumbnailUrl":"https://s3.amazonaws.com/b/t.png?X-Amz-Signature=abc&X-Amz-Credential=x","accessToken":"secret-token","links":{"next":{"href":"https://app.connect.trimble.com/tc/api/2.1/projects?skipToken=q"}}}`))
	})
	env := invoke(t, f.g, apiPrincipal(), ToolAPIRead, `{"product":"trimble-connect","key":"core:GET /projects/{projectId}","path_params":{"projectId":"p1"},"query_params":{"fullyLoaded":true}}`)
	if env.Status != "ok" {
		t.Fatalf("%+v", env.Error)
	}
	if got := f.last.Load().(string); got != "/tc/api/2.0/projects/p1?fullyLoaded=true" {
		t.Fatalf("request %s", got)
	}
	b, _ := json.Marshal(env.Result)
	s := string(b)
	if strings.Contains(s, "abc") || strings.Contains(s, "secret-token") || !strings.Contains(s, "skipToken=q") {
		t.Fatalf("redaction wrong: %s", s)
	}
	if len(env.Warnings) == 0 || len(env.UntrustedFields) == 0 {
		t.Fatal("redaction warning and untrusted label expected")
	}
}

func TestAPIReadRejections(t *testing.T) {
	f := setupAPI(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{}`)) })
	cases := map[string]struct {
		args string
		code errs.Code
	}{
		"undocumented query":   {`{"product":"trimble-connect","key":"core:GET /projects/{projectId}","path_params":{"projectId":"p1"},"query_params":{"evil":"1"}}`, errs.Validation},
		"missing path param":   {`{"product":"trimble-connect","key":"core:GET /projects/{projectId}"}`, errs.Validation},
		"path traversal":       {`{"product":"trimble-connect","key":"core:GET /projects/{projectId}","path_params":{"projectId":"../x"}}`, errs.Validation},
		"wrong type":           {`{"product":"trimble-connect","key":"core:GET /projects/{projectId}","path_params":{"projectId":"p1"},"query_params":{"fullyLoaded":"maybe"}}`, errs.Validation},
		"excluded":             {`{"product":"trimble-connect","key":"core:GET /files/fs/{fileId}/downloadurl","path_params":{"fileId":"f"}}`, errs.PolicyDenied},
		"change via read":      {`{"product":"trimble-connect","key":"core:DELETE /projects/{projectId}","path_params":{"projectId":"p1"}}`, errs.Validation},
		"variant":              {`{"product":"trimble-connect","key":"tcps-stage@2.0:GET /projects/{projectId}","path_params":{"projectId":"p1"}}`, errs.Validation},
		"unknown key":          {`{"product":"trimble-connect","key":"core:GET /nope"}`, errs.Validation},
		"header not permitted": {`{"product":"trimble-connect","key":"core:GET /projects/{projectId}","path_params":{"projectId":"p1"},"header_params":{"Authorization":"Bearer x"}}`, errs.Validation},
		"body on read":         {`{"product":"trimble-connect","key":"core:GET /projects/{projectId}","path_params":{"projectId":"p1"},"body":{}}`, errs.Validation},
	}
	for name, c := range cases {
		env := invoke(t, f.g, apiPrincipal(), ToolAPIRead, c.args)
		if env.Error == nil || env.Error.Code != c.code {
			t.Errorf("%s: got %+v", name, env.Error)
		}
	}
	if f.calls.Load() != 0 {
		t.Fatalf("rejected calls reached upstream %d times", f.calls.Load())
	}
}

func TestAPIReadEnforcesProjectGrant(t *testing.T) {
	f := setupAPI(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`[]`)) })
	p := apiPrincipal()
	p.Projects = []domain.ProjectID{"p2"}
	env := invoke(t, f.g, p, ToolAPIRead, `{"product":"trimble-connect","key":"core:GET /projects/{projectId}","path_params":{"projectId":"p1"}}`)
	if env.Error == nil || env.Error.Code != errs.Authorization {
		t.Fatalf("other project: %+v", env.Error)
	}
	// An operation that names no project cannot be bound to the grant.
	env = invoke(t, f.g, p, ToolAPIRead, `{"product":"trimble-connect","key":"core:GET /2.1/projects"}`)
	if env.Error == nil || env.Error.Code != errs.PolicyDenied {
		t.Fatalf("unbound op: %+v", env.Error)
	}
	env = invoke(t, f.g, p, ToolAPIRead, `{"product":"trimble-connect","key":"core:GET /projects/{projectId}","path_params":{"projectId":"p2"}}`)
	if env.Status != "ok" {
		t.Fatalf("granted project: %+v", env.Error)
	}
}

func TestAPIReadOtherAPIsAndScope(t *testing.T) {
	f := setupAPI(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`[{"guid":"t1","title":"Clash"}]`)) })
	key := ""
	for _, o := range []string{"topics:GET /bcf/3.0/projects/{project_id}/topics", "topics:GET /bcf/2.1/projects/{project_id}/topics"} {
		if _, ok := lookupKey(o); ok {
			key = o
			break
		}
	}
	if key == "" {
		t.Skip("topics list operation not in catalogue")
	}
	env := invoke(t, f.g, apiPrincipal(), ToolAPIRead, `{"product":"trimble-connect","key":"`+key+`","path_params":{"project_id":"p1"}}`)
	if env.Status != "ok" {
		t.Fatalf("%+v", env.Error)
	}
	if names := toolNames(f.g, principal(authz.ScopeCapabilitiesRead)); strings.Contains(strings.Join(names, ","), ToolAPIRead) {
		t.Fatal("api read visible without trimble:api:read")
	}
}

func TestAPIPlanNeverSends(t *testing.T) {
	f := setupAPI(t, func(http.ResponseWriter, *http.Request) {})
	env := invoke(t, f.g, apiPrincipal(), ToolAPIPlan, `{"product":"trimble-connect","key":"core:DELETE /projects/{projectId}","path_params":{"projectId":"p1"},"reason":"cleanup"}`)
	if env.Status != "ok" {
		t.Fatalf("%+v", env.Error)
	}
	b, _ := json.Marshal(env.Result)
	s := string(b)
	if !strings.Contains(s, `"executed":false`) || !strings.Contains(s, `"approval_level":"L4"`) || !strings.Contains(s, "/tc/api/2.0/projects/p1") {
		t.Fatalf("%s", s)
	}
	if f.calls.Load() != 0 {
		t.Fatal("a plan reached upstream")
	}
	for name, args := range map[string]string{
		"read via plan": `{"product":"trimble-connect","key":"core:GET /projects/{projectId}","path_params":{"projectId":"p1"},"reason":"x"}`,
		"no reason":     `{"product":"trimble-connect","key":"core:DELETE /projects/{projectId}","path_params":{"projectId":"p1"},"reason":""}`,
	} {
		if env := invoke(t, f.g, apiPrincipal(), ToolAPIPlan, args); env.Error == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

func lookupKey(k string) (string, bool) {
	return k, slices.Contains(catalogKeys(), k)
}
