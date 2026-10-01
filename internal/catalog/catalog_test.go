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
		if !slices.Contains(Dispositions, o.Disposition) {
			t.Errorf("%s: invalid disposition %q", o.Key, o.Disposition)
		}
		if strings.TrimSpace(o.Reason) == "" {
			t.Errorf("%s: no reason", o.Key)
		}
		api, hasAPI := APIByID(o.API)
		switch o.Disposition {
		case Read:
			if o.Method != "GET" {
				t.Errorf("%s: only GET may be an executable read", o.Key)
			}
			if !hasAPI || api.Kind != KindConnect {
				t.Errorf("%s: executable reads must belong to a Trimble Connect API", o.Key)
			}
		case Plan:
			if o.Method == "GET" || o.Method == "HEAD" {
				t.Errorf("%s: reads must not be plan-only", o.Key)
			}
			if !hasAPI || api.Kind != KindConnect {
				t.Errorf("%s: plan operations must belong to a Trimble Connect API", o.Key)
			}
		case Reference:
			if !hasAPI || api.Kind != KindReference {
				t.Errorf("%s: reference operations must belong to a reference API", o.Key)
			}
		case Variant:
			p, ok := Lookup(o.CoveredBy)
			if !ok || p.Method != o.Method || p.Path != o.Path || p.Disposition == Variant || p.API != o.API {
				t.Errorf("%s: covered_by %q is not the matching operation", o.Key, o.CoveredBy)
			}
		}
		if hasAPI && api.Kind == KindReference && o.Disposition != Reference && o.Disposition != Variant &&
			!(o.Disposition == Excluded && (strings.HasPrefix(o.Reason, "safety: ") || strings.HasPrefix(o.Reason, "webhook: ") || o.Method == "ANY")) {
			t.Errorf("%s: an operation of reference API %s must be reference, variant or safety-excluded", o.Key, api.ID)
		}
	}
	for _, s := range c.Sources {
		if perSource[s.ID] != s.Ops {
			t.Errorf("source %s declares %d operations, catalogue has %d", s.ID, s.Ops, perSource[s.ID])
		}
		if !slices.Contains([]string{"production", "variant", "internal", "empty", "reference", "excluded", "identity"}, s.Class) {
			t.Errorf("source %s: class %q", s.ID, s.Class)
		}
		if !slices.Contains([]string{"swaggerhub", "portal", "maps", "app-xchange", "confluence", "direct", "doc", "vista", "oidc"}, s.Kind) {
			t.Errorf("source %s: kind %q", s.ID, s.Kind)
		}
		if (s.SHA256 == "") != (s.Unavailable != "") || !strings.HasPrefix(s.URL, "https://") {
			t.Errorf("source %s: provenance missing", s.ID)
		}
		if s.Kind == "swaggerhub" && !strings.HasPrefix(s.URL, "https://api.swaggerhub.com/apis/Trimble-Connect/") {
			t.Errorf("source %s: SwaggerHub source outside the Trimble-Connect organisation", s.ID)
		}
		if (s.Class == "excluded" || s.Class == "internal" || s.Class == "empty") && strings.TrimSpace(s.Note) == "" {
			t.Errorf("source %s: excluded without a reason", s.ID)
		}
	}
}

