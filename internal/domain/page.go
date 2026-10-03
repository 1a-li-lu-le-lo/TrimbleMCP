package domain

import "github.com/1a-li-lu-le-lo/trimblemcp/internal/errs"

// PageRequest asks for one page of a list. Token is opaque to callers: it is
// produced by the bridge and only round-tripped.
type PageRequest struct {
	Token string
	Size  int
}

// Page limits applied to every list operation unless an adapter declares
// tighter ones.
const (
	DefaultPageSize = 50
	MaxPageSize     = 100
	maxTokenLen     = 512
)

// Normalize applies defaults and validates bounds.
func (p PageRequest) Normalize() (PageRequest, error) {
	if p.Size == 0 {
		p.Size = DefaultPageSize
	}
	if p.Size < 1 || p.Size > MaxPageSize {
		return p, errs.Newf(errs.Validation, "page_size must be between 1 and %d", MaxPageSize)
	}
	if len(p.Token) > maxTokenLen {
		return p, errs.Newf(errs.Validation, "page_token is too long")
	}
	return p, nil
}

// PageInfo describes where a returned page sits in the full result set. A
// list result is complete only when Complete is true; tools never silently
// stop at the first page.
type PageInfo struct {
	NextToken string `json:"next_page_token,omitempty"`
	Complete  bool   `json:"complete"`
	PageSize  int    `json:"page_size"`
	Returned  int    `json:"returned"`
	Total     *int   `json:"total,omitempty"`
}
