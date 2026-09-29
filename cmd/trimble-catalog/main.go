// Command trimble-catalog regenerates the Trimble API operation catalogue
// (internal/catalog/catalog.json) and its reference documentation from the
// official Trimble Connect OpenAPI definitions published on SwaggerHub.
//
// Every operation in every retrieved definition receives exactly one
// disposition, so no endpoint is left unaccounted for:
//
//	read      production GET, executable through trimble_api_read
//	plan      production POST/PUT/PATCH/DELETE, dry-run through trimble_api_plan
//	variant   same method and path as a production operation (stage/int/qa/test copy)
//	excluded  not callable by the bridge, with the reason recorded
//
// Usage (network required to refresh the inputs):
//
//	make catalog
//	# or: go run ./cmd/trimble-catalog -specs DIR -index FILE -regions FILE
//
// The raw definitions are not committed; the catalogue records each source's
// URL and SHA-256 so a refresh can be diffed.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/catalog"
)

type (
	Catalog   = catalog.Catalog
	Source    = catalog.Source
	API       = catalog.API
	Param     = catalog.Param
	Operation = catalog.Operation
)

// sourceClass records the reviewed classification of every definition in the
// Trimble-Connect SwaggerHub organisation. A definition missing here fails the
// build, so a newly published API cannot slip through unclassified.
var sourceClass = map[string]struct{ class, of, api, note string }{
	"tcps@2.0":                  {"production", "", "core", ""},
	"model@v1":                  {"production", "", "model", ""},
	"model-feature-api@1.0":     {"production", "", "model-feature", ""},
	"org-prod@v1":               {"production", "", "org", ""},
	"pset-prod@v1":              {"production", "", "pset", ""},
	"topic@v2":                  {"production", "", "topics", ""},
	"topic-exchange-prod@v1":    {"production", "", "topic-exchange", ""},
	"issue@v1":                  {"production", "", "issues", ""},
	"support@v1":                {"production", "", "support", ""},
	"tdrive@1.0":                {"production", "", "drive", "Trimble Drive (beta)"},
	"fms@1.0":                   {"production", "", "file-service", "File Service (preview)"},
	"tcps-stage@2.0":            {"variant", "tcps@2.0", "", "staging copy"},
	"tcps-int@2.0":              {"variant", "tcps@2.0", "", "integration copy"},
	"tcps-qa@2.0":               {"variant", "tcps@2.0", "", "QA copy"},
	"files-int@1.0":             {"variant", "tcps@2.0", "", "integration copy of Core file-upload operations"},
	"model-stage@v1":            {"variant", "model@v1", "", "staging copy"},
	"ModelService@v1":           {"variant", "model@v1", "", "development build (model-api.dev host)"},
	"org-stage@v1":              {"variant", "org-prod@v1", "", "staging copy"},
	"org-stage-us-east-1@v1":    {"variant", "org-prod@v1", "", "staging copy"},
	"test-api-key-temp-2@v1":    {"variant", "org-prod@v1", "", "test copy published by Trimble"},
	"pset-stage@v1":             {"variant", "pset-prod@v1", "", "staging copy"},
	"topic-stage@v2":            {"variant", "topic@v2", "", "staging copy"},
	"topic-exchange-stage@v1":   {"variant", "topic-exchange-prod@v1", "", "staging copy"},
	"issue-stage@v1":            {"variant", "issue@v1", "", "staging copy"},
	"IssueMgmt@v0":              {"variant", "issue@v1", "", "earlier draft (v0)"},
	"IssueMgmt-Stage@1.0":       {"variant", "issue@v1", "", "staging draft (virtserver mock host)"},
	"support-stage@v1":          {"variant", "support@v1", "", "staging copy"},
	"tdrive-int@1.0":            {"variant", "tdrive@1.0", "", "integration copy"},
	"tdrive-stage@1.0":          {"empty", "", "", "published definition contains no paths"},
	"custom-attribute-stage@v1": {"internal", "", "", "staging-only definition served from localhost; no production host"},
	"tcps.internal@2.0":         {"internal", "", "", "Trimble-internal API ('TC Internal API'); not offered to integrators"},
	"tcps.internal-int@2.0":     {"internal", "", "", "Trimble-internal API (integration)"},
	"tcps.internal-qa@2.0":      {"internal", "", "", "Trimble-internal API (QA)"},
	"tcps.internal-stage@2.0":   {"internal", "", "", "Trimble-internal API (staging)"},
}

