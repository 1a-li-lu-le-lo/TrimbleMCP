package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/catalog"
)

// manifest is cmd/trimble-catalog/sources-other.json.
type manifest struct {
	Portal struct {
		Base     string   `json:"base"`
		Sections []string `json:"sections"`
	} `json:"portal"`
	Vista struct {
		Index  string `json:"index"`
		DocURL string `json:"doc_url"`
	} `json:"vista"`
	Identity struct {
		ID     string `json:"id"`
		URL    string `json:"url"`
		DocURL string `json:"doc_url"`
	} `json:"identity"`
	Rules  []*rule `json:"rules"`
	Safety []struct {
		MatchKey string `json:"match_key"`
		Reason   string `json:"reason"`
	} `json:"safety"`
}

// rule classifies discovered definitions. The first matching rule wins.
type rule struct {
	MatchSpec   string `json:"match_spec"`
	MatchDoc    string `json:"match_doc"`
	MatchSource string `json:"match_source"`
	MatchPrefix string `json:"match_prefix"`
	Class       string `json:"class"` // connect | reference | excluded | identity
	API         string `json:"api"`
	Family      string `json:"family"`
	Product     string `json:"product"`
	Auth        string `json:"auth"`
	Access      string `json:"access"`
	Requires    string `json:"requires"`
	Reason      string `json:"reason"`
	Note        string `json:"note"`
	// CoveredByAPI, on an excluded rule, marks operations that the named
	// reference API also documents as variants of it; only the rest are
	// excluded.
	CoveredByAPI string `json:"covered_by_api"`
	used         bool
}

func (r *rule) matches(d found) bool {
	switch {
	case r.MatchSpec != "":
		return strings.HasPrefix(d.SpecURL, r.MatchSpec)
	case r.MatchDoc != "":
		for _, u := range d.DocURLs {
			if strings.Contains(u, r.MatchDoc) {
				return true
			}
		}
		return false
	case r.MatchSource != "":
		return d.ID == r.MatchSource || strings.HasPrefix(d.ID, r.MatchSource+"/")
	case r.MatchPrefix != "":
		return strings.HasPrefix(d.ID, r.MatchPrefix)
	}
	return false
}

// found is one entry of discovered.json.
type found struct {
	ID      string   `json:"id"`
	SpecURL string   `json:"spec_url"`
	DocURLs []string `json:"doc_urls"`
	Kind    string   `json:"kind"`
	Error   string   `json:"error"`
	// Unavailable records a definition the manifest lists as known to fail
	// to download, with the reason; it is catalogued with no operations.
	Unavailable string `json:"unavailable"`
}

// identityEndpoints classifies every endpoint field of the Trimble Identity
// OpenID Connect discovery document. An endpoint field missing here fails
// the build.
var identityEndpoints = map[string]struct{ method, reason string }{
	"authorization_endpoint":               {"GET", "browser sign-in redirect; used only by `trimblectl auth login` (authorization code with PKCE S256). Agents never handle credentials"},
	"token_endpoint":                       {"POST", "issues and refreshes tokens; used only by internal/identity (Serial PKCE). Agents never handle credentials"},
	"revocation_endpoint":                  {"POST", "revokes tokens; used only by `trimblectl auth logout`"},
	"userinfo_endpoint":                    {"GET", "returns the signed-in user's personal profile; the bridge does not need it (data minimisation)"},
	"device_authorization_endpoint":        {"POST", "device-code grant; the bridge uses authorization code with PKCE instead"},
	"end_session_endpoint":                 {"GET", "browser sign-out redirect; not used by the bridge or agents"},
	"jwks_uri":                             {"GET", "public keys for validating Trimble-issued tokens; infrastructure, not a data API"},
	"mtls_endpoint_aliases.token_endpoint": {"POST", "mutual-TLS token endpoint for confidential clients; the bridge is a public client"},
}

// identityDocFields are discovery fields that are links to documents, not
// endpoints.
var identityDocFields = map[string]bool{"issuer": true, "service_documentation": true, "op_tos_uri": true, "op_policy_uri": true}

