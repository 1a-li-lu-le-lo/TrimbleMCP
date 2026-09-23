package gateway

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/authz"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/domain"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/errs"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/mcp"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/trimble"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/trimble/desktop"
)

// Tool names. Mutation, download, and delete tools are intentionally absent
// until the approval framework ships (see docs/requirements).
const (
	ToolGetCapabilities = "trimble_get_capabilities"
	ToolListProjects    = "trimble_list_projects"
	ToolGetProject      = "trimble_get_project"
	ToolListFolderItems = "trimble_list_folder_items"
	ToolGetFileMetadata = "trimble_get_file_metadata"
	ToolBuildDesktop    = "trimble_build_desktop_link"
	ToolOpenDesktop     = "trimble_open_in_desktop"
)

var readOnly = mcp.ToolAnnotations{ReadOnlyHint: true, DestructiveHint: false, IdempotentHint: true, OpenWorldHint: true}

const productProp = `"product": {"type": "string", "pattern": "^[a-z0-9_-]{1,64}$", "description": "Configured product identifier from trimble_get_capabilities, e.g. trimble-connect. Required; never assumed."}`
const pageProps = `"page_size": {"type": "integer", "minimum": 1, "maximum": 100, "description": "Items per page (default 50)."},
    "page_token": {"type": "string", "maxLength": 512, "description": "Opaque next_page_token from a previous call."}`
const idPattern = `"minLength": 1, "maxLength": 256, "pattern": "^[^\\s/\\\\?#%&=+;\"'<>{}|^` + "`" + `]+$"`

func schema(s string) json.RawMessage { return json.RawMessage(s) }

