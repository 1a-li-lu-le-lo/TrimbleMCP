// Package desktop supports the documented command-line interface of Trimble
// Connect for Windows: its registered URL scheme
//
//	"trimbleconnect:/projects/[project-id]?show=[view parameter],[panel parameter]"
//
// which opens the desktop application at a project, view, and panel. The
// interface only navigates the application; it reads and changes no data.
// Source: https://help.trimble.com/doc/trimble-connect/trimble-connect/connect-for-windows/getting-started/using-the-command-line
package desktop

import (
	"context"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/domain"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/errs"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/trimble"
)

// Product is the bridge's identifier for Trimble Connect for Windows.
const Product domain.Product = "trimble-connect-desktop"

// Source is the official page the URI syntax is implemented from.
const Source = "https://help.trimble.com/doc/trimble-connect/trimble-connect/connect-for-windows/getting-started/using-the-command-line"

// LastVerified is the date Source was last checked.
const LastVerified = "2026-09-23"

// Scheme prefix exactly as documented (single slash).
const schemePrefix = "trimbleconnect:/projects/"

// Documented parameter values. Parameters are case-insensitive upstream; the
// bridge accepts any casing and emits the documented spelling.
var (
	views  = []string{"projects", "data", "3D"}
	panels = []string{"clashes", "models", "objects", "ToDos", "views"}
)

// Views returns the documented view parameter values.
func Views() []string { return append([]string(nil), views...) }

// Panels returns the documented panel parameter values.
func Panels() []string { return append([]string(nil), panels...) }

// projectID is deliberately narrower than domain IDs: the URI is handed to
// the operating system's protocol handler, so only characters that need no
// escaping in any URI component are allowed. The documented example project
// ID (rQR1yhTGj9I) fits this shape.
var projectID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

func canonical(v string, allowed []string) (string, bool) {
	for _, a := range allowed {
		if strings.EqualFold(v, a) {
			return a, true
		}
	}
	return "", false
}

// BuildURI validates inputs and returns the launch URI. view and panel are
// optional; a panel requires a view because the documented form is
// "show=[view],[panel]" and a panel-only form is not documented.
func BuildURI(project domain.ProjectID, view, panel string) (trimble.DesktopLink, error) {
	if !projectID.MatchString(string(project)) {
		return trimble.DesktopLink{}, errs.Newf(errs.Validation,
			"project_id must be 1-64 letters, digits, '-' or '_' for the desktop launch URI")
	}
	link := trimble.DesktopLink{Project: project, URI: schemePrefix + string(project)}
	if view == "" && panel == "" {
		return link, nil
	}
	if view == "" {
		return trimble.DesktopLink{}, errs.Newf(errs.Validation, "panel requires a view (documented form is show=[view],[panel])")
	}
	v, ok := canonical(view, views)
	if !ok {
		return trimble.DesktopLink{}, errs.Newf(errs.Validation, "view must be one of %s", strings.Join(views, ", "))
	}
	link.View = v
	show := v
	if panel != "" {
		pn, ok := canonical(panel, panels)
		if !ok {
			return trimble.DesktopLink{}, errs.Newf(errs.Validation, "panel must be one of %s", strings.Join(panels, ", "))
		}
		link.Panel = pn
		show += "," + pn
	}
	link.URI += "?show=" + show
	return link, nil
}

// OpenFunc hands a validated URI to the operating system.
type OpenFunc func(ctx context.Context, uri string) error

// Config configures the adapter.
type Config struct {
	// LaunchEnabled permits opening the application. When false only links
	// are built. Launching additionally requires a Windows host.
	LaunchEnabled bool
	// Open overrides the OS launcher (tests). Nil uses the platform default.
	Open OpenFunc
	// GOOS overrides runtime.GOOS (tests).
	GOOS string
}

// Adapter implements trimble.DesktopLauncher.
type Adapter struct {
	cfg  Config
	open OpenFunc
	now  func() time.Time
}

