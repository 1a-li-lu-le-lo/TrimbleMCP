package connect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/domain"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/errs"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/trimble"
)

// Fixtures below reproduce only the response fields documented in the
// Trimble Connect OpenAPI definition (tcps 2.0 / 2.1 paths).

type staticTokens string

func (s staticTokens) Token(context.Context) (string, error) { return string(s), nil }

func newTest(t *testing.T, h http.HandlerFunc) (*Adapter, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	a, err := New(Config{
		BaseURLOverride: srv.URL + "/tc/api", AllowInsecureOverride: true,
		Tokens: staticTokens("test-token"), RatePerSecond: 1000, Burst: 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	a.sleep = func(context.Context, time.Duration) error { return nil }
	return a, srv
}

func TestBaseURLOnlyDocumentedHosts(t *testing.T) {
	u, err := BaseURL(Production, "eu")
	if err != nil || u != "https://app21.connect.trimble.com/tc/api" {
		t.Fatalf("got %q %v", u, err)
	}
	if _, err := BaseURL(Stage, "ap-au"); err == nil {
		t.Fatal("undocumented staging region must be rejected")
	}
	if _, err := BaseURL(Production, "mars"); err == nil {
		t.Fatal("unknown region must be rejected")
	}
}

func TestOverrideRequiresExplicitTestFlag(t *testing.T) {
	_, err := New(Config{BaseURLOverride: "https://evil.example/tc/api", Tokens: staticTokens("x")})
	if !errs.Is(err, errs.Configuration) {
		t.Fatalf("expected configuration error, got %v", err)
	}
}

func TestListProjectsPaginationAndAuth(t *testing.T) {
	var sawToken atomic.Bool
	a, srv := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tc/api/2.1/projects" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") == "Bearer test-token" {
			sawToken.Store(true)
		}
		if strings.Contains(r.URL.RawQuery, "test-token") {
			t.Error("token leaked into query string")
		}
		if r.URL.Query().Get("pageSize") != "2" {
			t.Errorf("pageSize = %q", r.URL.Query().Get("pageSize"))
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("skipToken") {
		case "":
			w.Write([]byte(`{"items":[{"id":"p1","name":"Bridge","rootId":"f1","createdAt":"2026-01-02T03:04:05Z","updatedAt":"2026-02-03T04:05:06.123Z","size":10},
			{"id":"p2","name":"Tunnel","rootId":"f2"}],
			"links":{"self":{"href":"/tc/api/2.1/projects"},"next":{"href":"` + "http://" + r.Host + `/tc/api/2.1/projects?pageSize=2&skipToken=abc"}}}`))
		case "abc":
			w.Write([]byte(`{"items":[{"id":"p3","name":"Dam","rootId":"f3"}],"links":{"self":{"href":"x"}}}`))
		default:
			w.WriteHeader(400)
		}
	})
	_ = srv
	ctx := context.Background()
	p1, err := a.ListProjects(ctx, trimble.ListProjectsQuery{Page: domain.PageRequest{Size: 2}})
	if err != nil {
		t.Fatal(err)
	}
	if !sawToken.Load() {
		t.Fatal("bearer token not sent in Authorization header")
	}
	if len(p1.Projects) != 2 || p1.Page.Complete || p1.Page.NextToken == "" {
		t.Fatalf("page 1: %+v", p1.Page)
	}
	if p1.Projects[0].RootFolder != "f1" || p1.Projects[0].CreatedAt == nil || p1.Projects[0].CreatedAt.Location() != time.UTC {
		t.Fatalf("project mapping: %+v", p1.Projects[0])
	}
	p2, err := a.ListProjects(ctx, trimble.ListProjectsQuery{Page: domain.PageRequest{Size: 2, Token: p1.Page.NextToken}})
	if err != nil {
		t.Fatal(err)
	}
	if len(p2.Projects) != 1 || !p2.Page.Complete || p2.Page.NextToken != "" {
		t.Fatalf("page 2: %+v", p2.Page)
	}
}

