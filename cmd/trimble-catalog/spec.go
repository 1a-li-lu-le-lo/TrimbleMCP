package main

import (
	"fmt"
	"sort"
	"strings"
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

func (s *spec) info() (title, version string) {
	i := obj(s.raw["info"])
	return str(i, "title"), str(i, "version")
}

// servers returns the documented server URLs verbatim.
func (s *spec) servers() []string {
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
				if it := s.resolve(sch["items"]); pr.Type == "array" && it != nil {
					pr.Type = "array:" + typeOf(it)
				}
				for _, x := range asSlice(sch["enum"]) {
					pr.Enum = append(pr.Enum, fmt.Sprint(x))
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

// body returns the request body's content types and required top-level
// fields, for OpenAPI 3 requestBody or Swagger 2.0 body/formData parameters.
func (s *spec) body(pathItem, op map[string]any) (types, req []string) {
	if rb := s.resolve(op["requestBody"]); rb != nil {
		content := obj(rb["content"])
		for ct := range content {
			types = append(types, ct)
		}
		sort.Strings(types)
		// Required fields come from the JSON schema when there is one, else
		// from the first content type in sorted order (deterministic).
		pick := firstOr(types, "")
		if _, ok := content["application/json"]; ok {
			pick = "application/json"
		}
		if sch := s.resolve(obj(content[pick])["schema"]); sch != nil {
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

// pathItemKeys are the non-operation keys a path item may carry.
var pathItemKeys = map[string]bool{"summary": true, "description": true, "servers": true, "parameters": true}

// operations returns every operation in the definition, ordered by path and
// method. Any path-item key that is neither an operation nor a documented
// non-operation field fails the build, so no operation can be skipped.
func (s *spec) operations() []Operation {
	if len(asSlice(s.raw["webhooks"])) > 0 || len(obj(s.raw["webhooks"])) > 0 {
		fail("%s declares webhooks; classify them before regenerating", s.id)
	}
	paths := obj(s.raw["paths"])
	pkeys := make([]string, 0, len(paths))
	for p := range paths {
		pkeys = append(pkeys, p)
	}
	sort.Strings(pkeys)
	var out []Operation
	for _, p := range pkeys {
		item := s.resolve(paths[p])
		for k := range item {
			if !pathItemKeys[k] && !strings.HasPrefix(k, "x-") && !contains(methods, k) {
				fail("%s %s: unexpected path-item key %q", s.id, p, k)
			}
		}
		for _, m := range methods {
			opv, ok := item[m]
			if !ok {
				continue
			}
			op := obj(opv)
			if len(obj(op["callbacks"])) > 0 {
				fail("%s %s %s declares callbacks; classify them before regenerating", s.id, m, p)
			}
			o := Operation{
				Source: s.id, Method: strings.ToUpper(m), Path: p, OperationID: str(op, "operationId"),
				Summary: clip(firstNonEmpty(str(op, "summary"), str(op, "description")), 200),
				Params:  s.params(item, op),
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
