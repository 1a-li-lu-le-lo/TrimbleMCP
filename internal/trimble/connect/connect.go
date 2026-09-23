// Package connect is the read-only adapter for the Trimble Connect REST API
// (Core API, "Trimble Connect API" 2.0/2.1).
//
// Every endpoint and field used here is traced to the official OpenAPI
// definition and developer guides listed in Sources. Fields that could not be
// verified are not read. See docs/trimble-products/trimble-connect.md.
package connect

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/domain"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/errs"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/ratelimit"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/trimble"
)

// Product is the bridge's identifier for Trimble Connect.
const Product domain.Product = "trimble-connect"

// Sources are the official documents this adapter is implemented from.
var Sources = []string{
	"https://developer.trimble.com/docs/connect",
	"https://developer.trimble.com/docs/connect/guides/access",
	"https://developer.trimble.com/docs/connect/reference",
	"https://api.swaggerhub.com/apis/Trimble-Connect/tcps/2.0",
	"https://app.connect.trimble.com/tc/api/2.0/regions",
	"https://id.trimble.com/.well-known/openid-configuration",
}

// LastVerified is the date the Sources were last checked.
const LastVerified = "2026-09-23"

// Environment selects production or Trimble's staging servers.
type Environment string

const (
	Stage      Environment = "stage"
	Production Environment = "production"
)

// hosts maps (environment, region) to the documented regional host. Region
// keys are the `region` values returned by /regions. Staging hosts are listed
// in the OpenAPI `servers` block; only the three documented there are allowed.
var hosts = map[Environment]map[domain.Region]string{
	Production: {
		"us":    "app.connect.trimble.com",
		"eu":    "app21.connect.trimble.com",
		"eu-gb": "app22.connect.trimble.com",
		"ap":    "app31.connect.trimble.com",
		"ap-au": "app32.connect.trimble.com",
	},
	Stage: {
		"us": "app.stage.connect.trimble.com",
		"eu": "app21.stage.connect.trimble.com",
		"ap": "app31.stage.connect.trimble.com",
	},
}

// BaseURL returns the documented API base (".../tc/api") for env and region.
func BaseURL(env Environment, region domain.Region) (string, error) {
	h, ok := hosts[env][region]
	if !ok {
		return "", errs.Newf(errs.Configuration, "no documented Trimble Connect host for environment %q region %q", env, region)
	}
	return "https://" + h + "/tc/api", nil
}

// TokenSource supplies a Trimble Identity access token for the tenant this
// adapter instance is bound to. Implementations never log tokens.
type TokenSource interface {
	Token(ctx context.Context) (string, error)
}

// Config configures one adapter instance (one tenant, one region).
type Config struct {
	Environment Environment
	Region      domain.Region
	// BaseURLOverride is for contract tests against a local mock only. It is
	// rejected unless AllowInsecureOverride is set.
	BaseURLOverride       string
	AllowInsecureOverride bool
	Tokens                TokenSource
	HTTPClient            *http.Client
	Timeout               time.Duration
	RatePerSecond         float64
	Burst                 int
	MaxAttempts           int
	UserAgent             string
}

// Adapter implements trimble.ProjectReader (list only) and trimble.FileReader.
type Adapter struct {
	cfg     Config
	base    *url.URL
	client  *http.Client
	limiter *ratelimit.Keyed
	now     func() time.Time
	sleep   func(context.Context, time.Duration) error
}

// maxBody bounds upstream response bodies read into memory.
const maxBody = 8 << 20

