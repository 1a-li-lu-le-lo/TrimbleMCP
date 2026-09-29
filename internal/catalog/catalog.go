package catalog

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

//go:embed catalog.json
var raw []byte

// Dispositions.
const (
	Read     = "read"
	Plan     = "plan"
	Variant  = "variant"
	Excluded = "excluded"
)

var (
	once   sync.Once
	loaded *Catalog
	byKey  map[string]*Operation
	apis   map[string]*API
	errLd  error
)

func load() {
	var c Catalog
	if err := json.Unmarshal(raw, &c); err != nil {
		errLd = fmt.Errorf("catalog.json: %w", err)
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

// APIByID returns the production API with id.
func APIByID(id string) (*API, bool) {
	once.Do(load)
	a, ok := apis[id]
	return a, ok
}

// Filter returns operations of an API (or all when api is empty) with the
// given disposition (or any when empty) whose key, summary, operationId or
// tags contain every word of query (case-insensitive).
func Filter(api, disposition, query string) []Operation {
	c := Must()
	words := strings.Fields(strings.ToLower(query))
	var out []Operation
	for _, o := range c.Operations {
		if api != "" && o.API != api {
			continue
		}
		if disposition != "" && o.Disposition != disposition {
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

// BaseURL returns the base for api in env and region. Core paths that do
// not start with a version segment are served under /tc/api/2.0.
func (a *API) BaseURL(env, region string) (string, bool) {
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