// New returns an adapter.
func New(cfg Config) *Adapter {
	if cfg.GOOS == "" {
		cfg.GOOS = runtime.GOOS
	}
	open := cfg.Open
	if open == nil {
		open = platformOpen
	}
	return &Adapter{cfg: cfg, open: open, now: func() time.Time { return time.Now().UTC() }}
}

func (a *Adapter) canLaunch() bool { return a.cfg.LaunchEnabled && a.cfg.GOOS == "windows" }

// Describe implements trimble.Base.
func (a *Adapter) Describe() trimble.Descriptor {
	caps := []trimble.Capability{trimble.CapDesktopLink}
	if a.canLaunch() {
		caps = append(caps, trimble.CapDesktopLaunch)
	}
	launch := "disabled (TRIMBLE_CONNECT_DESKTOP_LAUNCH=false)"
	switch {
	case a.canLaunch():
		launch = "enabled on this Windows host via ShellExecuteW"
	case a.cfg.LaunchEnabled:
		launch = "unavailable: host OS is " + a.cfg.GOOS + ", Trimble Connect for Windows requires Windows"
	}
	return trimble.Descriptor{
		Product:      Product,
		OfficialName: "Trimble Connect for Windows command line (trimbleconnect: URL scheme)",
		APIVersion:   "documented URL scheme",
		BaseURL:      "trimbleconnect:/projects/{project-id}?show={view},{panel}",
		Auth:         "none at the bridge; the desktop application uses its own signed-in Trimble ID session",
		TenantModel:  "whatever Trimble ID is signed in to Trimble Connect for Windows on the operator's machine",
		Capabilities: caps,
		Pagination:   "not applicable",
		RateLimits:   "bridge limits launches per caller",
		Idempotency:  "link building is pure; each launch opens or focuses the application",
		Webhooks:     "none",
		Retries:      "none",
		TimeoutText:  "n/a",
		Units:        "not applicable",
		CRS:          "not applicable",
		DataClass:    "project identifier only; no project data is read",
		Owner:        "trimble-mcp-bridge maintainers",
		Sources:      []string{Source},
		LastVerified: LastVerified,
		// The URI syntax is verified; that its project ID equals the REST API
		// project ID is an assumption until confirmed (see assumptions A-8).
		Status:        trimble.Provisional,
		DisableSwitch: "TRIMBLE_CONNECT_DESKTOP_ENABLED=false; launch: TRIMBLE_CONNECT_DESKTOP_LAUNCH=false",
		ReadOnly:      true,
		LaunchMode:    launch,
	}
}

// Health implements trimble.Base.
func (a *Adapter) Health(context.Context) trimble.Health {
	return trimble.Health{Status: "ok", CheckedAt: a.now(), Detail: "link builder ready; application presence is not probed"}
}

// BuildLink implements trimble.DesktopLauncher.
func (a *Adapter) BuildLink(project domain.ProjectID, view, panel string) (trimble.DesktopLink, error) {
	return BuildURI(project, view, panel)
}

// Launch implements trimble.DesktopLauncher. The link is rebuilt from its
// parts so a caller cannot smuggle a different URI through.
func (a *Adapter) Launch(ctx context.Context, link trimble.DesktopLink) error {
	if !a.canLaunch() {
		return errs.Newf(errs.UnsupportedCapability, "launching Trimble Connect for Windows is not enabled on this host")
	}
	rebuilt, err := BuildURI(link.Project, link.View, link.Panel)
	if err != nil {
		return err
	}
	if rebuilt.URI != link.URI {
		return errs.Newf(errs.Validation, "launch URI does not match its validated parts")
	}
	if err := a.open(ctx, rebuilt.URI); err != nil {
		return errs.Wrap(errs.UpstreamUnavailable, err).WithSafeMessage(
			"Windows could not open the trimbleconnect: link; is Trimble Connect for Windows installed?")
	}
	return nil
}
