// Package trimble defines the capability-specific adapter contracts. There is
// no universal "Trimble client": each verified API family implements only the
// interfaces its official documentation supports, and the gateway exposes a
// tool only when some enabled adapter implements the matching interface.
package trimble

import (
	"context"
	"time"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/domain"
)

// Capability names an operation an adapter may support.
type Capability string

const (
	CapListProjects Capability = "list_projects"
	CapGetProject   Capability = "get_project"
	CapListFolder   Capability = "list_folder_items"
	CapFileMetadata Capability = "get_file_metadata"
	// CapDesktopLink builds a documented Trimble Connect for Windows launch
	// URI. It has no side effects.
	CapDesktopLink Capability = "build_desktop_link"
	// CapDesktopLaunch opens Trimble Connect for Windows on the operator's own
	// machine via its registered URL scheme. It reads and changes no data.
	CapDesktopLaunch Capability = "launch_desktop"
)

// VerificationStatus records how well an adapter's contract is evidenced.
type VerificationStatus string

const (
	// Verified: every implemented endpoint is traced to current official docs.
	Verified VerificationStatus = "verified"
	// Provisional: implemented from official docs that could not be fully
	// re-verified in the current session; must be re-verified before pilot.
	Provisional VerificationStatus = "provisional"
	// Simulated: a mock with no upstream; never talks to Trimble.
	Simulated VerificationStatus = "simulated"
)

// Descriptor is the declaration every adapter publishes. It is surfaced
// (without secrets) through trimble_get_capabilities.
type Descriptor struct {
	Product       domain.Product     `json:"product"`
	OfficialName  string             `json:"official_name"`
	APIVersion    string             `json:"api_version"`
	BaseURL       string             `json:"base_url"`
	Auth          string             `json:"auth"`
	Scopes        []string           `json:"upstream_scopes"`
	TenantModel   string             `json:"tenant_model"`
	Capabilities  []Capability       `json:"capabilities"`
	Pagination    string             `json:"pagination"`
	RateLimits    string             `json:"rate_limits"`
	Idempotency   string             `json:"idempotency"`
	Webhooks      string             `json:"webhooks"`
	Retries       string             `json:"retries"`
	Timeout       time.Duration      `json:"-"`
	TimeoutText   string             `json:"timeout"`
	Units         string             `json:"units"`
	CRS           string             `json:"crs"`
	Region        domain.Region      `json:"region,omitempty"`
	DataClass     string             `json:"data_class"`
	Owner         string             `json:"owner"`
	Sources       []string           `json:"sources"`
	LastVerified  string             `json:"last_verified"`
	Status        VerificationStatus `json:"verification_status"`
	DisableSwitch string             `json:"disable_switch"`
	ReadOnly      bool               `json:"read_only"`
	// LaunchMode describes local-application launching, when relevant.
	LaunchMode string `json:"launch_mode,omitempty"`
}

// Supports reports whether the descriptor lists c.
func (d Descriptor) Supports(c Capability) bool {
	for _, x := range d.Capabilities {
		if x == c {
			return true
		}
	}
	return false
}

// Health is a point-in-time upstream health observation.
type Health struct {
	Status    string    `json:"status"` // ok | degraded | unavailable | unknown
	CheckedAt time.Time `json:"checked_at"`
	Detail    string    `json:"detail,omitempty"`
}

// Base is implemented by every adapter.
type Base interface {
	Describe() Descriptor
	Health(ctx context.Context) Health
}

// Project is the canonical project summary. Fields absent upstream stay empty;
// nothing is inferred.
type Project struct {
	ID          domain.ProjectID `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	RootFolder  domain.FolderID  `json:"root_folder_id,omitempty"`
	Region      domain.Region    `json:"region,omitempty"`
	CreatedAt   *time.Time       `json:"created_at,omitempty"`
	UpdatedAt   *time.Time       `json:"updated_at,omitempty"`
	Access      string           `json:"access,omitempty"`
}

// ItemKind distinguishes folder entries.
type ItemKind string

const (
	KindFolder ItemKind = "folder"
	KindFile   ItemKind = "file"
)

// Item is an entry within a folder.
type Item struct {
	ID         string     `json:"id"`
	Kind       ItemKind   `json:"kind"`
	Name       string     `json:"name"`
	ParentID   string     `json:"parent_id,omitempty"`
	VersionID  string     `json:"version_id,omitempty"`
	SizeBytes  *int64     `json:"size_bytes,omitempty"`
	ModifiedAt *time.Time `json:"modified_at,omitempty"`
	Checksum   string     `json:"checksum,omitempty"`
	// ChecksumAlgorithm names the documented algorithm of Checksum.
	ChecksumAlgorithm string `json:"checksum_algorithm,omitempty"`
}

// FileMetadata describes one file.
type FileMetadata struct {
	Item
	ProjectID domain.ProjectID `json:"project_id,omitempty"`
	CreatedAt *time.Time       `json:"created_at,omitempty"`
	Revision  *int             `json:"revision,omitempty"`
}

// ListProjectsQuery narrows a project listing.
type ListProjectsQuery struct {
	Page domain.PageRequest
}

// ProjectPage is one page of projects.
type ProjectPage struct {
	Projects []Project
	Page     domain.PageInfo
	Prov     domain.Provenance
}

// ProjectReader lists and reads projects.
type ProjectReader interface {
	Base
	ListProjects(ctx context.Context, q ListProjectsQuery) (ProjectPage, error)
	GetProject(ctx context.Context, id domain.ProjectID) (Project, domain.Provenance, error)
}

// ItemPage is one page of folder items.
type ItemPage struct {
	Items []Item
	Page  domain.PageInfo
	Prov  domain.Provenance
}

// FileReader reads folder listings and file metadata. It never downloads
// content; downloads are a separate Level 1 capability.
//
// Every call names the project the caller was authorized for. Implementations
// must confirm, from upstream data, that the folder or file belongs to that
// project and return errs.ResourceNotFound otherwise. This closes the IDOR gap
// where an opaque folder or file ID from another project is supplied.
type FileReader interface {
	Base
	ListFolderItems(ctx context.Context, project domain.ProjectID, folder domain.FolderID, page domain.PageRequest) (ItemPage, error)
	GetFileMetadata(ctx context.Context, project domain.ProjectID, file domain.FileID) (FileMetadata, domain.Provenance, error)
}

// DesktopLink is a validated launch URI for a desktop Trimble application.
type DesktopLink struct {
	URI     string           `json:"uri"`
	Project domain.ProjectID `json:"project_id"`
	View    string           `json:"view,omitempty"`
	Panel   string           `json:"panel,omitempty"`
}

// DesktopLauncher builds and opens launch URIs for a desktop application's
// documented command-line interface. It never reads or writes project data.
type DesktopLauncher interface {
	Base
	BuildLink(project domain.ProjectID, view, panel string) (DesktopLink, error)
	Launch(ctx context.Context, link DesktopLink) error
}