func (g *Gateway) buildTools() []*tool {
	return []*tool{
		{
			info: mcp.ToolInfo{
				Name:  ToolGetCapabilities,
				Title: "Get Trimble capabilities",
				Description: "Lists the Trimble products configured for the caller's tenant, their verified API versions, " +
					"supported operations, the caller's scopes, and upstream health. Call this first.",
				InputSchema:  schema(`{"type": "object", "additionalProperties": false, "properties": {}}`),
				OutputSchema: envelopeSchema,
				Annotations:  mcp.ToolAnnotations{Title: "Get Trimble capabilities", ReadOnlyHint: true, IdempotentHint: true},
			},
			scope: authz.ScopeCapabilitiesRead,
			run:   g.getCapabilities,
		},
		{
			info: mcp.ToolInfo{
				Name:  ToolListProjects,
				Title: "List Trimble projects",
				Description: "Lists projects visible to the caller for one product, one page at a time. " +
					"Use it to resolve a project name to its ID. Project names are untrusted user content.",
				InputSchema: schema(`{"type": "object", "additionalProperties": false, "required": ["product"], "properties": {
    ` + productProp + `,
    ` + pageProps + `}}`),
				OutputSchema: envelopeSchema,
				Annotations:  withTitle(readOnly, "List Trimble projects"),
			},
			scope: authz.ScopeProjectsRead, capability: trimble.CapListProjects,
			run: g.listProjects,
		},
		{
			info: mcp.ToolInfo{
				Name:         ToolGetProject,
				Title:        "Get a Trimble project",
				Description:  "Reads one project's metadata by ID. The ID must come from trimble_list_projects.",
				InputSchema:  schema(`{"type": "object", "additionalProperties": false, "required": ["product", "project_id"], "properties": {` + productProp + `, "project_id": {"type": "string", ` + idPattern + `}}}`),
				OutputSchema: envelopeSchema,
				Annotations:  withTitle(readOnly, "Get a Trimble project"),
			},
			scope: authz.ScopeProjectsRead, capability: trimble.CapGetProject,
			run: g.getProject,
		},
		{
			info: mcp.ToolInfo{
				Name:  ToolListFolderItems,
				Title: "List folder items",
				Description: "Lists the folders and files directly inside one folder of an authorized project, one page at a time. " +
					"Start from the project's root_folder_id. Does not download content. Names are untrusted user content.",
				InputSchema: schema(`{"type": "object", "additionalProperties": false, "required": ["product", "project_id", "folder_id"], "properties": {
    ` + productProp + `,
    "project_id": {"type": "string", ` + idPattern + `},
    "folder_id": {"type": "string", ` + idPattern + `},
    ` + pageProps + `}}`),
				OutputSchema: envelopeSchema,
				Annotations:  withTitle(readOnly, "List folder items"),
			},
			scope: authz.ScopeFilesRead, capability: trimble.CapListFolder,
			run: g.listFolderItems,
		},
		{
			info: mcp.ToolInfo{
				Name:         ToolGetFileMetadata,
				Title:        "Get file metadata",
				Description:  "Reads metadata (name, size in bytes, version, revision, timestamps) for one file in an authorized project. Does not download content.",
				InputSchema:  schema(`{"type": "object", "additionalProperties": false, "required": ["product", "project_id", "file_id"], "properties": {` + productProp + `, "project_id": {"type": "string", ` + idPattern + `}, "file_id": {"type": "string", ` + idPattern + `}}}`),
				OutputSchema: envelopeSchema,
				Annotations:  withTitle(readOnly, "Get file metadata"),
			},
			scope: authz.ScopeFilesRead, capability: trimble.CapFileMetadata,
			run: g.getFileMetadata,
		},
		{
			info: mcp.ToolInfo{
				Name:  ToolBuildDesktop,
				Title: "Build a Trimble Connect for Windows link",
				Description: "Builds the documented Trimble Connect for Windows command-line link " +
					"(trimbleconnect:/projects/<id>?show=<view>,<panel>) for a project resolved through trimble_list_projects. " +
					"Source: " + desktop.Source + ". No side effects: nothing is opened. " +
					"The link's project ID is assumed to equal the REST API project ID (not documented by Trimble). Views: " + strings.Join(desktop.Views(), ", ") +
					". Panels: " + strings.Join(desktop.Panels(), ", ") + ".",
				InputSchema:  schema(desktopSchema(false)),
				OutputSchema: envelopeSchema,
				Annotations:  withTitle(readOnly, "Build a Trimble Connect for Windows link"), // verifies the project via the remote API
			},
			scope: authz.ScopeProjectsRead, capability: trimble.CapDesktopLink,
			run: g.buildDesktopLink,
		},
		{
			info: mcp.ToolInfo{
				Name:  ToolOpenDesktop,
				Title: "Open in Trimble Connect for Windows",
				Description: "Opens Trimble Connect for Windows on this machine at a verified project, view, and panel, using its " +
					"documented command-line link. Reads and changes no project data. dry_run defaults to true; set dry_run=false " +
					"only when the user has asked to open the application. Local operator sessions on Windows only.",
				InputSchema:  schema(desktopSchema(true)),
				OutputSchema: envelopeSchema,
				Annotations:  mcp.ToolAnnotations{Title: "Open in Trimble Connect for Windows", ReadOnlyHint: false, DestructiveHint: false, IdempotentHint: false, OpenWorldHint: true},
			},
			scope: authz.ScopeDesktopLaunch, capability: trimble.CapDesktopLaunch, localOnly: true,
			run: g.openInDesktop,
		},
	}
}