// New validates cfg and returns an adapter.
func New(cfg Config) (*Adapter, error) {
	if cfg.Tokens == nil {
		return nil, errs.Newf(errs.Configuration, "trimble-connect requires a token source")
	}
	raw := cfg.BaseURLOverride
	if raw != "" && !cfg.AllowInsecureOverride {
		return nil, errs.Newf(errs.Configuration, "base URL override is only permitted in tests")
	}
	if raw == "" {
		var err error
		if raw, err = BaseURL(cfg.Environment, cfg.Region); err != nil {
			return nil, err
		}
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return nil, errs.Newf(errs.Configuration, "invalid base URL")
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.RatePerSecond == 0 {
		// Trimble does not publish Connect rate limits; stay conservative.
		cfg.RatePerSecond = 5
	}
	if cfg.Burst == 0 {
		cfg.Burst = 10
	}
	if cfg.MaxAttempts == 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "trimble-mcp-bridge/0.1"
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{
			Timeout: cfg.Timeout,
			// Never follow redirects with the bearer token attached to a
			// different host.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		}
	}
	return &Adapter{
		cfg: cfg, base: u, client: client,
		limiter: ratelimit.NewKeyed(cfg.RatePerSecond, cfg.Burst),
		now:     func() time.Time { return time.Now().UTC() },
		sleep:   sleepCtx,
	}, nil
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// Describe implements trimble.Base.
func (a *Adapter) Describe() trimble.Descriptor {
	return trimble.Descriptor{
		Product:      Product,
		OfficialName: "Trimble Connect REST API (Core)",
		APIVersion:   "2.0 / 2.1",
		BaseURL:      a.base.String(),
		Auth:         "Trimble Identity OAuth 2.0 authorization code with PKCE (client credentials not supported by Connect)",
		Scopes:       []string{"openid", "<application scope from Trimble Developer Console>"},
		TenantModel:  "per Trimble Identity user; projects are bound to one region and resources are independent per region",
		Capabilities: []trimble.Capability{trimble.CapListProjects, trimble.CapListFolder, trimble.CapFileMetadata},
		Pagination:   "v2.1 pageSize + skipToken; next page signalled by links.next",
		RateLimits:   "not published by Trimble; client-side limit " + strconv.FormatFloat(a.cfg.RatePerSecond, 'f', -1, 64) + " req/s",
		Idempotency:  "read-only adapter; GET only",
		Webhooks:     "none documented for the REST API",
		Retries:      "429/500/503 retried; 502/504 retried for GET; Retry-After honoured",
		TimeoutText:  a.cfg.Timeout.String(),
		Units:        "file sizes in bytes as reported upstream",
		CRS:          "not applicable to the implemented operations",
		Region:       a.cfg.Region,
		DataClass:    "customer project metadata (confidential)",
		Owner:        "trimble-mcp-bridge maintainers",
		Sources:      Sources,
		LastVerified: LastVerified,
		// GET /projects/{id} and /users/me could not be verified in the
		// published spec and are intentionally not implemented.
		Status:        trimble.Provisional,
		DisableSwitch: "TRIMBLE_CONNECT_ENABLED=false",
		ReadOnly:      true,
	}
}

// Health implements trimble.Base. It performs no upstream call because no
// documented unauthenticated health endpoint exists; it reports configuration
// health only.
func (a *Adapter) Health(context.Context) trimble.Health {
	return trimble.Health{Status: "unknown", CheckedAt: a.now(), Detail: "no documented health endpoint; configuration valid"}
}

// ---- wire types (private; only verified fields) ----

type wireLinks struct {
	Next *struct {
		Href string `json:"href"`
	} `json:"next"`
}

type wireProject struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	RootID    string `json:"rootId"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type wireProjectList struct {
	Items *[]wireProject `json:"items"`
	Links wireLinks      `json:"links"`
}

type wireItem struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	VersionID  string `json:"versionId"`
	ParentID   string `json:"parentId"`
	ModifiedOn string `json:"modifiedOn"`
	ProjectID  string `json:"projectId"`
}

type wireItemList struct {
	Items *[]wireItem `json:"items"`
	Links wireLinks   `json:"links"`
}

type wireFile struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	VersionID  string `json:"versionId"`
	ParentID   string `json:"parentId"`
	CreatedOn  string `json:"createdOn"`
	ModifiedOn string `json:"modifiedOn"`
	Size       *int64 `json:"size"`
	ProjectID  string `json:"projectId"`
	Revision   *int   `json:"revision"`
	Hash       string `json:"hash"`
}

// ---- operations ----

// ListProjects implements trimble.ProjectReader via GET /2.1/projects.
func (a *Adapter) ListProjects(ctx context.Context, q trimble.ListProjectsQuery) (trimble.ProjectPage, error) {
	page, err := q.Page.Normalize()
	if err != nil {
		return trimble.ProjectPage{}, err
	}
	skip, err := decodeToken(page.Token)
	if err != nil {
		return trimble.ProjectPage{}, err
	}
	v := url.Values{"pageSize": {strconv.Itoa(page.Size)}}
	if skip != "" {
		v.Set("skipToken", skip)
	}
	var body wireProjectList
	if err := a.get(ctx, []string{"2.1", "projects"}, v, &body); err != nil {
		return trimble.ProjectPage{}, err
	}
	if body.Items == nil {
		return trimble.ProjectPage{}, errs.Newf(errs.UpstreamMalformed, "project list response has no items array")
	}
	out := make([]trimble.Project, 0, len(*body.Items))
	for _, w := range *body.Items {
		if _, err := domain.ParseProjectID(w.ID); err != nil {
			return trimble.ProjectPage{}, errs.Wrap(errs.UpstreamMalformed, err)
		}
		p := trimble.Project{
			ID: domain.ProjectID(w.ID), Name: w.Name, Region: a.cfg.Region,
			CreatedAt: parseTime(w.CreatedAt), UpdatedAt: parseTime(w.UpdatedAt),
		}
		if w.RootID != "" {
			if _, err := domain.ParseFolderID(w.RootID); err != nil {
				return trimble.ProjectPage{}, errs.Wrap(errs.UpstreamMalformed, err)
			}
			p.RootFolder = domain.FolderID(w.RootID)
		}
		out = append(out, p)
	}
	info, err := a.pageInfo(body.Links, skip, page.Size, len(out))
	if err != nil {
		return trimble.ProjectPage{}, err
	}
	return trimble.ProjectPage{Projects: out, Page: info, Prov: a.prov("GET /2.1/projects")}, nil
}

// GetProject is not implemented: the single-project endpoint could not be
// verified in the published specification. The capability is not declared,
// so the gateway never routes here.
func (a *Adapter) GetProject(context.Context, domain.ProjectID) (trimble.Project, domain.Provenance, error) {
	return trimble.Project{}, domain.Provenance{}, errs.New(errs.UnsupportedCapability)
}

// ListFolderItems implements trimble.FileReader via GET /2.1/folders/{id}/items.
func (a *Adapter) ListFolderItems(ctx context.Context, project domain.ProjectID, folder domain.FolderID, pr domain.PageRequest) (trimble.ItemPage, error) {
	page, err := pr.Normalize()
	if err != nil {
		return trimble.ItemPage{}, err
	}
	skip, err := decodeToken(page.Token)
	if err != nil {
		return trimble.ItemPage{}, err
	}
	v := url.Values{"pageSize": {strconv.Itoa(page.Size)}}
	if skip != "" {
		v.Set("skipToken", skip)
	}
	var body wireItemList
	if err := a.get(ctx, []string{"2.1", "folders", string(folder), "items"}, v, &body); err != nil {
		return trimble.ItemPage{}, err
	}
	if body.Items == nil {
		return trimble.ItemPage{}, errs.Newf(errs.UpstreamMalformed, "folder item response has no items array")
	}
	out := make([]trimble.Item, 0, len(*body.Items))
	for _, w := range *body.Items {
		if w.ProjectID == "" {
			// Ownership cannot be confirmed; fail closed.
			return trimble.ItemPage{}, errs.Newf(errs.UpstreamMalformed, "folder item has no projectId")
		}
		if domain.ProjectID(w.ProjectID) != project {
			return trimble.ItemPage{}, errs.New(errs.ResourceNotFound)
		}
		if w.ID == "" {
			return trimble.ItemPage{}, errs.Newf(errs.UpstreamMalformed, "folder item has no id")
		}
		out = append(out, trimble.Item{
			ID: w.ID, Kind: kind(w.Type), Name: w.Name, ParentID: w.ParentID,
			VersionID: w.VersionID, ModifiedAt: parseTime(w.ModifiedOn),
		})
	}
	info, err := a.pageInfo(body.Links, skip, page.Size, len(out))
	if err != nil {
		return trimble.ItemPage{}, err
	}
	return trimble.ItemPage{Items: out, Page: info, Prov: a.prov("GET /2.1/folders/{folderId}/items")}, nil
}

// GetFileMetadata implements trimble.FileReader via GET /2.0/files/{id}.
func (a *Adapter) GetFileMetadata(ctx context.Context, project domain.ProjectID, file domain.FileID) (trimble.FileMetadata, domain.Provenance, error) {
	var w wireFile
	if err := a.get(ctx, []string{"2.0", "files", string(file)}, nil, &w); err != nil {
		return trimble.FileMetadata{}, domain.Provenance{}, err
	}
	if w.ID == "" {
		return trimble.FileMetadata{}, domain.Provenance{}, errs.Newf(errs.UpstreamMalformed, "file response has no id")
	}
	if w.ProjectID == "" {
		return trimble.FileMetadata{}, domain.Provenance{}, errs.Newf(errs.UpstreamMalformed, "file response has no projectId")
	}
	if domain.ProjectID(w.ProjectID) != project {
		return trimble.FileMetadata{}, domain.Provenance{}, errs.New(errs.ResourceNotFound)
	}
	md := trimble.FileMetadata{
		Item: trimble.Item{
			ID: w.ID, Kind: kind(w.Type), Name: w.Name, ParentID: w.ParentID, VersionID: w.VersionID,
			SizeBytes: w.Size, ModifiedAt: parseTime(w.ModifiedOn), Checksum: w.Hash,
		},
		ProjectID: domain.ProjectID(w.ProjectID),
		CreatedAt: parseTime(w.CreatedOn),
		Revision:  w.Revision,
	}
	return md, a.prov("GET /2.0/files/{fileId}"), nil
}

// kind maps the documented upstream type values. Anything else is reported
// as-is rather than guessed.
func kind(t string) trimble.ItemKind {
	switch strings.ToUpper(t) {
	case "FOLDER":
		return trimble.KindFolder
	case "FILE":
		return trimble.KindFile
	}
	return trimble.ItemKind(strings.ToLower(t))
}

func (a *Adapter) prov(source string) domain.Provenance {
	return domain.Provenance{Product: Product, APIVersion: "2.x", Region: a.cfg.Region, Source: source, ObservedAt: a.now()}
}

// pageInfo turns links.next into an opaque bridge token. The href itself is
// never followed: only its skipToken is extracted, after checking the host,
// so a malicious or malformed link cannot redirect the bearer token.
func (a *Adapter) pageInfo(links wireLinks, current string, size, returned int) (domain.PageInfo, error) {
	info := domain.PageInfo{PageSize: size, Returned: returned, Complete: true}
	if links.Next == nil || links.Next.Href == "" {
		return info, nil
	}
	u, err := a.base.Parse(links.Next.Href)
	if err != nil || u.Host != a.base.Host {
		return info, errs.Newf(errs.UpstreamMalformed, "pagination link points outside the configured host")
	}
	next := u.Query().Get("skipToken")
	if next == "" {
		return info, errs.Newf(errs.UpstreamMalformed, "pagination link has no skipToken")
	}
	if next == current {
		return info, errs.Newf(errs.UpstreamMalformed, "pagination cursor repeated")
	}
	info.Complete = false
	info.NextToken = encodeToken(next)
	return info, nil
}

const tokenPrefix = "tc1:"

func encodeToken(skip string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(tokenPrefix + skip))
}

func decodeToken(tok string) (string, error) {
	if tok == "" {
		return "", nil
	}
	b, err := base64.RawURLEncoding.DecodeString(tok)
	if err != nil || !strings.HasPrefix(string(b), tokenPrefix) || len(b) == len(tokenPrefix) {
		return "", errs.Newf(errs.Validation, "page_token is not valid for trimble-connect")
	}
	return string(b[len(tokenPrefix):]), nil
}

func parseTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return nil
	}
	t = t.UTC()
	return &t
}

// ---- HTTP ----

func (a *Adapter) waitLimiter(ctx context.Context) error {
	for {
		ok, wait := a.limiter.Allow("connect")
		if ok {
			return nil
		}
		if err := a.sleep(ctx, wait); err != nil {
			return errs.Wrap(errs.UpstreamTimeout, err)
		}
	}
}

// get performs an authenticated GET with bounded, jittered retries. Each
// path segment is escaped individually, so an ID can never add segments.
func (a *Adapter) get(ctx context.Context, segments []string, q url.Values, out any) error {
	escaped := make([]string, len(segments))
	for i, s := range segments {
		escaped[i] = url.PathEscape(s)
	}
	u := a.base.JoinPath(escaped...)
	if q != nil {
		u.RawQuery = q.Encode()
	}
	var last error
	for attempt := 1; attempt <= a.cfg.MaxAttempts; attempt++ {
		if err := a.waitLimiter(ctx); err != nil {
			return err
		}
		retryAfter, err := a.do(ctx, u.String(), out)
		if err == nil {
			return nil
		}
		last = err
		e := errs.As(err)
		if !e.Retryable || attempt == a.cfg.MaxAttempts {
			break
		}
		backoff := time.Duration(1<<(attempt-1)) * 500 * time.Millisecond
		backoff += time.Duration(rand.Int64N(int64(250 * time.Millisecond)))
		if retryAfter > 0 {
			backoff = min(retryAfter, 30*time.Second)
		}
		if err := a.sleep(ctx, backoff); err != nil {
			return errs.Wrap(errs.UpstreamTimeout, err)
		}
	}
	return last
}

func (a *Adapter) do(ctx context.Context, rawURL string, out any) (time.Duration, error) {
	tok, err := a.cfg.Tokens.Token(ctx)
	if err != nil {
		return 0, errs.Wrap(errs.Authentication, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, errs.Wrap(errs.Internal, err)
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", a.cfg.UserAgent)
	resp, err := a.client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return 0, errs.Wrap(errs.UpstreamTimeout, redact(err))
		}
		if ctx.Err() != nil {
			return 0, errs.Wrap(errs.UpstreamTimeout, ctx.Err())
		}
		return 0, errs.Wrap(errs.UpstreamUnavailable, redact(err))
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		return 0, errs.Wrap(errs.UpstreamUnavailable, err)
	}
	if len(body) > maxBody {
		return 0, errs.Newf(errs.UpstreamMalformed, "upstream response exceeded %d bytes", maxBody)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if err := json.Unmarshal(body, out); err != nil {
			return 0, errs.Wrap(errs.UpstreamMalformed, err)
		}
		return 0, nil
	}
	return retryAfter(resp.Header.Get("Retry-After")), mapStatus(resp.StatusCode, body)
}

// mapStatus follows the retry guidance in the Connect reference error table.
// The upstream body is kept only as a private cause.
func mapStatus(status int, body []byte) error {
	cause := fmt.Errorf("upstream status %d: %.200s", status, body)
	switch status {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return errs.Wrap(errs.Validation, cause)
	case http.StatusUnauthorized:
		return errs.Wrap(errs.Authentication, cause)
	case http.StatusForbidden:
		return errs.Wrap(errs.Authorization, cause)
	case http.StatusNotFound:
		return errs.Wrap(errs.ResourceNotFound, cause)
	case http.StatusConflict:
		return errs.Wrap(errs.VersionConflict, cause)
	case http.StatusTooManyRequests:
		return errs.Wrap(errs.RateLimited, cause)
	case http.StatusGatewayTimeout:
		return errs.Wrap(errs.UpstreamTimeout, cause)
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return errs.Wrap(errs.UpstreamUnavailable, cause)
	}
	e := errs.Wrap(errs.UpstreamUnavailable, cause)
	e.Retryable = false
	return e
}

func retryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}
	if n, err := strconv.Atoi(v); err == nil && n >= 0 {
		return time.Duration(n) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

// redact strips URLs (which never carry tokens here, but may carry IDs) from
// transport errors before they are retained as causes.
func redact(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		return fmt.Errorf("%s request failed: %w", ue.Op, ue.Err)
	}
	return err
}