func TestPaginationLinkToForeignHostRejected(t *testing.T) {
	a, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"items":[],"links":{"next":{"href":"https://attacker.example/tc/api/2.1/projects?skipToken=zzz"}}}`))
	})
	_, err := a.ListProjects(context.Background(), trimble.ListProjectsQuery{})
	if !errs.Is(err, errs.UpstreamMalformed) {
		t.Fatalf("expected malformed error, got %v", err)
	}
}

func TestRepeatedCursorDetected(t *testing.T) {
	a, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"items":[],"links":{"next":{"href":"/tc/api/2.1/projects?skipToken=same"}}}`))
	})
	tok := encodeToken("same")
	_, err := a.ListProjects(context.Background(), trimble.ListProjectsQuery{Page: domain.PageRequest{Token: tok}})
	if !errs.Is(err, errs.UpstreamMalformed) {
		t.Fatalf("expected repeated-cursor error, got %v", err)
	}
}

func TestMissingItemsIsMalformed(t *testing.T) {
	a, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"links":{}}`)) })
	if _, err := a.ListProjects(context.Background(), trimble.ListProjectsQuery{}); !errs.Is(err, errs.UpstreamMalformed) {
		t.Fatalf("got %v", err)
	}
}

func TestForeignTokenRejected(t *testing.T) {
	a, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) { t.Error("must not call upstream") })
	if _, err := a.ListProjects(context.Background(), trimble.ListProjectsQuery{Page: domain.PageRequest{Token: "bm90LWEtdG9rZW4"}}); !errs.Is(err, errs.Validation) {
		t.Fatalf("got %v", err)
	}
}

func TestFolderItemsEnforceProject(t *testing.T) {
	a, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("fields") != "size" {
			t.Errorf("fields=size must be requested to get documented sizes")
		}
		if r.URL.Path != "/tc/api/2.1/folders/f1/items" && r.URL.Path != "/tc/api/2.1/folders/fold:er@x/items" {
			t.Errorf("path %s (raw %s)", r.URL.Path, r.URL.EscapedPath())
		}
		w.Write([]byte(`{"items":[
		  {"id":"d1","name":"Drawings","type":"FOLDER","parentId":"f1","parentType":"FOLDER","projectId":"p1","hasChildren":true},
		  {"id":"x1","name":"model.ifc","type":"FILE","versionId":"v1","parentId":"f1","modifiedOn":"2026-03-01T00:00:00Z","projectId":"p1","size":42,"hash":"d41d8cd98f00b204e9800998ecf8427e"}],
		  "links":{}}`))
	})
	ctx := context.Background()
	page, err := a.ListFolderItems(ctx, "p1", "f1", domain.PageRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || page.Items[0].Kind != trimble.KindFolder || page.Items[1].Kind != trimble.KindFile || !page.Page.Complete {
		t.Fatalf("items: %+v", page)
	}
	if f := page.Items[1]; f.SizeBytes == nil || *f.SizeBytes != 42 || f.ChecksumAlgorithm != "md5" {
		t.Fatalf("size/hash mapping: %+v", f)
	}
	if _, err := a.ListFolderItems(ctx, "p1", "fold:er@x", domain.PageRequest{}); err != nil {
		t.Fatalf("ID with sub-delimiters must be escaped once, not rejected or double-escaped: %v", err)
	}
	if _, err := a.ListFolderItems(ctx, "p2", "f1", domain.PageRequest{}); !errs.Is(err, errs.ResourceNotFound) {
		t.Fatalf("cross-project listing must be rejected, got %v", err)
	}
}

func TestFolderItemWithoutProjectFailsClosed(t *testing.T) {
	a, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"items":[{"id":"x","name":"n","type":"FILE"}],"links":{}}`))
	})
	if _, err := a.ListFolderItems(context.Background(), "p1", "f1", domain.PageRequest{}); !errs.Is(err, errs.UpstreamMalformed) {
		t.Fatalf("got %v", err)
	}
}

func TestFileMetadata(t *testing.T) {
	a, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tc/api/2.0/files/x1" {
			t.Errorf("path %s", r.URL.Path)
		}
		w.Write([]byte(`{"id":"x1","name":"model.ifc","type":"FILE","versionId":"v3","parentId":"f1","parentType":"FOLDER",
		  "createdOn":"2026-01-01T00:00:00Z","modifiedOn":"2026-03-01T12:00:00+02:00","size":2048,"projectId":"p1"}`))
	})
	ctx := context.Background()
	md, prov, err := a.GetFileMetadata(ctx, "p1", "x1")
	if err != nil {
		t.Fatal(err)
	}
	if *md.SizeBytes != 2048 || md.Revision != nil || md.Checksum != "" || md.ModifiedAt.Hour() != 10 || prov.Product != Product {
		t.Fatalf("metadata: %+v", md)
	}
	if _, _, err := a.GetFileMetadata(ctx, "other", "x1"); !errs.Is(err, errs.ResourceNotFound) {
		t.Fatalf("IDOR must be rejected, got %v", err)
	}
}