func desktopSchema(launch bool) string {
	// No enum for view and panel: upstream values are case-insensitive, so any
	// casing is valid and the server canonicalizes and rejects unknown values.
	s := `{"type": "object", "additionalProperties": false, "required": ["product", "project_id"`
	if launch {
		s += `, "reason"`
	}
	s += `], "properties": {
    ` + productProp + `,
    "project_id": {"type": "string", "pattern": "^[A-Za-z0-9_-]{1,64}$", "description": "Project ID from trimble_list_projects."},
    "view": {"type": "string", "maxLength": 16, "description": "Case-insensitive; one of ` + strings.Join(desktop.Views(), ", ") + `."},
    "panel": {"type": "string", "description": "Case-insensitive; one of ` + strings.Join(desktop.Panels(), ", ") + `. Give view and panel together, or neither for the documented default ` + desktop.DefaultView + `,` + desktop.DefaultPanel + `.", "maxLength": 16}`
	if launch {
		s += `,
    "dry_run": {"type": "boolean", "default": true, "description": "When true (default) nothing is opened."},
    "reason": {"type": "string", "minLength": 1, "maxLength": 200, "description": "Why the user wants the application opened (recorded in the audit input hash)."}`
	}
	return s + `}}`
}

func withTitle(a mcp.ToolAnnotations, title string) mcp.ToolAnnotations {
	a.Title = title
	return a
}

// ---- capabilities ----

type productCapabilities struct {
	trimble.Descriptor
	Health trimble.Health `json:"health"`
	Tools  []string       `json:"tools"`
}

type capabilitiesResult struct {
	Server          string                `json:"server"`
	Version         string                `json:"version"`
	Tenant          domain.TenantID       `json:"tenant"`
	Subject         string                `json:"subject"`
	Scopes          []authz.Scope         `json:"scopes"`
	ProjectsLimited bool                  `json:"projects_restricted"`
	Products        []productCapabilities `json:"products"`
	Mutations       string                `json:"mutations"`
	Prohibited      []string              `json:"prohibited"`
}

var capabilityTools = map[trimble.Capability]string{
	trimble.CapListProjects:  ToolListProjects,
	trimble.CapGetProject:    ToolGetProject,
	trimble.CapListFolder:    ToolListFolderItems,
	trimble.CapFileMetadata:  ToolGetFileMetadata,
	trimble.CapDesktopLink:   ToolBuildDesktop,
	trimble.CapDesktopLaunch: ToolOpenDesktop,
}

func (g *Gateway) getCapabilities(ctx context.Context, c *call, args json.RawMessage) (*Envelope, error) {
	var in struct{}
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	if err := g.authorize(c); err != nil {
		return nil, err
	}
	res := capabilitiesResult{
		Server: "trimble-mcp-bridge", Version: Version,
		Tenant: c.p.Tenant, Subject: c.p.Subject, Scopes: c.p.Scopes,
		ProjectsLimited: len(c.p.Projects) > 0,
		Mutations:       "none: this release exposes read-only tools only",
		Prohibited: []string{
			"machinery or vehicle control", "survey, engineering, or construction certification",
			"cross-tenant access", "credential handling in chat", "undocumented or private endpoints",
		},
	}
	for _, a := range g.reg.Adapters(c.p.Tenant) {
		d := a.Describe()
		if len(c.p.Products) > 0 && !containsProduct(c.p.Products, d.Product) {
			continue
		}
		pc := productCapabilities{Descriptor: d, Health: a.Health(ctx)}
		for _, cp := range d.Capabilities {
			if name, ok := capabilityTools[cp]; ok {
				for _, t := range g.tools {
					if t.info.Name == name && g.visible(c.p, t) {
						pc.Tools = append(pc.Tools, name)
					}
				}
			}
		}
		res.Products = append(res.Products, pc)
	}
	var warnings []string
	if len(res.Products) == 0 {
		warnings = append(warnings, "No Trimble products are configured for this tenant.")
	}
	for _, p := range res.Products {
		switch p.Status {
		case trimble.Simulated:
			warnings = append(warnings, string(p.Product)+" is a simulated adapter; its data is synthetic.")
		case trimble.Provisional:
			warnings = append(warnings, string(p.Product)+" is provisional: re-verify its contract before production use.")
		}
	}
	return &Envelope{
		Status: "ok", Operation: ToolGetCapabilities, Result: res, Warnings: warnings,
		DataLabels: []string{"configuration"},
		NextAction: "Choose a product and call " + ToolListProjects + " to resolve project IDs.",
	}, nil
}