// addOther catalogues every non-Connect definition recorded by
// scripts/fetch-trimble-other-specs.py.
func addOther(cat *Catalog, dir, manifestPath string) {
	var man manifest
	mustJSON(manifestPath, &man)
	var list []found
	mustJSON(filepath.Join(dir, "discovered.json"), &list)
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })

	type group struct {
		rule    *rule
		name    string
		sources []*Source
		specs   []*spec
	}
	groups := map[string]*group{}
	var groupOrder []string
	var excluded []*Source
	var excludedSpecs []*spec

	for _, d := range list {
		if d.Error != "" {
			fail("retrieval of %s (%s) failed: %s; rerun scripts/fetch-trimble-other-specs.py", d.ID, d.SpecURL, d.Error)
		}
		var r *rule
		for _, x := range man.Rules {
			if x.matches(d) {
				r = x
				break
			}
		}
		if r == nil {
			fail("unclassified definition %s (%s): add a rule to %s after review", d.ID, d.SpecURL, manifestPath)
		}
		r.used = true
		if r.Class == "connect" {
			attachConnectDocs(cat, d)
			continue
		}
		if d.Unavailable != "" {
			// Listed in the manifest as known to fail; recorded so it is
			// accounted for, with no operations to catalogue.
			if r.Class != "excluded" {
				fail("%s is unavailable (%s) but its rule is %q; only excluded definitions may be unavailable", d.ID, d.Unavailable, r.Class)
			}
			cat.Sources = append(cat.Sources, Source{ID: d.ID, Title: d.ID, URL: d.SpecURL, DocURLs: d.DocURLs, Kind: d.Kind,
				Class: r.Class, Family: r.Family, Product: r.Product, Note: r.Reason, Unavailable: d.Unavailable})
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, strings.ReplaceAll(d.ID, "/", "__")+".json"))
		if err != nil {
			fail("read %s: %v", d.ID, err)
		}
		sp := &spec{id: d.ID}
		if err := json.Unmarshal(b, &sp.raw); err != nil {
			fail("parse %s: %v", d.ID, err)
		}
		sum := sha256.Sum256(b)
		title, version := sp.info()
		src := &Source{ID: d.ID, Title: title, Version: version, URL: d.SpecURL, DocURLs: d.DocURLs, Kind: d.Kind,
			SHA256: hex.EncodeToString(sum[:]), Class: r.Class, Family: r.Family, Product: r.Product, Note: firstNonEmpty(r.Reason, r.Note)}
		if d.Kind == "vista" {
			src.DocURLs = []string{man.Vista.DocURL} // per-operation pages are listed in the definition's x-doc-pages
		}
		switch r.Class {
		case "reference":
			api, name := apiName(r.API, d.ID)
			src.API = api
			g := groups[api]
			if g == nil {
				g = &group{rule: r, name: name}
				groups[api] = g
				groupOrder = append(groupOrder, api)
			}
			g.sources = append(g.sources, src)
			g.specs = append(g.specs, sp)
		case "excluded":
			if r.Reason == "" {
				fail("excluded rule for %s has no reason", d.ID)
			}
			src.VariantOf = r.CoveredByAPI
			excluded, excludedSpecs = append(excluded, src), append(excludedSpecs, sp)
		case "identity":
			addIdentity(cat, src, sp)
		default:
			fail("rule for %s has unknown class %q", d.ID, r.Class)
		}
	}
	for _, r := range man.Rules {
		if !r.used {
			fail("rule %+v in %s matches no discovered definition; review it", *r, manifestPath)
		}
	}

	for _, api := range groupOrder {
		g := groups[api]
		r := g.rule
		if r.Family == "" || r.Product == "" || r.Auth == "" || r.Access == "" || r.Requires == "" {
			fail("reference rule for %s must set family, product, auth, access and requires", api)
		}
		a := API{ID: api, Kind: catalog.KindReference, Title: r.Product, Family: r.Family, Product: r.Product,
			Source: g.sources[0].ID, Status: "reference", Auth: r.Auth, Access: r.Access, Requires: r.Requires}
		if strings.Contains(r.API, "{name}") {
			a.Title = r.Product + ": " + g.sources[0].Title
		}
		if len(g.sources[0].DocURLs) > 0 {
			a.DocURL = g.sources[0].DocURLs[0]
		}
		sec := map[string]bool{}
		srv := map[string]bool{}
		first := map[string]string{} // "METHOD path" -> key of the first definition declaring it
		for i, src := range g.sources {
			a.Sources = append(a.Sources, src.ID)
			for _, s := range g.specs[i].servers() {
				if !srv[s] {
					srv[s] = true
					a.Servers = append(a.Servers, s)
				}
			}
			for _, s := range g.specs[i].security() {
				sec[s] = true
			}
			ops := g.specs[i].operations()
			for _, o := range ops {
				o.API = api
				mp := o.Method + " " + o.Path
				if k, dup := first[mp]; dup {
					o.Key = src.ID + ":" + mp
					o.Disposition, o.CoveredBy = catalog.Variant, k
					o.Reason = "also published in " + src.ID + "; the same call as " + k
				} else {
					o.Key = api + ":" + mp
					first[mp] = o.Key
					o.Disposition, o.Reason = catalog.Reference, "reference only; the bridge never executes it (see the API's requirements)"
					if o.Method == "ANY" {
						o.Disposition, o.Reason = catalog.Excluded, "API Gateway catch-all (any method); not a documented operation"
					}
					if cred := credentialUse(o); cred != "" {
						o.Disposition, o.Reason = catalog.Excluded, "safety: "+cred+"; agents never handle credentials"
					}
				}
				cat.Operations = append(cat.Operations, o)
			}
			src.Ops = len(ops) + addWebhooks(cat, src, g.specs[i])
			cat.Sources = append(cat.Sources, *src)
		}
		for s := range sec {
			a.Security = append(a.Security, s)
		}
		sort.Strings(a.Security)
		if a.Security == nil {
			a.Security = []string{}
		}
		cat.APIs = append(cat.APIs, a)
	}

	// Safety exclusions: reference operations that could reach machinery,
	// vehicles or field positioning are never planned.
	for _, sr := range man.Safety {
		re, err := regexp.Compile(sr.MatchKey)
		if err != nil || !strings.HasPrefix(sr.Reason, "safety: ") {
			fail("safety rule %q: invalid pattern or reason (must start with \"safety: \")", sr.MatchKey)
		}
		n := 0
		for i := range cat.Operations {
			o := &cat.Operations[i]
			if o.Disposition == catalog.Reference && re.MatchString(o.Key) {
				o.Disposition, o.Reason = catalog.Excluded, sr.Reason
				n++
			}
		}
		if n == 0 {
			fail("safety rule %q matches no reference operation; review it", sr.MatchKey)
		}
	}

	keys := map[string]*Operation{}
	for i := range cat.Operations {
		keys[cat.Operations[i].Key] = &cat.Operations[i]
	}
	for i, src := range excluded {
		ops := excludedSpecs[i].operations()
		for _, o := range ops {
			o.Key = src.ID + ":" + o.Method + " " + o.Path
			o.Disposition, o.Reason = catalog.Excluded, src.Note
			if api := src.VariantOf; api != "" {
				if !hasAPI(cat, api) {
					fail("covered_by_api %q for %s is not a catalogued API", api, src.ID)
				}
				if k := api + ":" + o.Method + " " + o.Path; keys[k] != nil && keys[k].Disposition == catalog.Reference {
					o.API, o.Disposition, o.CoveredBy = api, catalog.Variant, k
					o.Reason = "also published in " + src.ID + " (" + src.Note + "); the same call as " + k
				}
			}
			cat.Operations = append(cat.Operations, o)
		}
		src.Ops = len(ops) + addWebhooks(cat, src, excludedSpecs[i])
		cat.Sources = append(cat.Sources, *src)
	}
}

