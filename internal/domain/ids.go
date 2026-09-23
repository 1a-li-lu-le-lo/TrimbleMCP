// Package domain holds the canonical value objects shared by every adapter.
// Only concepts that are genuinely shared across Trimble products live here;
// product-specific data stays behind its adapter.
package domain

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/errs"
)

// Distinct identifier types. They are deliberately separate named types so a
// FolderID cannot be passed where a ProjectID is expected without an explicit,
// visible conversion. Values always originate from an upstream API or from
// operator configuration; the bridge never fabricates them.
type (
	TenantID       string
	OrganizationID string
	ProjectID      string
	FolderID       string
	FileID         string
	VersionID      string
	UserID         string
	JobID          string
	AuditID        string
	Product        string
	Region         string
)

const maxIDLen = 256

// validateID enforces a conservative identifier shape. Upstream IDs are opaque,
// but none of the verified APIs use whitespace, control characters, or URL
// delimiters in IDs, and rejecting them blocks path and query injection when
// IDs are placed into request paths.
func validateID(kind, s string) error {
	if s == "" {
		return errs.Newf(errs.Validation, "%s is required", kind)
	}
	if len(s) > maxIDLen {
		return errs.Newf(errs.Validation, "%s exceeds %d characters", kind, maxIDLen)
	}
	if !utf8.ValidString(s) {
		return errs.Newf(errs.Validation, "%s is not valid UTF-8", kind)
	}
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsControl(r) || !unicode.IsPrint(r) {
			return errs.Newf(errs.Validation, "%s contains whitespace or control characters", kind)
		}
		if strings.ContainsRune(`/\?#%&=+;"'<>{}|^`+"`", r) {
			return errs.Newf(errs.Validation, "%s contains a reserved character %q", kind, r)
		}
	}
	if s == "." || s == ".." {
		return errs.Newf(errs.Validation, "%s is not a valid identifier", kind)
	}
	return nil
}

func ParseTenantID(s string) (TenantID, error) {
	return TenantID(s), validateID("tenant_id", s)
}

func ParseProjectID(s string) (ProjectID, error) {
	return ProjectID(s), validateID("project_id", s)
}

func ParseFolderID(s string) (FolderID, error) {
	return FolderID(s), validateID("folder_id", s)
}

func ParseFileID(s string) (FileID, error) {
	return FileID(s), validateID("file_id", s)
}

func ParseProduct(s string) (Product, error) {
	if err := validateID("product", s); err != nil {
		return "", err
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return "", errs.Newf(errs.Validation, "product must be lower-case letters, digits, '-' or '_'")
		}
	}
	return Product(s), nil
}

func ParseRegion(s string) (Region, error) {
	return Region(s), validateID("region", s)
}

// MustID panics on invalid IDs. For tests and compile-time constants only.
func MustID[T ~string](v T, err error) T {
	if err != nil {
		panic(fmt.Sprintf("invalid id %q: %v", string(v), err))
	}
	return v
}
