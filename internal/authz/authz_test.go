package authz

import (
	"testing"
	"time"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/domain"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/errs"
)

func TestAuthorize(t *testing.T) {
	now := time.Now()
	p := &Principal{Subject: "a", Tenant: "t1", Scopes: []Scope{ScopeProjectsRead}, Projects: []domain.ProjectID{"p1"}, Products: []domain.Product{"trimble-connect"}}
	cases := []struct {
		name string
		p    *Principal
		r    Request
		code errs.Code
	}{
		{"ok", p, Request{Tenant: "t1", Scope: ScopeProjectsRead, Project: "p1", Product: "trimble-connect"}, ""},
		{"nil principal", nil, Request{Scope: ScopeProjectsRead}, errs.Authentication},
		{"cross tenant", p, Request{Tenant: "t2", Scope: ScopeProjectsRead}, errs.Authorization},
		{"missing scope", p, Request{Tenant: "t1", Scope: ScopeFilesRead}, errs.Authorization},
		{"write scope not implied", p, Request{Tenant: "t1", Scope: ScopeFilesWrite}, errs.Authorization},
		{"project not granted", p, Request{Tenant: "t1", Scope: ScopeProjectsRead, Project: "p2"}, errs.Authorization},
		{"product not granted", p, Request{Tenant: "t1", Scope: ScopeProjectsRead, Product: "mock"}, errs.Authorization},
		{"expired", &Principal{Subject: "a", Tenant: "t1", Scopes: []Scope{ScopeProjectsRead}, Expires: now.Add(-time.Second)}, Request{Scope: ScopeProjectsRead}, errs.Authentication},
	}
	for _, c := range cases {
		err := Authorize(c.p, c.r, now)
		if c.code == "" && err != nil || c.code != "" && !errs.Is(err, c.code) {
			t.Errorf("%s: got %v want %s", c.name, err, c.code)
		}
	}
}

func TestDefaultsAreReadOnly(t *testing.T) {
	for _, s := range ReadOnlyDefault {
		if s == ScopeFilesWrite || s == ScopeFilesDelete {
			t.Fatal("write scope in defaults")
		}
	}
	if _, err := ValidateScopes([]string{"trimble:everything"}); err == nil {
		t.Fatal("unknown scope accepted")
	}
}