// addWebhooks catalogues a definition's OpenAPI 3.1 webhooks as excluded:
// they are requests the provider sends to the customer's own endpoint, so
// there is nothing for anyone to call at the provider. Their keys carry a
// /webhook suffix on the source id.
func addWebhooks(cat *Catalog, src *Source, sp *spec) int {
	hooks := sp.webhooks()
	for _, o := range hooks {
		o.API = src.API
		o.Key = src.ID + "/webhook:" + o.Method + " " + o.Path
		o.Disposition = catalog.Excluded
		o.Reason = "webhook: the provider sends this request to the customer's endpoint; nothing to call at the provider"
		cat.Operations = append(cat.Operations, o)
	}
	return len(hooks)
}

// credentialUse explains why an operation needs a credential in its
// request (a credential-named body field, or a required credential
// parameter), or returns "".
func credentialUse(o Operation) string {
	if o.Method != "GET" && o.Method != "HEAD" && len(o.CredentialFields) > 0 {
		return "the request body carries credential fields (" + strings.Join(o.CredentialFields, ", ") + ")"
	}
	for _, p := range o.Params {
		if p.Required && catalog.IsCredentialName(p.Name) {
			return "the request requires the credential parameter " + p.Name
		}
	}
	return ""
}

func hasAPI(cat *Catalog, id string) bool {
	for _, a := range cat.APIs {
		if a.ID == id {
			return true
		}
	}
	return false
}

