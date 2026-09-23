// Package authz implements the server-side authorization decision for every
// MCP tool call. Decisions depend only on the authenticated Principal and the
// requested operation — never on model output or conversational tone.
package authz

import (
	"slices"
	"time"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/domain"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/errs"
)

// Scope is an internal bridge scope. It is independent of upstream Trimble
// OAuth scopes.
type Scope string

const (
	ScopeCapabilitiesRead Scope = "trimble:capabilities:read"
	ScopeProjectsRead     Scope = "trimble:projects:read"
	ScopeFilesRead        Scope = "trimble:files:read"
	ScopeFilesWrite       Scope = "trimble:files:write"
	ScopeFilesDelete      Scope = "trimble:files:delete"
	ScopeAuditRead        Scope = "trimble:audit:read"
	// ScopeDesktopLaunch permits opening a desktop Trimble application on the
	// operator's own machine. Never granted by default; local principals only.
	ScopeDesktopLaunch Scope = "trimble:desktop:launch"
)

// ReadOnlyDefault is granted when configuration does not say otherwise.
var ReadOnlyDefault = []Scope{ScopeCapabilitiesRead, ScopeProjectsRead}

// KnownScopes lists every scope the bridge recognises. Unknown scopes in
// configuration are rejected rather than silently ignored.
var KnownScopes = []Scope{
	ScopeCapabilitiesRead, ScopeProjectsRead, ScopeFilesRead,
	ScopeFilesWrite, ScopeFilesDelete, ScopeAuditRead, ScopeDesktopLaunch,
}

// Principal is an authenticated caller bound to exactly one tenant.
type Principal struct {
	Subject  string           `json:"subject"`
	Client   string           `json:"client"`
	Tenant   domain.TenantID  `json:"tenant"`
	Scopes   []Scope          `json:"scopes"`
	Products []domain.Product `json:"products,omitempty"`
	// Projects restricts access to listed projects. Empty means every project
	// the tenant's upstream credential can see.
	Projects []domain.ProjectID `json:"projects,omitempty"`
	Expires  time.Time          `json:"expires,omitempty"`
	// Local is true only for the operator principal of a stdio or CLI
	// process on the operator's own machine. Remote principals are never
	// local, so they can never launch desktop applications.
	Local bool `json:"local"`
}

// Request is the resource and operation being authorized.
type Request struct {
	Tenant  domain.TenantID
	Product domain.Product
	Scope   Scope
	Project domain.ProjectID // empty when not project-scoped
}

// Authorize returns nil when p may perform r.
func Authorize(p *Principal, r Request, now time.Time) error {
	if p == nil || p.Subject == "" || p.Tenant == "" {
		return errs.New(errs.Authentication)
	}
	if !p.Expires.IsZero() && now.After(p.Expires) {
		return errs.Newf(errs.Authentication, "the caller's authorization has expired")
	}
	if r.Tenant != "" && r.Tenant != p.Tenant {
		// Never reveal whether the other tenant exists.
		return errs.New(errs.Authorization)
	}
	if !slices.Contains(p.Scopes, r.Scope) {
		return errs.Newf(errs.Authorization, "missing scope %s", r.Scope)
	}
	if r.Product != "" && len(p.Products) > 0 && !slices.Contains(p.Products, r.Product) {
		return errs.Newf(errs.Authorization, "product %s is not granted to this caller", r.Product)
	}
	if r.Project != "" && !p.ProjectAllowed(r.Project) {
		return errs.New(errs.Authorization)
	}
	return nil
}

// ProjectAllowed reports whether the principal's project restriction admits id.
func (p *Principal) ProjectAllowed(id domain.ProjectID) bool {
	return len(p.Projects) == 0 || slices.Contains(p.Projects, id)
}

// ValidateScopes rejects unknown scopes.
func ValidateScopes(in []string) ([]Scope, error) {
	out := make([]Scope, 0, len(in))
	for _, s := range in {
		if !slices.Contains(KnownScopes, Scope(s)) {
			return nil, errs.Newf(errs.Configuration, "unknown scope %q", s)
		}
		out = append(out, Scope(s))
	}
	return out, nil
}
