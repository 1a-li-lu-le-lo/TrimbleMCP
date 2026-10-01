package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/catalog"
)

// spec is one parsed OpenAPI 3.x or Swagger 2.0 definition.
type spec struct {
	id  string
	raw map[string]any
}

func str(m map[string]any, k string) string {
	s, _ := m[k].(string)
	return s
}

func obj(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asSlice(v any) []any { s, _ := v.([]any); return s }

func asStrings(v any) []string {
	var out []string
	for _, x := range asSlice(v) {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// typeOf returns a schema's type; OpenAPI 3.1 type arrays yield their first
// non-null member.
func typeOf(m map[string]any) string {
	switch t := m["type"].(type) {
	case string:
		return t
	case []any:
		for _, x := range t {
			if s, ok := x.(string); ok && s != "null" {
				return s
			}
		}
	}
	return ""
}

func (s *spec) swagger2() bool { return str(s.raw, "swagger") != "" }

func (s *spec) asyncAPI() bool { return str(s.raw, "asyncapi") != "" }

func (s *spec) info() (title, version string) {
	i := obj(s.raw["info"])
	return str(i, "title"), str(i, "version")
}

// servers returns the documented server URLs verbatim.
func (s *spec) servers() []string {
	if s.asyncAPI() {
		var names []string
		for n := range obj(s.raw["servers"]) {
			names = append(names, n)
		}
		sort.Strings(names)
		var out []string
		for _, n := range names {
			sv := obj(obj(s.raw["servers"])[n])
			u := str(sv, "url") // AsyncAPI 2
			if u == "" {
				u = str(sv, "protocol") + "://" + str(sv, "host") + str(sv, "pathname")
			}
			out = append(out, u)
		}
		return out
	}
	if s.swagger2() {
		host, base := str(s.raw, "host"), str(s.raw, "basePath")
		schemes := asStrings(s.raw["schemes"])
		if host == "" {
			if base == "" {
				return nil
			}
			return []string{base}
		}
		if len(schemes) == 0 {
			schemes = []string{"https"}
		}
		var out []string
		for _, sc := range schemes {
			out = append(out, sc+"://"+host+base)
		}
		return out
	}
	var out []string
	for _, sv := range asSlice(s.raw["servers"]) {
		if u := str(obj(sv), "url"); u != "" {
			out = append(out, u)
		}
	}
	return out
}

// security returns the names of the declared security schemes.
func (s *spec) security() []string {
	m := obj(obj(s.raw["components"])["securitySchemes"])
	if s.swagger2() {
		m = obj(s.raw["securityDefinitions"])
	}
	out := make([]string, 0, len(m))
	for name := range m {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
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

// params returns the path, query, header and cookie parameters. Swagger 2.0
// body and formData parameters describe the request body (see body).
func (s *spec) params(pathItem, op map[string]any) []Param {
	seen := map[string]int{}
	var out []Param
	add := func(list any) {
		for _, p := range asSlice(list) {
			pm := s.resolve(p)
			if pm == nil {
				continue
			}
			pr := Param{Name: str(pm, "name"), In: str(pm, "in"), Desc: clip(str(pm, "description"), 160)}
			if pr.In == "body" || pr.In == "formData" {
				continue
			}
			pr.Required, _ = pm["required"].(bool)
			if pr.In == "path" {
				pr.Required = true
			}
			sch := s.resolve(pm["schema"])
			if sch == nil && typeOf(pm) != "" {
				sch = pm // Swagger 2.0: the type is on the parameter itself
			}
			if sch != nil {
				pr.Type, pr.Format = typeOf(sch), str(sch, "format")
				enum := sch["enum"]
				if it := s.resolve(sch["items"]); pr.Type == "array" && it != nil {
					pr.Type = "array:" + typeOf(it)
					enum = it["enum"] // array values are checked item by item
				}
				for _, x := range asSlice(enum) {
					pr.Enum = append(pr.Enum, fmt.Sprint(x))
				}
				if pr.Type == "array" || strings.HasPrefix(pr.Type, "array:") {
					pr.Join = s.arrayJoin(pm)
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

// arrayJoin returns the delimiter for a non-exploded array parameter:
// OpenAPI 3 style form/spaceDelimited/pipeDelimited with explode:false
// (form explodes by default), or Swagger 2.0 collectionFormat (csv by
// default; multi repeats the parameter).
func (s *spec) arrayJoin(pm map[string]any) string {
	if s.swagger2() {
		switch f := str(pm, "collectionFormat"); f {
		case "", "csv":
			return ","
		case "ssv":
			return " "
		case "tsv":
			return "\t"
		case "pipes":
			return "|"
		case "multi":
			return ""
		default:
			fail("%s: parameter %s has unknown collectionFormat %q", s.id, str(pm, "name"), f)
		}
	}
	style := str(pm, "style")
	if style == "" {
		style = map[string]string{"query": "form", "cookie": "form", "path": "simple", "header": "simple"}[str(pm, "in")]
	}
	explode, set := pm["explode"].(bool)
	if !set {
		explode = style == "form"
	}
	if explode {
		return ""
	}
	switch style {
	case "form", "simple":
		return ","
	case "spaceDelimited":
		return " "
	case "pipeDelimited":
		return "|"
	}
	fail("%s: parameter %s has array style %q, which the bridge cannot serialise", s.id, str(pm, "name"), style)
	return ""
}

// body returns the request body's content types and required top-level
// fields, for OpenAPI 3 requestBody or Swagger 2.0 body/formData parameters.
func (s *spec) body(pathItem, op map[string]any) (types, req []string) {
	if rb := s.resolve(op["requestBody"]); rb != nil {
		content := obj(rb["content"])
		for ct := range content {
			types = append(types, ct)
		}
		sort.Strings(types)
		// Required fields come from the schema of the content type the plan
		// presents (application/json when offered; deterministic).
		if sch := s.resolve(obj(content[catalog.PreferredContentType(types)])["schema"]); sch != nil {
			req = asStrings(sch["required"])
		}
		sort.Strings(req)
		return types, req
	}
	if !s.swagger2() {
		return nil, nil
	}
	consumes := asStrings(op["consumes"])
	if len(consumes) == 0 {
		consumes = asStrings(s.raw["consumes"])
	}
	var hasBody, hasForm bool
	for _, list := range []any{pathItem["parameters"], op["parameters"]} {
		for _, p := range asSlice(list) {
			pm := s.resolve(p)
			switch str(pm, "in") {
			case "body":
				hasBody = true
				if sch := s.resolve(pm["schema"]); sch != nil {
					req = append(req, asStrings(sch["required"])...)
				}
			case "formData":
				hasForm = true
				if r, _ := pm["required"].(bool); r {
					req = append(req, str(pm, "name"))
				}
			}
		}
	}
	switch {
	case (hasBody || hasForm) && len(consumes) > 0:
		types = append(types, consumes...)
	case hasBody:
		types = []string{"application/json"}
	case hasForm:
		types = []string{"application/x-www-form-urlencoded"}
	}
	sort.Strings(types)
	sort.Strings(req)
	return types, req
}

var methods = []string{"get", "head", "post", "put", "patch", "delete", "options", "trace"}

// anyMethod is the API Gateway extension for a catch-all method; it is
// catalogued as method ANY so that it, too, has a disposition.
const anyMethod = "x-amazon-apigateway-any-method"

// pathItemKeys are the non-operation keys a path item may carry.
var pathItemKeys = map[string]bool{"summary": true, "description": true, "servers": true, "parameters": true}

// operations returns every operation in the definition, ordered by path and
// method. Any path-item key that is neither an operation nor a documented
// non-operation field fails the build, so no operation can be skipped.
func (s *spec) operations() []Operation {
	if s.asyncAPI() {
		return s.asyncOperations()
	}
	return s.pathOperations(obj(s.raw["paths"]))
}

// webhooks returns the OpenAPI 3.1 webhooks: requests the API provider
// sends to the customer's endpoint. Their path is /<webhook name>.
func (s *spec) webhooks() []Operation {
	hooks := map[string]any{}
	for name, item := range obj(s.raw["webhooks"]) {
		hooks["/"+name] = item
	}
	return s.pathOperations(hooks)
}

func (s *spec) pathOperations(paths map[string]any) []Operation {
	pkeys := make([]string, 0, len(paths))
	for p := range paths {
		pkeys = append(pkeys, p)
	}
	sort.Strings(pkeys)
	var out []Operation
	for _, p := range pkeys {
		raw := obj(paths[p])
		if ref, ok := raw["$ref"]; ok {
			if len(raw) > 1 {
				fail("%s %s: a path-item $ref with sibling keys is not supported", s.id, p)
			}
			if raw = s.resolve(raw); raw == nil {
				fail("%s %s: path-item $ref %v does not resolve", s.id, p, ref)
			}
		}
		item := raw
		for k := range item {
			if !pathItemKeys[k] && !strings.HasPrefix(k, "x-") && !contains(methods, k) {
				fail("%s %s: unexpected path-item key %q", s.id, p, k)
			}
		}
		for _, m := range append(append([]string{}, methods...), anyMethod) {
			opv, ok := item[m]
			if !ok {
				continue
			}
			op := obj(opv)
			if len(obj(op["callbacks"])) > 0 {
				fail("%s %s %s declares callbacks; classify them before regenerating", s.id, m, p)
			}
			method := strings.ToUpper(m)
			if m == anyMethod {
				method = "ANY"
			}
			o := Operation{
				Source: s.id, Method: method, Path: p, OperationID: str(op, "operationId"),
				Summary: clip(firstNonEmpty(str(op, "summary"), str(op, "description")), 200),
				Params:  reconcilePathParams(s.id, p, s.params(item, op)),
			}
			o.Deprecated, _ = op["deprecated"].(bool)
			for _, t := range asSlice(op["tags"]) {
				o.Tags = append(o.Tags, fmt.Sprint(t))
			}
			o.BodyTypes, o.BodyReq = s.body(item, op)
			out = append(out, o)
		}
	}
	return out
}

// asyncOperations catalogues an AsyncAPI definition's operations from the
// client's side: messages the application sends are SUBSCRIBE (the client
// receives them), messages it receives are PUBLISH. The path is the channel
// address. AsyncAPI 2.x (channel subscribe/publish) and 3.x (operations
// with action send/receive) are supported; anything else fails the build.
func (s *spec) asyncOperations() []Operation {
	var out []Operation
	add := func(method, channel string, op map[string]any) {
		out = append(out, Operation{Source: s.id, Method: method, Path: channel, OperationID: str(op, "operationId"),
			Summary: clip(firstNonEmpty(str(op, "summary"), str(op, "description"), str(op, "title")), 200)})
	}
	address := func(name string, ch map[string]any) string {
		a := str(ch, "address")
		if a == "" {
			a = name
		}
		if !strings.HasPrefix(a, "/") {
			a = "/" + a
		}
		return a
	}
	switch v := str(s.raw, "asyncapi"); {
	case strings.HasPrefix(v, "2."):
		for name, ch := range obj(s.raw["channels"]) {
			for k, method := range map[string]string{"subscribe": "SUBSCRIBE", "publish": "PUBLISH"} {
				if op := obj(obj(ch)[k]); op != nil {
					add(method, address(name, obj(ch)), op)
				}
			}
		}
	case strings.HasPrefix(v, "3."):
		for id, opv := range obj(s.raw["operations"]) {
			op := obj(opv)
			ref := str(obj(op["channel"]), "$ref")
			name := ref[strings.LastIndex(ref, "/")+1:]
			ch := s.resolve(op["channel"])
			if ch == nil {
				fail("%s: operation %s has an unresolvable channel", s.id, id)
			}
			method := map[string]string{"send": "SUBSCRIBE", "receive": "PUBLISH"}[str(op, "action")]
			if method == "" {
				fail("%s: operation %s has unknown action %q", s.id, id, str(op, "action"))
			}
			if str(op, "operationId") == "" {
				op = map[string]any{"operationId": id, "summary": str(op, "summary"), "description": str(op, "description"), "title": str(op, "title")}
			}
			add(method, address(name, ch), op)
		}
	default:
		fail("%s: unsupported AsyncAPI version %q", s.id, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path+out[i].Method < out[j].Path+out[j].Method })
	return out
}

var templateName = regexp.MustCompile(`\{([^{}]+)\}`)

// reconcilePathParams makes every {name} in the path template match exactly
// one declared path parameter. A declaration that differs only in case (an
// upstream typo, e.g. {PCOID} declared as pcoID) takes the template's
// spelling; a name with no declaration gets a required string parameter, so
// the operation stays usable and validated.
func reconcilePathParams(id, path string, ps []Param) []Param {
	for _, m := range templateName.FindAllStringSubmatch(path, -1) {
		name, found := m[1], false
		for i := range ps {
			if ps[i].In == "path" && ps[i].Name == name {
				found = true
			}
		}
		for i := range ps {
			if !found && ps[i].In == "path" && strings.EqualFold(ps[i].Name, name) {
				ps[i].Name, found = name, true
			}
		}
		if !found {
			ps = append([]Param{{Name: name, In: "path", Required: true, Type: "string", Desc: "declared only in the path template"}}, ps...)
		}
	}
	return ps
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func firstNonEmpty(a ...string) string {
	for _, s := range a {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func firstOr(s []string, def string) string {
	if len(s) > 0 {
		return s[0]
	}
	return def
}