// apiName expands a rule's api template with the definition's name: the part
// of its id after the section, without a trailing -v1. A name equal to the
// template prefix collapses ("truckmate-{name}" + "truckmate" = "truckmate").
func apiName(tmpl, id string) (api, name string) {
	name = id
	if i := strings.LastIndex(id, "/"); i >= 0 {
		name = id[i+1:]
	}
	name = strings.TrimSuffix(name, "-v1")
	if !strings.Contains(tmpl, "{name}") {
		return tmpl, name
	}
	prefix := strings.TrimSuffix(tmpl, "-{name}")
	if name == prefix {
		return prefix, name
	}
	return strings.ReplaceAll(tmpl, "{name}", name), name
}

// attachConnectDocs records a Developer Portal page for a Trimble Connect
// definition, which must be one the SwaggerHub organisation marks production.
func attachConnectDocs(cat *Catalog, d found) {
	rest := strings.TrimPrefix(d.SpecURL, "https://api.swaggerhub.com/apis/Trimble-Connect/")
	slug := strings.Replace(strings.Trim(rest, "/"), "/", "@", 1)
	for i := range cat.Sources {
		s := &cat.Sources[i]
		if s.ID != slug {
			continue
		}
		if s.Class != "production" {
			fail("the Developer Portal links %s (%s), which sourceClass does not mark production", slug, d.ID)
		}
		for _, u := range d.DocURLs {
			if !contains(s.DocURLs, u) {
				s.DocURLs = append(s.DocURLs, u)
			}
		}
		sort.Strings(s.DocURLs)
		return
	}
	fail("the Developer Portal links %s (%s), which is not in the SwaggerHub organisation listing", d.SpecURL, d.ID)
}

// addIdentity catalogues the endpoints of the OpenID Connect discovery
// document; every one is excluded with the reason it is not agent-callable.
func addIdentity(cat *Catalog, src *Source, sp *spec) {
	type ep struct{ field, raw string }
	var eps []ep
	// Every absolute URL in the document, at the top level or one level down
	// (mtls_endpoint_aliases), is an endpoint unless it is a documentation
	// link; each must be classified in identityEndpoints.
	isURL := func(v any) (string, bool) {
		s, ok := v.(string)
		if !ok {
			return "", false
		}
		u, err := url.Parse(s)
		return s, err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != ""
	}
	for k, v := range sp.raw {
		if identityDocFields[k] {
			continue
		}
		if s, ok := isURL(v); ok {
			eps = append(eps, ep{k, s})
		}
		for ak, av := range obj(v) {
			if s, ok := isURL(av); ok {
				eps = append(eps, ep{k + "." + ak, s})
			}
		}
	}
	sort.Slice(eps, func(i, j int) bool { return eps[i].field < eps[j].field })
	add := func(key, method, path, summary, reason string) {
		cat.Operations = append(cat.Operations, Operation{Key: key, Source: src.ID, Method: method, Path: path,
			Summary: summary, Disposition: catalog.Excluded, Reason: reason})
		src.Ops++
	}
	u, err := url.Parse(src.URL)
	if err != nil {
		fail("identity discovery URL: %v", err)
	}
	add(src.ID+":GET "+u.Path, "GET", u.Path, "OpenID Connect discovery document", "discovery metadata; the source of this list, not a data API")
	host := u.Host
	for _, e := range eps {
		c, ok := identityEndpoints[e.field]
		if !ok {
			fail("Trimble Identity discovery declares %s (%s); classify it in identityEndpoints", e.field, e.raw)
		}
		eu, err := url.Parse(e.raw)
		if err != nil || eu.Scheme != "https" {
			fail("identity %s: %q is not an https URL", e.field, e.raw)
		}
		prefix := src.ID
		if eu.Host != host {
			prefix += "/" + strings.SplitN(eu.Host, ".", 2)[0] // e.g. trimble-identity/mtls
		}
		add(prefix+":"+c.method+" "+eu.Path, c.method, eu.Path, e.field+" ("+eu.Host+")", c.reason)
	}
	for f := range identityEndpoints {
		found := false
		for _, e := range eps {
			found = found || e.field == f
		}
		if !found {
			fail("identityEndpoints entry %s is no longer in the discovery document", f)
		}
	}
	title := src.Product
	src.Title, src.Version = title, "OpenID Connect discovery"
	cat.Sources = append(cat.Sources, *src)
}
