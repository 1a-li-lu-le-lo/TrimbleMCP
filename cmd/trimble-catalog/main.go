// Command trimble-catalog regenerates the Trimble API operation catalogue
// (internal/catalog/catalog.json.gz) and its reference documentation from:
//
//   - every definition in the official Trimble-Connect SwaggerHub
//     organisation (scripts/fetch-trimble-specs.sh);
//   - every definition linked from the Trimble Developer Portal, the
//     directly published product definitions, Vista's per-operation
//     fragments and the Trimble Identity discovery document
//     (scripts/fetch-trimble-other-specs.py, classified by sources-other.json).
//
// Every operation in every retrieved definition receives exactly one
// disposition, so no endpoint is left unaccounted for:
//
//	read       Trimble Connect production GET, executable through trimble_api_read
//	plan       Trimble Connect production change, dry-run through trimble_api_plan
//	reference  another Trimble product's documented operation; searchable and
//	           plannable, never executed (the bridge holds no credentials for it)
//	variant    the same call as the operation named in covered_by
//	excluded   not callable by the bridge, with the reason recorded
//
// A definition that matches no classification, a download that failed, or a
// classification that no longer matches anything fails the build.
//
// Usage (network required to refresh the inputs):
//
//	make catalog
//	# or: go run ./cmd/trimble-catalog -specs DIR -index FILE -regions FILE -other DIR -retrieved YYYY-MM-DD
//
// The raw definitions are not committed; the catalogue records each source's
// URL and SHA-256 so a refresh can be diffed.
package main