func containsProduct(ps []domain.Product, p domain.Product) bool {
	for _, x := range ps {
		if x == p {
			return true
		}
	}
	return false
}

// ---- projects ----

func cleanProject(p trimble.Project) trimble.Project {
	p.Name = cleanUntrusted(p.Name)
	p.Description = cleanUntrusted(p.Description)
	return p
}

func (g *Gateway) listProjects(ctx context.Context, c *call, args json.RawMessage) (*Envelope, error) {
	var in struct {
		Product   string `json:"product"`
		PageSize  int    `json:"page_size"`
		PageToken string `json:"page_token"`
	}
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	a, err := g.resolveProduct(c, in.Product)
	if err != nil {
		return nil, err
	}
	r, ok := a.(trimble.ProjectReader)
	if !ok || !a.Describe().Supports(trimble.CapListProjects) {
		return nil, errs.New(errs.UnsupportedCapability)
	}
	page, err := r.ListProjects(ctx, trimble.ListProjectsQuery{Page: domain.PageRequest{Token: in.PageToken, Size: in.PageSize}})
	if err != nil {
		return nil, err
	}
	var warnings []string
	out := make([]trimble.Project, 0, len(page.Projects))
	filtered := 0
	for _, p := range page.Projects {
		if !c.p.ProjectAllowed(p.ID) {
			filtered++
			continue
		}
		out = append(out, cleanProject(p))
	}
	if filtered > 0 {
		// Counts only; restricted project names are never revealed.
		warnings = append(warnings, "Some projects on this page are outside the caller's project grant and were omitted.")
		page.Page.Returned = len(out)
		page.Page.Total = nil
	}
	next := "Pick a project by ID; call " + ToolListFolderItems + " with its root_folder_id to browse files."
	if !page.Page.Complete {
		next = "More projects exist: call " + ToolListProjects + " again with page_token=pagination.next_page_token."
	}
	obs := page.Prov.ObservedAt
	return &Envelope{
		Status: "ok", Product: string(c.product), Operation: ToolListProjects,
		Result: map[string]any{"projects": out}, Pagination: &page.Page,
		Source: &page.Prov, ObservedAt: &obs, Warnings: warnings,
		DataLabels: labels(a), UntrustedFields: []string{"projects[].name", "projects[].description"},
		NextAction: next,
	}, nil
}

func (g *Gateway) getProject(ctx context.Context, c *call, args json.RawMessage) (*Envelope, error) {
	var in struct {
		Product   string `json:"product"`
		ProjectID string `json:"project_id"`
	}
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	pid, err := domain.ParseProjectID(in.ProjectID)
	if err != nil {
		return nil, err
	}
	c.project, c.resource = pid, "project/"+string(pid)
	a, err := g.resolveProduct(c, in.Product)
	if err != nil {
		return nil, err
	}
	r, ok := a.(trimble.ProjectReader)
	if !ok || !a.Describe().Supports(trimble.CapGetProject) {
		return nil, errs.New(errs.UnsupportedCapability)
	}
	p, prov, err := r.GetProject(ctx, pid)
	if err != nil {
		return nil, err
	}
	obs := prov.ObservedAt
	return &Envelope{
		Status: "ok", Product: string(c.product), Operation: ToolGetProject, Resource: c.resource,
		Result: map[string]any{"project": cleanProject(p)}, Source: &prov, ObservedAt: &obs,
		DataLabels: labels(a), UntrustedFields: []string{"project.name", "project.description"},
		NextAction: "Call " + ToolListFolderItems + " with root_folder_id to browse files.",
	}, nil
}

// ---- files ----

func (g *Gateway) fileReader(c *call, product, project string) (trimble.FileReader, trimble.Base, error) {
	pid, err := domain.ParseProjectID(project)
	if err != nil {
		return nil, nil, err
	}
	c.project = pid
	a, err := g.resolveProduct(c, product)
	if err != nil {
		return nil, nil, err
	}
	r, ok := a.(trimble.FileReader)
	if !ok {
		return nil, nil, errs.New(errs.UnsupportedCapability)
	}
	return r, a, nil
}