var apiMeta = map[string]struct{ title, status, doc string }{
	"core":           {"Trimble Connect Core API", "ga", "https://developer.trimble.com/docs/connect/core"},
	"model":          {"Trimble Connect Model API", "ga", "https://developer.trimble.com/docs/connect"},
	"model-feature":  {"Trimble Connect Model Feature Service", "ga", "https://developer.trimble.com/docs/connect"},
	"org":            {"Trimble Connect Org Service (Core Account / Organizer)", "ga", "https://developer.trimble.com/docs/connect"},
	"pset":           {"Trimble Connect Property Set Service", "ga", "https://developer.trimble.com/docs/connect"},
	"topics":         {"Trimble Connect Topics API (BCF 2.1 / 3.0)", "ga", "https://developer.trimble.com/docs/connect"},
	"topic-exchange": {"Trimble Connect Topics Exchange Service (BCF)", "ga", "https://developer.trimble.com/docs/connect"},
	"issues":         {"Trimble Connect Issue Management Service", "ga", "https://developer.trimble.com/docs/connect"},
	"support":        {"Trimble Connect Support Service", "ga", "https://developer.trimble.com/docs/connect"},
	"drive":          {"Trimble Drive", "beta", "https://api.swaggerhub.com/apis/Trimble-Connect/tdrive/1.0"},
	"file-service":   {"Trimble Connect File Service", "preview", "https://api.swaggerhub.com/apis/Trimble-Connect/fms/1.0"},
}

// excludedReads are production GET operations the bridge never executes.
var excludedReads = map[string]string{
	"core:GET /files/fs/{fileId}/downloadurl": "returns a presigned download URL for file content; content access is Level 1 and signed URLs must not be exposed to agents (see docs/security/threat-model.md)",
	"core:GET /shares/token/{stoken}":         "resolves a share token, which grants access outside project membership; agents must not handle share tokens",
}

// curatedTools maps catalogue operations to the typed tools that already
// wrap them with stronger validation.
var curatedTools = map[string]string{
	"core:GET /2.1/projects":                 "trimble_list_projects",
	"core:GET /projects/{projectId}":         "trimble_get_project",
	"core:GET /2.1/folders/{folderId}/items": "trimble_list_folder_items",
	"core:GET /files/{fileId}":               "trimble_get_file_metadata",
}

var (
	digitRegion = map[string]string{"": "us", "11": "us", "21": "eu", "22": "eu-gb", "31": "ap", "32": "ap-au"}
	awsRegion   = map[string]string{"us-east-1": "us", "eu-west-1": "eu", "eu-west-2": "eu-gb", "ap-southeast-1": "ap", "ap-southeast-2": "ap-au"}
	hostDigits  = regexp.MustCompile(`^(?:app|model-api|model-wf|open)(\d{2})?\.`)
	hostAWS     = regexp.MustCompile(`^(?:org|pset)-api\.([a-z]+-[a-z]+-\d)\.`)
	hostDrive   = regexp.MustCompile(`^(us|eu)\.(?:stage\.|int\.)?drive\.cde\.trimble\.com$`)
)

// classifyServer maps a server URL to (environment, region).
func classifyServer(raw string) (env, region string, ok bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || !strings.HasSuffix(u.Host, ".trimble.com") {
		return "", "", false
	}
	h := u.Host
	env = "production"
	switch {
	case strings.Contains(h, ".int."), strings.Contains(h, ".qa."), strings.Contains(h, ".dev."):
		return "", "", false
	case strings.Contains(h, ".stage."):
		env = "stage"
	}
	switch {
	case h == "support-api.connect.trimble.com":
		return env, "global", true
	case hostDigits.MatchString(h):
		return env, digitRegion[hostDigits.FindStringSubmatch(h)[1]], true
	case hostAWS.MatchString(h):
		r, ok := awsRegion[hostAWS.FindStringSubmatch(h)[1]]
		return env, r, ok
	case hostDrive.MatchString(h):
		return env, hostDrive.FindStringSubmatch(h)[1], true
	}
	return "", "", false
}

