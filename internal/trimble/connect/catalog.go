package connect

import (
	"context"
	"net/url"
	"regexp"
	"strings"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/catalog"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/errs"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/trimble"
)

var pathParam = regexp.MustCompile(`\{([^{}]+)\}`)

// catalogEnv maps the adapter environment to catalogue host environments.
func (a *Adapter) catalogEnv() string {
	if a.cfg.Environment == Stage {
		return "stage"
	}
	return "production"
}

// CatalogBase implements trimble.CatalogClient: the documented base URL of
// api for this adapter's environment and region. APIs served from a single
// global host (support) use it for every region.
func (a *Adapter) CatalogBase(api string) (string, error) {
	if b, ok := a.cfg.CatalogBaseOverride[api]; ok && a.cfg.AllowInsecureOverride {
		return b, nil
	}
	ap, ok := catalog.APIByID(api)
	if !ok || ap.Kind != catalog.KindConnect {
		return "", errs.Newf(errs.UnsupportedCapability, "%q is not a Trimble Connect API", api)
	}
	env := a.catalogEnv()
	if b, ok := ap.BaseURL(env, string(a.cfg.Region)); ok {
		return b, nil
	}
	if b, ok := ap.BaseURL(env, "global"); ok {
		return b, nil
	}
	return "", errs.Newf(errs.UnsupportedCapability,
		"the %s API has no documented %s host for region %s", api, env, a.cfg.Region)
}

// CatalogURL builds the request URL for op with the given path parameters
// (each escaped as a single segment) and query values.
func (a *Adapter) CatalogURL(op *catalog.Operation, path map[string]string, query url.Values) (*url.URL, error) {
	base, err := a.CatalogBase(op.API)
	if err != nil {
		return nil, err
	}
	var missing []string
	p := pathParam.ReplaceAllStringFunc(op.RequestPath(), func(m string) string {
		name := m[1 : len(m)-1]
		v, ok := path[name]
		if !ok {
			missing = append(missing, name)
			return m
		}
		return url.PathEscape(v)
	})
	if len(missing) > 0 {
		return nil, errs.Newf(errs.Validation, "missing path parameter(s): %s", strings.Join(missing, ", "))
	}
	u, err := url.Parse(strings.TrimRight(base, "/") + p)
	if err != nil {
		return nil, errs.Wrap(errs.Internal, err)
	}
	b, _ := url.Parse(base)
	if u.Host != b.Host || u.Scheme != b.Scheme {
		return nil, errs.Newf(errs.Validation, "request would leave the documented host")
	}
	if query != nil {
		u.RawQuery = query.Encode()
	}
	return u, nil
}

// CatalogRead implements trimble.CatalogClient: a GET of a catalogued read
// operation. The caller (gateway) has already validated parameters against
// the catalogue and authorized the call.
func (a *Adapter) CatalogRead(ctx context.Context, op *catalog.Operation, path map[string]string, query url.Values, headers map[string]string) (trimble.CatalogResponse, error) {
	if op.Disposition != catalog.Read || op.Method != "GET" {
		return trimble.CatalogResponse{}, errs.Newf(errs.PolicyDenied, "operation %s is not an executable read", op.Key)
	}
	u, err := a.CatalogURL(op, path, query)
	if err != nil {
		return trimble.CatalogResponse{}, err
	}
	resp, err := a.fetch(ctx, u.String(), headers)
	if err != nil {
		return trimble.CatalogResponse{}, err
	}
	out := trimble.CatalogResponse{
		Status: resp.Status, ContentType: resp.ContentType, Body: resp.Body,
		Headers: map[string]string{},
		Prov:    a.prov(op.Method + " " + op.Path),
	}
	for _, h := range []string{"Content-Range", "ETag", "Last-Modified"} {
		if v := resp.Header.Get(h); v != "" {
			out.Headers[h] = v
		}
	}
	return out, nil
}
