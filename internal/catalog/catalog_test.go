package catalog

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// The completeness contract: no operation in any retrieved definition is
// unaccounted for.
func TestEveryOperationHasADisposition(t *testing.T) {
	c := Must()
	if len(c.Operations) == 0 || len(c.Sources) == 0 {
		t.Fatal("empty catalogue")
	}
	seen := map[string]bool{}
	perSource := map[string]int{}
	for _, o := range c.Operations {
		if seen[o.Key] {
			t.Errorf("duplicate key %s", o.Key)
		}
		seen[o.Key] = true
		perSource[o.Source]++
		if !slices.Contains([]string{Read, Plan, Variant, Excluded}, o.Disposition) {
			t.Errorf("%s: invalid disposition %q", o.Key, o.Disposition)
		}
		if strings.TrimSpace(o.Reason) == "" {
			t.Errorf("%s: no reason", o.Key)
		}
		switch o.Disposition {
		case Read:
			if o.Method != "GET" && o.Method != "HEAD" {
				t.Errorf("%s: only GET/HEAD may be executable reads", o.Key)
			}
		case Plan:
			if o.Method == "GET" || o.Method == "HEAD" {
				t.Errorf("%s: reads must not be plan-only", o.Key)
			}
		case Variant:
			p, ok := Lookup(o.CoveredBy)
			if !ok || p.Method != o.Method || p.Path != o.Path || p.Disposition == Variant {
				t.Errorf("%s: covered_by %q is not the matching production operation", o.Key, o.CoveredBy)
			}
		}
	}
	for _, s := range c.Sources {
		if perSource[s.ID] != s.Ops {
			t.Errorf("source %s declares %d operations, catalogue has %d", s.ID, s.Ops, perSource[s.ID])
		}
		if !slices.Contains([]string{"production", "variant", "internal", "empty"}, s.Class) {
			t.Errorf("source %s: class %q", s.ID, s.Class)
		}
		if s.SHA256 == "" || !strings.HasPrefix(s.URL, "https://api.swaggerhub.com/apis/Trimble-Connect/") {
			t.Errorf("source %s: provenance missing", s.ID)
		}
	}
}

func TestProductionHostsAreDocumentedTrimbleHTTPS(t *testing.T) {
	for _, a := range Must().APIs {
		if len(a.Hosts["production"]) == 0 {
			t.Errorf("%s: no production host", a.ID)
		}
		for env, hs := range a.Hosts {
			for region, h := range hs {
				u, err := url.Parse(h)
				if err != nil || u.Scheme != "https" || !strings.HasSuffix(u.Host, ".trimble.com") {
					t.Errorf("%s %s/%s: %q is not an https trimble.com host", a.ID, env, region, h)
				}
			}
		}
	}
}

func TestNoMutationIsExecutable(t *testing.T) {
	for _, o := range Must().Operations {
		if o.Disposition == Read && o.Method != "GET" && o.Method != "HEAD" {
			t.Fatalf("%s would execute a %s", o.Key, o.Method)
		}
	}
}

func TestSensitiveReadsExcluded(t *testing.T) {
	for _, k := range []string{"core:GET /files/fs/{fileId}/downloadurl", "core:GET /shares/token/{stoken}"} {
		o, ok := Lookup(k)
		if !ok || o.Disposition != Excluded {
			t.Errorf("%s must be excluded", k)
		}
	}
}

func TestCoreRequestPathRule(t *testing.T) {
	o, _ := Lookup("core:GET /projects/{projectId}")
	if o.RequestPath() != "/2.0/projects/{projectId}" {
		t.Fatal(o.RequestPath())
	}
	o, _ = Lookup("core:GET /2.1/projects")
	if o.RequestPath() != "/2.1/projects" {
		t.Fatal(o.RequestPath())
	}
}

// The generated reference docs must list every operation of their API.
func TestGeneratedDocsListEveryOperation(t *testing.T) {
	dir := "../../docs/trimble-products/endpoints"
	c := Must()
	for _, a := range c.APIs {
		b, err := os.ReadFile(filepath.Join(dir, a.ID+".md"))
		if err != nil {
			t.Fatal(err)
		}
		doc := string(b)
		for _, o := range c.Operations {
			if o.Source == a.Source && !strings.Contains(doc, "| `"+o.Method+"` | `"+o.Path+"` |") {
				t.Errorf("%s.md does not list %s", a.ID, o.Key)
			}
		}
	}
	nb, err := os.ReadFile(filepath.Join(dir, "non-production.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range c.Sources {
		if s.Class != "production" && !strings.Contains(string(nb), "## "+s.ID+" ") {
			t.Errorf("non-production.md does not account for %s", s.ID)
		}
	}
}

var keyShape = regexp.MustCompile(`^[A-Za-z0-9@._-]+:(GET|HEAD|POST|PUT|PATCH|DELETE|OPTIONS) /`)

func TestKeyShape(t *testing.T) {
	for _, o := range Must().Operations {
		if !keyShape.MatchString(o.Key) {
			t.Errorf("malformed key %q", o.Key)
		}
	}
}
