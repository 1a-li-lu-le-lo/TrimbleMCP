// Package catalog embeds the generated catalogue of every operation in every
// Trimble API definition the bridge could find: the official Trimble Connect
// definitions (SwaggerHub organisation), every definition linked from the
// Trimble Developer Portal, directly published product definitions, and the
// Trimble Identity discovery document. Each operation carries the
// disposition that decides how the bridge handles it (read, plan, reference,
// variant, excluded). Regenerate with `make catalog`; see cmd/trimble-catalog.
package catalog

// Catalog is the generated artefact (catalog.json.gz).
type Catalog struct {
	Retrieved  string      `json:"retrieved"`
	Index      string      `json:"index"`
	Portal     string      `json:"portal"`
	Sources    []Source    `json:"sources"`
	APIs       []API       `json:"apis"`
	Operations []Operation `json:"operations"`
	// Services named by the Trimble Connect /regions response that publish
	// no API definition, so they have no catalogued operations.
	Undefined []Service `json:"services_without_definitions"`
}

// Source is one retrieved API definition.
type Source struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Version   string   `json:"version"`
	URL       string   `json:"url"`
	DocURLs   []string `json:"doc_urls,omitempty"`
	Kind      string   `json:"kind"` // swaggerhub | portal | maps | app-xchange | confluence | direct | doc | vista | oidc
	SHA256    string   `json:"sha256"`
	Class     string   `json:"class"` // production | variant | internal | empty | reference | excluded | identity
	VariantOf string   `json:"variant_of,omitempty"`
	API       string   `json:"api,omitempty"`
	Family    string   `json:"family,omitempty"`
	Product   string   `json:"product,omitempty"`
	Note      string   `json:"note,omitempty"`
	// Unavailable explains why a listed definition could not be retrieved;
	// such a source has no SHA-256 and no operations.
	Unavailable string `json:"unavailable,omitempty"`
	Ops         int    `json:"operations"`
}

// API kinds.
const (
	KindConnect   = "connect"   // executable reads against documented Trimble Connect hosts
	KindReference = "reference" // documented and plannable, never executed by the bridge
)

type API struct {
	ID       string                       `json:"id"`
	Kind     string                       `json:"kind"`
	Title    string                       `json:"title"`
	Family   string                       `json:"family"`
	Product  string                       `json:"product"`
	Source   string                       `json:"source"`
	Sources  []string                     `json:"sources,omitempty"`
	Status   string                       `json:"status"` // ga | preview | beta | reference
	DocURL   string                       `json:"doc_url"`
	Hosts    map[string]map[string]string `json:"hosts,omitempty"` // environment -> region -> base URL (connect only)
	Regions  []string                     `json:"regions,omitempty"`
	Servers  []string                     `json:"servers,omitempty"` // documented server URLs, verbatim (reference only)
	Security []string                     `json:"security"`
	Auth     string                       `json:"auth,omitempty"`
	Access   string                       `json:"access,omitempty"`
	Requires string                       `json:"requires,omitempty"`
}

// Service is a Trimble Connect regional service without a published
// definition.
type Service struct {
	Name string `json:"name"`
	Note string `json:"note"`
}

type Param struct {
	Name     string   `json:"name"`
	In       string   `json:"in"`
	Required bool     `json:"required"`
	Type     string   `json:"type,omitempty"`
	Format   string   `json:"format,omitempty"`
	Enum     []string `json:"enum,omitempty"` // for arrays, the allowed item values
	// Join is the delimiter that serialises an array parameter as one value
	// (OpenAPI explode:false styles, Swagger 2.0 collectionFormat); empty
	// means one repeated parameter per value.
	Join string `json:"join,omitempty"`
	Desc string `json:"description,omitempty"`
}

type Operation struct {
	Key         string   `json:"key"`
	API         string   `json:"api"`
	Source      string   `json:"source"`
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	OperationID string   `json:"operation_id,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Params      []Param  `json:"params,omitempty"`
	BodyTypes   []string `json:"body_content_types,omitempty"`
	BodyReq     []string `json:"body_required,omitempty"`
	Deprecated  bool     `json:"deprecated,omitempty"`
	Disposition string   `json:"disposition"`
	Reason      string   `json:"reason"`
	CoveredBy   string   `json:"covered_by,omitempty"`
	Tool        string   `json:"tool,omitempty"`
}
