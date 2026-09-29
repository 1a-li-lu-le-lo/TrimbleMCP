// Package catalog embeds the generated catalogue of every operation in every
// official Trimble Connect API definition, with the disposition that decides
// how the bridge handles it (read, plan, variant, excluded). Regenerate with
// `make catalog`; see cmd/trimble-catalog.
package catalog

// Catalog is the generated artefact (catalog.json).
type Catalog struct {
	Retrieved  string      `json:"retrieved"`
	Index      string      `json:"index"`
	Sources    []Source    `json:"sources"`
	APIs       []API       `json:"apis"`
	Operations []Operation `json:"operations"`
}

type Source struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Version   string `json:"version"`
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	Class     string `json:"class"` // production | variant | internal | empty
	VariantOf string `json:"variant_of,omitempty"`
	API       string `json:"api,omitempty"`
	Note      string `json:"note,omitempty"`
	Ops       int    `json:"operations"`
}

type API struct {
	ID       string                       `json:"id"`
	Title    string                       `json:"title"`
	Source   string                       `json:"source"`
	Status   string                       `json:"status"` // ga | preview | beta
	DocURL   string                       `json:"doc_url"`
	Hosts    map[string]map[string]string `json:"hosts"` // environment -> region -> base URL
	Regions  []string                     `json:"regions"`
	Security []string                     `json:"security"`
}

type Param struct {
	Name     string   `json:"name"`
	In       string   `json:"in"`
	Required bool     `json:"required"`
	Type     string   `json:"type,omitempty"`
	Format   string   `json:"format,omitempty"`
	Enum     []string `json:"enum,omitempty"`
	Desc     string   `json:"description,omitempty"`
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