type spec struct {
	raw  map[string]any
	info struct{ title, version string }
}

func str(m map[string]any, k string) string {
	s, _ := m[k].(string)
	return s
}

func obj(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func (s *spec) resolve(v any) map[string]any {
	m := obj(v)
	for i := 0; i < 8 && m != nil; i++ {
		ref := str(m, "$ref")
		if ref == "" {
			return m
		}
		var cur any = s.raw
		for _, part := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
			part = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
			cur = obj(cur)[part]
		}
		m = obj(cur)
	}
	return m
}

func clip(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len([]rune(s)) > n {
		return string([]rune(s)[:n-1]) + "…"
	}
	return s
}

func (s *spec) params(pathItem, op map[string]any) []Param {
	seen := map[string]int{}
	var out []Param
	add := func(list any) {
		arr, _ := list.([]any)
		for _, p := range arr {
			pm := s.resolve(p)
			if pm == nil {
				continue
			}
			pr := Param{Name: str(pm, "name"), In: str(pm, "in"), Desc: clip(str(pm, "description"), 160)}
			pr.Required, _ = pm["required"].(bool)
			if pr.In == "path" {
				pr.Required = true
			}
			if sch := s.resolve(pm["schema"]); sch != nil {
				pr.Type, pr.Format = str(sch, "type"), str(sch, "format")
				if it := s.resolve(sch["items"]); pr.Type == "array" && it != nil {
					pr.Type = "array:" + str(it, "type")
				}
				if e, ok := sch["enum"].([]any); ok {
					for _, x := range e {
						pr.Enum = append(pr.Enum, fmt.Sprint(x))
					}
				}
			}
			k := pr.In + "\x00" + pr.Name
			if i, ok := seen[k]; ok {
				out[i] = pr // operation-level overrides path-level
				continue
			}
			seen[k] = len(out)
			out = append(out, pr)
		}
	}
	add(pathItem["parameters"])
	add(op["parameters"])
	sort.SliceStable(out, func(i, j int) bool {
		order := map[string]int{"path": 0, "query": 1, "header": 2, "cookie": 3}
		return order[out[i].In] < order[out[j].In]
	})
	return out
}

func (s *spec) body(op map[string]any) (types, req []string) {
	rb := s.resolve(op["requestBody"])
	if rb == nil {
		return nil, nil
	}
	for ct, v := range obj(rb["content"]) {
		types = append(types, ct)
		if sch := s.resolve(obj(v)["schema"]); sch != nil && req == nil {
			if r, ok := sch["required"].([]any); ok {
				for _, x := range r {
					req = append(req, fmt.Sprint(x))
				}
			}
		}
	}
	sort.Strings(types)
	sort.Strings(req)
	return types, req
}

var methods = []string{"get", "head", "post", "put", "patch", "delete", "options"}