func TestProductionHostsAreDocumentedTrimbleHTTPS(t *testing.T) {
	for _, a := range Must().APIs {
		if a.Kind != KindConnect {
			continue
		}
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

// Reference APIs are never executable: no hosts, and they state what a
// deployment would need.
func TestReferenceAPIsAreNotExecutable(t *testing.T) {
	n := 0
	for _, a := range Must().APIs {
		if a.Family == "" || a.Product == "" {
			t.Errorf("%s: family and product are required", a.ID)
		}
		switch a.Kind {
		case KindConnect:
		case KindReference:
			n++
			if len(a.Hosts) > 0 || len(a.Regions) > 0 {
				t.Errorf("%s: reference API must not carry executable hosts", a.ID)
			}
			if _, ok := a.BaseURL("production", "us"); ok {
				t.Errorf("%s: BaseURL must refuse reference APIs", a.ID)
			}
			if a.Auth == "" || a.Access == "" || a.Requires == "" {
				t.Errorf("%s: auth, access and requires must be documented", a.ID)
			}
		default:
			t.Errorf("%s: kind %q", a.ID, a.Kind)
		}
	}
	if n == 0 {
		t.Fatal("no reference APIs: the non-Connect definitions were not catalogued")
	}
}

// Trimble Identity endpoints carry credentials: all are catalogued, none is
// callable.
func TestIdentityEndpointsExcluded(t *testing.T) {
	var n int
	for _, o := range Must().Operations {
		if strings.HasPrefix(o.Key, "trimble-identity") {
			n++
			if o.Disposition != Excluded {
				t.Errorf("%s must be excluded", o.Key)
			}
		}
	}
	if _, ok := Lookup("trimble-identity:POST /oauth/token"); !ok || n < 5 {
		t.Errorf("identity endpoints missing (%d found)", n)
	}
}

func TestServicesWithoutDefinitionsExplained(t *testing.T) {
	c := Must()
	if len(c.Undefined) == 0 {
		t.Fatal("expected the /regions services without definitions to be recorded")
	}
	for _, s := range c.Undefined {
		if s.Name == "" || s.Note == "" {
			t.Errorf("service %+v lacks a note", s)
		}
	}
}

// Operations that could reach machinery, vehicles or field positioning are
// never plannable.
func TestSafetyExclusions(t *testing.T) {
	for _, k := range []string{
		"ptx-farmengage:PUT /prescriptions/{orgId}/rx/{rxId}/vehicletarget/{vehicleId}",
		"ptx-farmengage:PUT /operations/{orgId}/workorders/{workOrderId}/vehicletarget/{vehicleId}",
		"mobile-manager:PUT /api/v1/correctionSource/",
		"ptx-farmengage:PATCH /resources/{orgId}/resourcefiles/{id}/send",
		"ptx-farmengage:POST /prescriptions/{orgId}/rx/importjob",
		"unity-construct:POST /api/v2/Authenticate",
		"truckmate:POST /login",
		"trimble-maps-routing-profile:PUT /routing/v1/routingprofiles/{routingProfileId}",
	} {
		o, ok := Lookup(k)
		if !ok || o.Disposition != Excluded || !strings.HasPrefix(o.Reason, "safety: ") {
			t.Errorf("%s must be safety-excluded, got %+v", k, o)
		}
	}
}

// Every {name} in a path template has exactly one declared path parameter,
// so every catalogued operation can be validated and planned.
func TestPathTemplatesMatchParameters(t *testing.T) {
	tmpl := regexp.MustCompile(`\{([^{}]+)\}`)
	for _, o := range Must().Operations {
		for _, m := range tmpl.FindAllStringSubmatch(o.Path, -1) {
			n := 0
			for _, p := range o.Params {
				if p.In == "path" && p.Name == m[1] {
					n++
				}
			}
			if n != 1 {
				t.Errorf("%s: path parameter %q declared %d times", o.Key, m[1], n)
			}
		}
	}
}

func TestPreferredContentType(t *testing.T) {
	for _, c := range []struct {
		in   []string
		want string
	}{
		{[]string{"application/*+json", "application/json", "text/json"}, "application/json"},
		{[]string{"application/*+json", "text/plain"}, "text/plain"},
		{nil, ""},
	} {
		if got := PreferredContentType(c.in); got != c.want {
			t.Errorf("%v: got %q", c.in, got)
		}
	}
}

func TestNoMutationIsExecutable(t *testing.T) {
	for _, o := range Must().Operations {
		if o.Disposition == Read && o.Method != "GET" {
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
		// Large APIs are split into one page per definition.
		subs, _ := filepath.Glob(filepath.Join(dir, a.ID+"--*.md"))
		for _, f := range subs {
			sb, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			doc += string(sb)
		}
		srcs := map[string]bool{a.Source: true}
		for _, s := range a.Sources {
			srcs[s] = true
		}
		for _, o := range c.Operations {
			if srcs[o.Source] && !strings.Contains(doc, "| `"+o.Method+"` | `"+o.Path+"` |") {
				t.Errorf("%s.md does not list %s", a.ID, o.Key)
			}
		}
	}
	nb, err := os.ReadFile(filepath.Join(dir, "non-production.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range c.Sources {
		if s.Class != "production" && s.Class != "reference" && !strings.Contains(string(nb), "## "+s.ID+" ") {
			t.Errorf("non-production.md does not account for %s", s.ID)
		}
	}
}

var keyShape = regexp.MustCompile(`^[A-Za-z0-9@._/-]+:(GET|HEAD|POST|PUT|PATCH|DELETE|OPTIONS|TRACE|ANY|SUBSCRIBE|PUBLISH) /`)

func TestKeyShape(t *testing.T) {
	for _, o := range Must().Operations {
		if !keyShape.MatchString(o.Key) {
			t.Errorf("malformed key %q", o.Key)
		}
	}
}

func TestIsCredentialName(t *testing.T) {
	for _, n := range []string{"password", "LoginPassword", "TinaPassword", "ntripPassword", "client_secret", "X-Api-Key",
		"apiKey", "access_token", "token", "GisToken", "mfaToken", "pwd", "Authorization", "privateKey"} {
		if !IsCredentialName(n) {
			t.Errorf("%s should be a credential name", n)
		}
	}
	for _, n := range []string{"skipToken", "continuationToken", "nextPageToken", "tokenType", "databaseToken", "page", "projectId", "keyword"} {
		if IsCredentialName(n) {
			t.Errorf("%s should not be a credential name", n)
		}
	}
}

// No plannable operation can need a credential: one that requires a
// credential parameter is excluded instead.
func TestNoPlannableOperationRequiresCredentials(t *testing.T) {
	for _, o := range Must().Operations {
		if o.Disposition != Plan && o.Disposition != Reference {
			continue
		}
		for _, p := range o.Params {
			if p.Required && IsCredentialName(p.Name) {
				t.Errorf("%s: required parameter %q is a credential, so the operation cannot be planned", o.Key, p.Name)
			}
		}
	}
}
