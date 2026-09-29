package catalog

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

//go:embed catalog.json.gz
var raw []byte

// Dispositions.
const (
	Read      = "read"      // Trimble Connect production GET, executable
	Plan      = "plan"      // Trimble Connect production change, dry-run plan only
	Reference = "reference" // another product's documented operation: searchable and plannable, never executed
	Variant   = "variant"   // same call as the operation named in CoveredBy
	Excluded  = "excluded"  // never callable; Reason says why
)

// Dispositions lists every disposition.
var Dispositions = []string{Read, Plan, Reference, Variant, Excluded}

var (
	once   sync.Once
	loaded *Catalog
	byKey  map[string]*Operation
	apis   map[string]*API
	errLd  error
)

func load() {
	var c Catalog
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err == nil {
		err = json.NewDecoder(zr).Decode(&c)
	}
	if err != nil {
		errLd = fmt.Errorf("catalog.json.gz: %w", err)
		return
	}
	byKey = make(map[string]*Operation, len(c.Operations))
	for i := range c.Operations {
		byKey[c.Operations[i].Key] = &c.Operations[i]
	}
	apis = map[string]*API{}
	for i := range c.APIs {
		apis[c.APIs[i].ID] = &c.APIs[i]
	}
	loaded = &c
}

// Get returns the embedded catalogue.
func Get() (*Catalog, error) {
	once.Do(load)
	return loaded, errLd
}

// Must returns the catalogue or panics; the catalogue is validated by tests.
func Must() *Catalog {
	c, err := Get()
	if err != nil {
		panic(err)
	}
	return c
}

// Lookup returns the operation with key.
func Lookup(key string) (*Operation, bool) {
	once.Do(load)
	o, ok := byKey[key]
	return o, ok
}

// APIByID returns the API with id (Trimble Connect or reference).
func APIByID(id string) (*API, bool) {
	once.Do(load)
	a, ok := apis[id]
	return a, ok
}

// Query selects catalogue operations. Empty fields match everything; Text
// matches when every word appears in the key, summary, operationId or tags
// (case-insensitive).
type Query struct {
	API         string
	Family      string
	Disposition string
	Text        string
}

// Filter returns the operations matching q, ordered by key.
func Filter(q Query) []Operation {
	c := Must()
	words := strings.Fields(strings.ToLower(q.Text))
	var out []Operation
	for _, o := range c.Operations {
		if q.API != "" && o.API != q.API {
			continue
		}
		if q.Family != "" {
			a, ok := APIByID(o.API)
			if !ok || a.Family != q.Family {
				continue
			}
		}
		if q.Disposition != "" && o.Disposition != q.Disposition {
			continue
		}
		hay := strings.ToLower(o.Key + " " + o.Summary + " " + o.OperationID + " " + strings.Join(o.Tags, " "))
		match := true
		for _, w := range words {
			if !strings.Contains(hay, w) {
				match = false
				break
			}
		}
		if match {
			out = append(out, o)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// Families returns the distinct API families, sorted.
func Families() []string {
	seen := map[string]bool{}
	var out []string
	for _, a := range Must().APIs {
		if !seen[a.Family] {
			seen[a.Family] = true
			out = append(out, a.Family)
		}
	}
	sort.Strings(out)
	return out
}

// BaseURL returns the base for api in env and region. Only Trimble Connect
// APIs have executable hosts; reference APIs never do.
func (a *API) BaseURL(env, region string) (string, bool) {
	if a.Kind != KindConnect {
		return "", false
	}
	b, ok := a.Hosts[env][region]
	return b, ok
}

// RequestPath returns the path to append to the API base for o, applying
// the Core API version-prefix rule.
func (o *Operation) RequestPath() string {
	if o.API == "core" && !strings.HasPrefix(o.Path, "/2.") {
		return "/2.0" + o.Path
	}
	return o.Path
}