func main() {
	specDir := flag.String("specs", "/tmp/specs", "directory of <slug>.json definitions")
	index := flag.String("index", "/tmp/sh-org.json", "SwaggerHub organisation listing JSON")
	regions := flag.String("regions", "/tmp/regions.json", "response of GET /tc/api/2.0/regions")
	out := flag.String("out", "internal/catalog/catalog.json", "catalogue output")
	docs := flag.String("docs", "docs/trimble-products/endpoints", "reference docs output directory")
	retrieved := flag.String("retrieved", "", "retrieval date (YYYY-MM-DD)")
	flag.Parse()
	if *retrieved == "" {
		fail("-retrieved is required")
	}

	var idx struct {
		APIs []struct {
			Name       string `json:"name"`
			Properties []struct {
				Type  string `json:"type"`
				URL   string `json:"url"`
				Value string `json:"value"`
			} `json:"properties"`
		} `json:"apis"`
	}
	mustJSON(*index, &idx)
	var regionList []map[string]any
	mustJSON(*regions, &regionList)

	cat := Catalog{Retrieved: *retrieved, Index: "https://api.swaggerhub.com/apis/Trimble-Connect"}
	specs := map[string]*spec{}
	srcURL := map[string]string{}
	for _, a := range idx.APIs {
		var u string
		for _, p := range a.Properties {
			if p.Type == "Swagger" {
				u = p.URL
			}
		}
		slug := strings.Replace(strings.SplitN(u, "/Trimble-Connect/", 2)[1], "/", "@", 1)
		if _, ok := sourceClass[slug]; !ok {
			fail("unclassified definition %s (%s): add it to sourceClass after review", slug, u)
		}
		b, err := os.ReadFile(filepath.Join(*specDir, slug+".json"))
		if err != nil {
			fail("read %s: %v", slug, err)
		}
		sp := &spec{}
		if err := json.Unmarshal(b, &sp.raw); err != nil {
			fail("parse %s: %v", slug, err)
		}
		sum := sha256.Sum256(b)
		info := obj(sp.raw["info"])
		c := sourceClass[slug]
		specs[slug] = sp
		srcURL[slug] = u
		cat.Sources = append(cat.Sources, Source{
			ID: slug, Title: str(info, "title"), Version: str(info, "version"), URL: u,
			SHA256: hex.EncodeToString(sum[:]), Class: c.class, VariantOf: c.of, API: c.api, Note: c.note,
		})
	}
	for slug := range sourceClass {
		if specs[slug] == nil {
			fail("classified definition %s is no longer published; review sourceClass", slug)
		}
	}
	sort.Slice(cat.Sources, func(i, j int) bool { return cat.Sources[i].ID < cat.Sources[j].ID })

	// Production APIs and their hosts.
	prodKey := map[string]map[string]string{} // source slug -> "METHOD path" -> key
	for _, src := range cat.Sources {
		if src.Class != "production" {
			continue
		}
		sp := specs[src.ID]
		m := apiMeta[src.API]
		a := API{ID: src.API, Title: m.title, Source: src.ID, Status: m.status, DocURL: m.doc, Hosts: map[string]map[string]string{}}
		for _, sv := range sp.raw["servers"].([]any) {
			raw := str(obj(sv), "url")
			env, region, ok := classifyServer(raw)
			if !ok {
				continue
			}
			if src.API == "core" && !strings.HasSuffix(strings.TrimRight(raw, "/"), "/tc/api") {
				continue // Core paths are relative to /tc/api (they carry /2.0 or /2.1 via prefix rule)
			}
			if a.Hosts[env] == nil {
				a.Hosts[env] = map[string]string{}
			}
			a.Hosts[env][region] = strings.TrimRight(raw, "/")
		}
		if len(a.Hosts["production"]) == 0 {
			fail("%s: no production host", src.ID)
		}
		for r := range a.Hosts["production"] {
			a.Regions = append(a.Regions, r)
		}
		sort.Strings(a.Regions)
		for name := range obj(obj(sp.raw["components"])["securitySchemes"]) {
			a.Security = append(a.Security, name)
		}
		sort.Strings(a.Security)
		cat.APIs = append(cat.APIs, a)
	}
	sort.Slice(cat.APIs, func(i, j int) bool { return cat.APIs[i].ID < cat.APIs[j].ID })
	crossCheckRegions(cat.APIs, regionList)

	apiOf := map[string]string{}
	for _, s := range cat.Sources {
		if s.Class == "production" {
			apiOf[s.ID] = s.API
		}
	}
	// Operations: production first so variants can reference them.
	order := append([]Source(nil), cat.Sources...)
	sort.SliceStable(order, func(i, j int) bool { return order[i].Class == "production" && order[j].Class != "production" })
	for si := range order {
		src := order[si]
		sp := specs[src.ID]
		paths := obj(sp.raw["paths"])
		pkeys := make([]string, 0, len(paths))
		for p := range paths {
			pkeys = append(pkeys, p)
		}
		sort.Strings(pkeys)
		n := 0
		for _, p := range pkeys {
			item := obj(paths[p])
			for _, m := range methods {
				opv, ok := item[m]
				if !ok {
					continue
				}
				op := obj(opv)
				n++
				M := strings.ToUpper(m)
				o := Operation{
					Source: src.ID, Method: M, Path: p, OperationID: str(op, "operationId"),
					Summary: clip(firstNonEmpty(str(op, "summary"), str(op, "description")), 200),
					Params:  sp.params(item, op),
				}
				o.Deprecated, _ = op["deprecated"].(bool)
				for _, t := range asSlice(op["tags"]) {
					o.Tags = append(o.Tags, fmt.Sprint(t))
				}
				o.BodyTypes, o.BodyReq = sp.body(op)
				switch src.Class {
				case "production":
					o.API = src.API
					o.Key = src.API + ":" + M + " " + p
					if prodKey[src.ID] == nil {
						prodKey[src.ID] = map[string]string{}
					}
					prodKey[src.ID][M+" "+p] = o.Key
					switch {
					case excludedReads[o.Key] != "":
						o.Disposition, o.Reason = "excluded", excludedReads[o.Key]
					case M == "HEAD":
						o.Disposition, o.Reason = "excluded", "HEAD returns no body; use the GET operation on the same path"
					case M == "GET":
						o.Disposition, o.Reason = "read", "production read; executable through trimble_api_read"
					case M == "OPTIONS":
						o.Disposition, o.Reason = "excluded", "CORS preflight; not an API operation"
					default:
						o.Disposition, o.Reason = "plan", "production change; dry-run plan through trimble_api_plan (execution requires the approval framework, ADR-0004)"
					}
					if t := curatedTools[o.Key]; t != "" {
						o.Tool = t
					}
				case "variant":
					o.API = apiOf[src.VariantOf]
					o.Key = src.ID + ":" + M + " " + p
					if k, ok := prodKey[src.VariantOf][M+" "+p]; ok {
						o.Disposition, o.CoveredBy = "variant", k
						o.Reason = fmt.Sprintf("%s of %s; same operation as %s", src.Note, src.VariantOf, k)
					} else {
						o.Disposition = "excluded"
						o.Reason = fmt.Sprintf("only in %s (%s); not published for production", src.ID, src.Note)
					}
				default:
					o.Key = src.ID + ":" + M + " " + p
					o.Disposition, o.Reason = "excluded", src.Note
				}
				cat.Operations = append(cat.Operations, o)
			}
		}
		for i := range cat.Sources {
			if cat.Sources[i].ID == src.ID {
				cat.Sources[i].Ops = n
			}
		}
	}
	for k := range excludedReads {
		if !hasKey(cat.Operations, k) {
			fail("excludedReads entry %s no longer exists", k)
		}
	}
	for k := range curatedTools {
		if !hasKey(cat.Operations, k) {
			fail("curatedTools entry %s no longer exists", k)
		}
	}
	sort.SliceStable(cat.Operations, func(i, j int) bool { return cat.Operations[i].Key < cat.Operations[j].Key })

	b, _ := json.MarshalIndent(cat, "", " ")
	if err := os.WriteFile(*out, append(b, '\n'), 0o644); err != nil {
		fail("%v", err)
	}
	writeDocs(*docs, &cat)
	fmt.Fprintf(os.Stderr, "catalogue: %d sources, %d APIs, %d operations\n", len(cat.Sources), len(cat.APIs), len(cat.Operations))
}

