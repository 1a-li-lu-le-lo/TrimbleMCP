// Package gateway maps authorized MCP tool calls onto Trimble adapters. It is
// the only place where authorization, audit, input validation, and output
// shaping meet.
package gateway

import (
	"sort"
	"sync"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/domain"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/errs"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/trimble"
)

// Registry holds adapters keyed by tenant and product. Adapters for one
// tenant are unreachable from another tenant's principal by construction.
type Registry struct {
	mu       sync.RWMutex
	adapters map[domain.TenantID]map[domain.Product]trimble.Base
	disabled map[domain.Product]bool
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		adapters: map[domain.TenantID]map[domain.Product]trimble.Base{},
		disabled: map[domain.Product]bool{},
	}
}

// Register binds an adapter to a tenant. The adapter's credentials must
// belong to that tenant.
func (r *Registry) Register(tenant domain.TenantID, a trimble.Base) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m := r.adapters[tenant]
	if m == nil {
		m = map[domain.Product]trimble.Base{}
		r.adapters[tenant] = m
	}
	m[a.Describe().Product] = a
}

// Disable is the per-product kill switch; it takes effect immediately.
func (r *Registry) Disable(p domain.Product, off bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.disabled[p] = off
}

// Adapters returns the tenant's enabled adapters sorted by product.
func (r *Registry) Adapters(tenant domain.TenantID) []trimble.Base {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []trimble.Base
	for p, a := range r.adapters[tenant] {
		if !r.disabled[p] {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Describe().Product < out[j].Describe().Product })
	return out
}

// Resolve returns the tenant's adapter for product, or an error that does not
// reveal whether the product exists for other tenants.
func (r *Registry) Resolve(tenant domain.TenantID, product domain.Product) (trimble.Base, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.adapters[tenant][product]
	if !ok || r.disabled[product] {
		return nil, errs.New(errs.UnsupportedProduct)
	}
	return a, nil
}

// anySupports reports whether any enabled adapter for tenant declares c.
func (r *Registry) anySupports(tenant domain.TenantID, c trimble.Capability) bool {
	for _, a := range r.Adapters(tenant) {
		if a.Describe().Supports(c) {
			return true
		}
	}
	return false
}