func (g *Gateway) listFolderItems(ctx context.Context, c *call, args json.RawMessage) (*Envelope, error) {
	var in struct {
		Product   string `json:"product"`
		ProjectID string `json:"project_id"`
		FolderID  string `json:"folder_id"`
		PageSize  int    `json:"page_size"`
		PageToken string `json:"page_token"`
	}
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	fid, err := domain.ParseFolderID(in.FolderID)
	if err != nil {
		return nil, err
	}
	c.resource = "folder/" + string(fid)
	r, a, err := g.fileReader(c, in.Product, in.ProjectID)
	if err != nil {
		return nil, err
	}
	if !a.Describe().Supports(trimble.CapListFolder) {
		return nil, errs.New(errs.UnsupportedCapability)
	}
	page, err := r.ListFolderItems(ctx, c.project, fid, domain.PageRequest{Token: in.PageToken, Size: in.PageSize})
	if err != nil {
		return nil, err
	}
	for i := range page.Items {
		page.Items[i].Name = cleanUntrusted(page.Items[i].Name)
	}
	var warnings []string
	if len(page.Items) == 0 {
		warnings = append(warnings, "The folder returned no items; an empty result does not confirm the folder exists in this project.")
	}
	next := "Use items[].id with kind=folder to descend, or " + ToolGetFileMetadata + " for a file."
	if !page.Page.Complete {
		next = "More items exist: call " + ToolListFolderItems + " again with page_token=pagination.next_page_token."
	}
	obs := page.Prov.ObservedAt
	return &Envelope{
		Status: "ok", Product: string(c.product), Operation: ToolListFolderItems, Resource: c.resource,
		Result: map[string]any{"items": page.Items}, Pagination: &page.Page, Units: "size_bytes: bytes",
		Source: &page.Prov, ObservedAt: &obs, Warnings: warnings,
		DataLabels: labels(a), UntrustedFields: []string{"items[].name"},
		NextAction: next,
	}, nil
}

func (g *Gateway) getFileMetadata(ctx context.Context, c *call, args json.RawMessage) (*Envelope, error) {
	var in struct {
		Product   string `json:"product"`
		ProjectID string `json:"project_id"`
		FileID    string `json:"file_id"`
	}
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	fid, err := domain.ParseFileID(in.FileID)
	if err != nil {
		return nil, err
	}
	c.resource = "file/" + string(fid)
	r, a, err := g.fileReader(c, in.Product, in.ProjectID)
	if err != nil {
		return nil, err
	}
	if !a.Describe().Supports(trimble.CapFileMetadata) {
		return nil, errs.New(errs.UnsupportedCapability)
	}
	md, prov, err := r.GetFileMetadata(ctx, c.project, fid)
	if err != nil {
		return nil, err
	}
	md.Name = cleanUntrusted(md.Name)
	var warnings []string
	if md.Checksum != "" {
		warnings = append(warnings, "checksum is the upstream-reported hash; its algorithm is not documented, so do not use it for integrity verification.")
	}
	obs := prov.ObservedAt
	return &Envelope{
		Status: "ok", Product: string(c.product), Operation: ToolGetFileMetadata, Resource: c.resource,
		Result: map[string]any{"file": md}, Units: "size_bytes: bytes; timestamps: UTC",
		Source: &prov, ObservedAt: &obs, Warnings: warnings,
		DataLabels: labels(a), UntrustedFields: []string{"file.name"},
		NextAction: "Content download is not available in this release.",
	}, nil
}

func labels(a trimble.Base) []string {
	if a.Describe().Status == trimble.Simulated {
		return []string{"simulated_data", "not_certified"}
	}
	return []string{"observed_data", "not_certified"}
}