import (
	"bytes"
	"compress/gzip"
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
	"org":            {"Trimble Connect Org Service (Organizer)", "ga", "https://developer.trimble.com/docs/connect"},
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

func main() {
	specDir := flag.String("specs", "/tmp/specs", "directory of <slug>.json definitions")
	index := flag.String("index", "/tmp/sh-org.json", "SwaggerHub organisation listing JSON")
	regions := flag.String("regions", "/tmp/regions.json", "response of GET /tc/api/2.0/regions")
	out := flag.String("out", "internal/catalog/catalog.json.gz", "catalogue output (gzip-compressed JSON)")
	docs := flag.String("docs", "docs/trimble-products/endpoints", "reference docs output directory")
	other := flag.String("other", "/tmp/trimble-specs/other", "directory written by scripts/fetch-trimble-other-specs.py")
	manifest := flag.String("manifest", "cmd/trimble-catalog/sources-other.json", "classification of non-Connect definitions")
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

	cat := Catalog{Retrieved: *retrieved, Index: "https://api.swaggerhub.com/apis/Trimble-Connect", Portal: "https://developer.trimble.com"}
	specs := map[string]*spec{}
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
		sp := &spec{id: slug}
		if err := json.Unmarshal(b, &sp.raw); err != nil {
			fail("parse %s: %v", slug, err)
		}
		sum := sha256.Sum256(b)
		title, version := sp.info()
		c := sourceClass[slug]
		specs[slug] = sp
		src := Source{
			ID: slug, Title: title, Version: version, URL: u, Kind: "swaggerhub",
			SHA256: hex.EncodeToString(sum[:]), Class: c.class, VariantOf: c.of, API: c.api, Note: c.note,
		}
		if c.class == "production" {
			src.Family, src.Product = "connect", "Trimble Connect"
		}
		cat.Sources = append(cat.Sources, src)
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
		a := API{ID: src.API, Kind: catalog.KindConnect, Title: m.title, Family: "connect", Product: "Trimble Connect",
			Source: src.ID, Status: m.status, DocURL: m.doc, Hosts: map[string]map[string]string{}, Security: sp.security()}
		for _, raw := range sp.servers() {
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
		ops := specs[src.ID].operations()
		if len(specs[src.ID].webhooks()) > 0 {
			fail("%s declares webhooks; classify them before regenerating", src.ID)
		}
		for _, o := range ops {
			M, p := o.Method, o.Path
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
					o.Disposition, o.Reason = catalog.Excluded, excludedReads[o.Key]
				case M == "HEAD":
					o.Disposition, o.Reason = catalog.Excluded, "HEAD returns no body; use the GET operation on the same path"
				case M == "GET":
					o.Disposition, o.Reason = catalog.Read, "production read; executable through trimble_api_read"
				case M == "OPTIONS" || M == "TRACE":
					o.Disposition, o.Reason = catalog.Excluded, "CORS preflight or diagnostic method; not an API operation"
				case M == "ANY":
					o.Disposition, o.Reason = catalog.Excluded, "API Gateway catch-all (any method); not a documented operation"
				default:
					o.Disposition, o.Reason = catalog.Plan, "production change; dry-run plan through trimble_api_plan (execution requires the approval framework, ADR-0004)"
				}
				if t := curatedTools[o.Key]; t != "" {
					o.Tool = t
				}
			case "variant":
				o.API = apiOf[src.VariantOf]
				o.Key = src.ID + ":" + M + " " + p
				if k, ok := prodKey[src.VariantOf][M+" "+p]; ok {
					o.Disposition, o.CoveredBy = catalog.Variant, k
					o.Reason = fmt.Sprintf("%s of %s; same operation as %s", src.Note, src.VariantOf, k)
				} else {
					o.Disposition = catalog.Excluded
					o.Reason = fmt.Sprintf("only in %s (%s); not published for production", src.ID, src.Note)
				}
			default:
				o.Key = src.ID + ":" + M + " " + p
				o.Disposition, o.Reason = catalog.Excluded, src.Note
			}
			cat.Operations = append(cat.Operations, o)
		}
		for i := range cat.Sources {
			if cat.Sources[i].ID == src.ID {
				cat.Sources[i].Ops = len(ops)
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
	cat.Undefined = undefinedServices(regionList)
	addOther(&cat, *other, *manifest)
	seen := map[string]bool{}
	for _, o := range cat.Operations {
		if seen[o.Key] {
			fail("duplicate operation key %s", o.Key)
		}
		seen[o.Key] = true
	}
	sort.Slice(cat.Sources, func(i, j int) bool { return cat.Sources[i].ID < cat.Sources[j].ID })
	sort.Slice(cat.APIs, func(i, j int) bool {
		if cat.APIs[i].Kind != cat.APIs[j].Kind {
			return cat.APIs[i].Kind == catalog.KindConnect
		}
		return cat.APIs[i].ID < cat.APIs[j].ID
	})
	sort.SliceStable(cat.Operations, func(i, j int) bool { return cat.Operations[i].Key < cat.Operations[j].Key })

	writeCatalog(*out, &cat)
	writeDocs(*docs, &cat)
	byDisp := map[string]int{}
	for _, o := range cat.Operations {
		byDisp[o.Disposition]++
	}
	fmt.Fprintf(os.Stderr, "catalogue: %d sources, %d APIs, %d operations %v\n", len(cat.Sources), len(cat.APIs), len(cat.Operations), byDisp)
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

// regionServices maps the service keys of the /regions response to catalogue
// APIs, or explains why a service has no catalogued operations. A service key
// missing here fails the build.
var regionServices = map[string]struct{ api, note string }{
	"tc-api":            {"core", ""},
	"model-api":         {"model", ""},
	"model-feature-api": {"model-feature", ""},
	"org-api":           {"org", ""},
	"pset-api":          {"pset", ""},
	"topic-api":         {"topics", ""},
	"issues-api":        {"issues", ""},
	"projects-api":      {"", "listed by /regions. It serves a definition at /v1/api-docs that Trimble's documentation does not link (catalogued as the excluded source tc-project-service); its one documented endpoint (POST /v1/projects/update-users, Core Account) is catalogued as connect-projects-api"},
	"user-api":          {"", "listed by /regions. It serves a definition at /v1/api-docs that Trimble's documentation does not link (catalogued as the excluded source tc-user-app-service); user operations are in the Core API"},
	"batch-api":         {"", "listed by /regions. It serves a definition at /v1/api-docs that Trimble's documentation does not link (catalogued as the excluded source tc-batch-service)"},
	"objects-sync-api":  {"", "listed by /regions; no definition is retrievable (/v1/api-docs requires authentication). The Trimble Connect .NET SDK's sync helper is its client"},
	"wopi-api":          {"", "WOPI host (the protocol Microsoft Office for the web uses to open files); a protocol endpoint for Office, not an integrator API. Its /v1/api-docs definition is catalogued as the excluded source tc-wopi-service"},
}

// regionFields are the non-service fields of a /regions entry.
var regionFields = map[string]bool{"awsRegion": true, "isMaster": true, "location": true, "map-bbox": true, "origin": true, "region": true, "serviceRegion": true, "trnRegion": true}

func undefinedServices(regions []map[string]any) []catalog.Service {
	seen := map[string]bool{}
	var out []catalog.Service
	for _, r := range regions {
		for k := range r {
			if regionFields[k] || seen[k] {
				continue
			}
			seen[k] = true
			rs, ok := regionServices[k]
			if !ok {
				fail("/regions names service %q, which is not classified in regionServices", k)
			}
			if rs.api == "" {
				out = append(out, catalog.Service{Name: k, Note: rs.note})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// writeCatalog writes the catalogue as deterministic gzip-compressed JSON:
// indented JSON inside, so `zcat catalog.json.gz | diff` reviews well.
func writeCatalog(path string, cat *Catalog) {
	b, err := json.MarshalIndent(cat, "", " ")
	if err != nil {
		fail("%v", err)
	}
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression) // zero header: no name or mtime
	zw.Write(append(b, '\n'))
	if err := zw.Close(); err != nil {
		fail("%v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		fail("%v", err)
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