// crossCheckRegions verifies derived production hosts against the official
// /regions response where it names the same service.
func crossCheckRegions(apis []API, regions []map[string]any) {
	key := map[string]string{"core": "tc-api", "model": "model-api", "org": "org-api", "pset": "pset-api", "topics": "topic-api", "issues": "issues-api", "model-feature": "model-feature-api"}
	for _, a := range apis {
		k, ok := key[a.ID]
		if !ok {
			continue
		}
		for _, r := range regions {
			region := str(r, "region")
			want := strings.TrimRight(str(r, k), "/")
			got := a.Hosts["production"][region]
			if want == "" || got == "" {
				continue
			}
			wu, _ := url.Parse(want)
			gu, _ := url.Parse(got)
			if wu.Host != gu.Host {
				fail("%s region %s: spec host %s disagrees with /regions %s", a.ID, region, gu.Host, wu.Host)
			}
		}
	}
}

func hasKey(ops []Operation, k string) bool {
	for _, o := range ops {
		if o.Key == k {
			return true
		}
	}
	return false
}

func asSlice(v any) []any { s, _ := v.([]any); return s }

func firstNonEmpty(a ...string) string {
	for _, s := range a {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func mustJSON(path string, v any) {
	b, err := os.ReadFile(path)
	if err != nil {
		fail("%v", err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		fail("%s: %v", path, err)
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "trimble-catalog: "+format+"\n", args...)
	os.Exit(1)
}