func TestRetryOn429ThenSuccess(t *testing.T) {
	var n atomic.Int32
	a, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`{"items":[],"links":{}}`))
	})
	if _, err := a.ListProjects(context.Background(), trimble.ListProjectsQuery{}); err != nil {
		t.Fatal(err)
	}
	if n.Load() != 2 {
		t.Fatalf("attempts = %d", n.Load())
	}
}

func TestNoRetryOn401AndBodyStaysPrivate(t *testing.T) {
	var n atomic.Int32
	a, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"secret upstream detail","code":"UNAUTHORIZED"}`))
	})
	_, err := a.ListProjects(context.Background(), trimble.ListProjectsQuery{})
	e := errs.As(err)
	if e.Code != errs.Authentication || n.Load() != 1 {
		t.Fatalf("code=%s attempts=%d", e.Code, n.Load())
	}
	if strings.Contains(e.Message, "secret upstream detail") {
		t.Fatal("upstream body leaked into safe message")
	}
}

func TestRetriesExhaustedOn503(t *testing.T) {
	var n atomic.Int32
	a, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	_, err := a.ListProjects(context.Background(), trimble.ListProjectsQuery{})
	if !errs.Is(err, errs.UpstreamUnavailable) || n.Load() != 3 {
		t.Fatalf("err=%v attempts=%d", err, n.Load())
	}
}

func TestRedirectNotFollowed(t *testing.T) {
	var hops atomic.Int32
	a, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		hops.Add(1)
		http.Redirect(w, r, "https://attacker.example/steal", http.StatusFound)
	})
	a.cfg.MaxAttempts = 1
	_, err := a.ListProjects(context.Background(), trimble.ListProjectsQuery{})
	if err == nil || hops.Load() != 1 {
		t.Fatalf("redirect must not be followed: err=%v hops=%d", err, hops.Load())
	}
}

func TestOversizedBodyRejected(t *testing.T) {
	a, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"items":[`))
		chunk := []byte(strings.Repeat(`{"id":"p","name":"n"},`, 1000))
		for i := 0; i < 500; i++ {
			w.Write(chunk)
		}
		w.Write([]byte(`{"id":"p"}],"links":{}}`))
	})
	if _, err := a.ListProjects(context.Background(), trimble.ListProjectsQuery{}); !errs.Is(err, errs.UpstreamMalformed) {
		t.Fatalf("got %v", err)
	}
}

func TestGetProject(t *testing.T) {
	a, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tc/api/2.0/projects/p1":
			w.Write([]byte(`{"id":"p1","name":"Bridge","description":"d","rootId":"f1","createdOn":"2026-01-01T00:00:00Z","modifiedOn":"2026-02-01T00:00:00Z","access":"FULL_ACCESS","location":"northAmerica"}`))
		case "/tc/api/2.0/projects/p2":
			w.Write([]byte(`{"id":"other","name":"x","rootId":"f9"}`))
		default:
			w.WriteHeader(404)
		}
	})
	ctx := context.Background()
	p, prov, err := a.GetProject(ctx, "p1")
	if err != nil || p.RootFolder != "f1" || p.Access != "FULL_ACCESS" || p.UpdatedAt == nil || prov.Source != "GET /2.0/projects/{projectId}" {
		t.Fatalf("%+v %v", p, err)
	}
	if _, _, err := a.GetProject(ctx, "missing"); !errs.Is(err, errs.ProjectNotFound) {
		t.Fatalf("404 must map to project_not_found: %v", err)
	}
	if _, _, err := a.GetProject(ctx, "p2"); !errs.Is(err, errs.UpstreamMalformed) {
		t.Fatalf("a response for another project must be rejected: %v", err)
	}
}

func TestDescriptor(t *testing.T) {
	a, _ := newTest(t, func(http.ResponseWriter, *http.Request) {})
	d := a.Describe()
	if !d.Supports(trimble.CapGetProject) || !d.ReadOnly || d.Status != trimble.Provisional || len(d.Sources) == 0 {
		t.Fatalf("descriptor: %+v", d)
	}
}
