package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/authz"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/domain"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/errs"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/trimble"
)

// maxVerifyPages bounds how many project pages are scanned to confirm that a
// project ID exists before a desktop link is built or launched.
const maxVerifyPages = 20

type desktopArgs struct {
	Product   string `json:"product"`
	ProjectID string `json:"project_id"`
	View      string `json:"view"`
	Panel     string `json:"panel"`
	DryRun    *bool  `json:"dry_run"`
	Reason    string `json:"reason"`
}

// verification describes whether a project ID was confirmed through an API.
type verification struct {
	Verified bool   `json:"verified"`
	Against  string `json:"against,omitempty"`
	Detail   string `json:"detail"`
	err      error
}

// verifyProject confirms, through the configured project API, that id is a
// project visible to this tenant. It never trusts the caller's word for it.
func (g *Gateway) verifyProject(ctx context.Context, c *call, id domain.ProjectID) verification {
	// Verification reads the project API, so the caller must be allowed to
	// read projects in that product. Otherwise the result would be an
	// existence oracle for a product the caller was never granted.
	if err := authz.Authorize(c.p, authz.Request{
		Tenant: c.p.Tenant, Product: g.desktopProjects, Scope: authz.ScopeProjectsRead, Project: id,
	}, g.now()); err != nil {
		return verification{Detail: fmt.Sprintf("the caller is not authorized to read %s projects, so the project ID was not checked", g.desktopProjects)}
	}
	a, err := g.reg.Resolve(c.p.Tenant, g.desktopProjects)
	if err != nil {
		return verification{Detail: fmt.Sprintf("%s is not configured, so the project ID could not be confirmed", g.desktopProjects)}
	}
	r, ok := a.(trimble.ProjectReader)
	if !ok || !a.Describe().Supports(trimble.CapListProjects) {
		return verification{Detail: fmt.Sprintf("%s cannot list projects, so the project ID could not be confirmed", g.desktopProjects)}
	}
	token := ""
	for page := 0; page < maxVerifyPages; page++ {
		res, err := r.ListProjects(ctx, trimble.ListProjectsQuery{Page: domain.PageRequest{Token: token, Size: domain.MaxPageSize}})
		if err != nil {
			return verification{Against: string(g.desktopProjects), Detail: "project lookup failed: " + string(errs.As(err).Code), err: err}
		}
		for _, p := range res.Projects {
			if p.ID == id {
				return verification{Verified: true, Against: string(g.desktopProjects), Detail: "project ID found via trimble_list_projects"}
			}
		}
		if res.Page.Complete {
			return verification{Against: string(g.desktopProjects), Detail: "project ID not found among projects visible to this tenant",
				err: errs.New(errs.ProjectNotFound)}
		}
		token = res.Page.NextToken
	}
	return verification{Against: string(g.desktopProjects),
		Detail: fmt.Sprintf("project not found in the first %d pages; lookup stopped", maxVerifyPages)}
}

func (g *Gateway) desktopCommon(ctx context.Context, c *call, args json.RawMessage) (desktopArgs, trimble.DesktopLauncher, trimble.DesktopLink, verification, error) {
	var in desktopArgs
	var none trimble.DesktopLink
	if err := decode(args, &in); err != nil {
		return in, nil, none, verification{}, err
	}
	pid, err := domain.ParseProjectID(in.ProjectID)
	if err != nil {
		return in, nil, none, verification{}, err
	}
	c.project, c.resource = pid, "project/"+string(pid)
	a, err := g.resolveProduct(c, in.Product)
	if err != nil {
		return in, nil, none, verification{}, err
	}
	l, ok := a.(trimble.DesktopLauncher)
	if !ok || !a.Describe().Supports(trimble.CapDesktopLink) {
		return in, nil, none, verification{}, errs.New(errs.UnsupportedCapability)
	}
	link, err := l.BuildLink(pid, in.View, in.Panel)
	if err != nil {
		return in, nil, none, verification{}, err
	}
	return in, l, link, g.verifyProject(ctx, c, pid), nil
}

func desktopWarnings(v verification) []string {
	w := []string{"The project ID in trimbleconnect: links is assumed to equal the Trimble Connect REST API project ID (assumption A-8)."}
	if !v.Verified {
		w = append(w, "Project not verified: "+v.Detail+".")
	}
	return w
}

func (g *Gateway) buildDesktopLink(ctx context.Context, c *call, args json.RawMessage) (*Envelope, error) {
	_, _, link, v, err := g.desktopCommon(ctx, c, args)
	if err != nil {
		return nil, err
	}
	if errs.Is(v.err, errs.ProjectNotFound) {
		return nil, v.err
	}
	next := "Give the link to the user to open on a Windows machine with Trimble Connect for Windows installed."
	if g.launchVisible(c) {
		next = "To open it here, call " + ToolOpenDesktop + " with the same arguments and dry_run=false."
	}
	return &Envelope{
		Status: "ok", Product: string(c.product), Operation: ToolBuildDesktop, Resource: c.resource,
		Result:     map[string]any{"link": link, "project_verification": v},
		DataLabels: []string{"local_launch_link", "no_project_data"},
		Warnings:   desktopWarnings(v), NextAction: next,
	}, nil
}

func (g *Gateway) launchVisible(c *call) bool {
	for _, t := range g.tools {
		if t.info.Name == ToolOpenDesktop {
			return g.visible(c.p, t)
		}
	}
	return false
}

func (g *Gateway) openInDesktop(ctx context.Context, c *call, args json.RawMessage) (*Envelope, error) {
	if !c.p.Local {
		// Defence in depth: the tool is hidden from remote principals, and a
		// remote caller must never open applications on the server host.
		return nil, errs.Newf(errs.PolicyDenied, "desktop launch is only available to the local operator session")
	}
	in, l, link, v, err := g.desktopCommon(ctx, c, args)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Reason) == "" || len(in.Reason) > 200 {
		return nil, errs.Newf(errs.Validation, "reason is required (1-200 characters)")
	}
	if !v.Verified {
		if v.err != nil {
			return nil, v.err
		}
		return nil, errs.Newf(errs.PolicyDenied, "the project ID could not be verified (%s); launching is refused", v.Detail)
	}
	dry := in.DryRun == nil || *in.DryRun
	result := map[string]any{"link": link, "project_verification": v, "dry_run": dry, "launched": false,
		"reversible": true, "rollback": "Close the Trimble Connect for Windows window; no project data was changed."}
	if dry {
		return &Envelope{
			Status: "ok", Product: string(c.product), Operation: ToolOpenDesktop, Resource: c.resource,
			Result: result, DataLabels: []string{"dry_run", "no_project_data"}, Warnings: desktopWarnings(v),
			NextAction: "Nothing was opened. If the user confirms, call again with dry_run=false.",
		}, nil
	}
	if ok, wait := g.launches.Allow(string(c.p.Tenant) + "\x00" + c.p.Subject); !ok {
		e := errs.Newf(errs.RateLimited, "too many desktop launches; wait before opening again")
		e.RetryAfter = int(wait.Seconds()) + 1
		return nil, e
	}
	if err := l.Launch(ctx, link); err != nil {
		return nil, err
	}
	result["launched"] = true
	return &Envelope{
		Status: "ok", Product: string(c.product), Operation: ToolOpenDesktop, Resource: c.resource,
		Result: result, DataLabels: []string{"local_side_effect", "no_project_data"}, Warnings: desktopWarnings(v),
		NextAction: "Ask the user to confirm Trimble Connect for Windows opened at the expected project; the bridge cannot observe the application.",
	}, nil
}
