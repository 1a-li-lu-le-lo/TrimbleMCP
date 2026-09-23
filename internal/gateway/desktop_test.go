package gateway

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/audit"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/authz"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/domain"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/errs"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/trimble/desktop"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/trimble/mock"
)

type desktopFixture struct {
	g      *Gateway
	mem    *audit.Memory
	opened *[]string
}

// setupDesktop registers the desktop adapter (launch enabled, fake Windows
// host) and the mock project API used for project verification.
func setupDesktop(t *testing.T, withAPI bool) desktopFixture {
	t.Helper()
	opened := &[]string{}
	reg := NewRegistry()
	reg.Register("tenant-a", desktop.New(desktop.Config{LaunchEnabled: true, GOOS: "windows",
		Open: func(_ context.Context, uri string) error { *opened = append(*opened, uri); return nil }}))
	if withAPI {
		reg.Register("tenant-a", mock.New())
	}
	mem := &audit.Memory{}
	g, err := New(Options{Registry: reg, Audit: mem, CallsPerSecond: 1000, Burst: 1000, DesktopProjectSource: mock.Product})
	if err != nil {
		t.Fatal(err)
	}
	return desktopFixture{g: g, mem: mem, opened: opened}
}

func localOperator(extra ...authz.Scope) *authz.Principal {
	p := principal(append([]authz.Scope{authz.ScopeCapabilitiesRead, authz.ScopeProjectsRead}, extra...)...)
	p.Local = true
	return p
}

func TestDesktopLinkBuiltAndVerified(t *testing.T) {
	f := setupDesktop(t, true)
	env := invoke(t, f.g, localOperator(), ToolBuildDesktop,
		`{"product":"trimble-connect-desktop","project_id":"mock-prj-007","view":"3d","panel":"todos"}`)
	if env.Status != "ok" {
		t.Fatalf("%+v", env.Error)
	}
	b, _ := json.Marshal(env.Result)
	if !strings.Contains(string(b), `"uri":"trimbleconnect:/projects/mock-prj-007?show=3D,ToDos"`) || !strings.Contains(string(b), `"verified":true`) {
		t.Fatalf("%s", b)
	}
	if len(*f.opened) != 0 {
		t.Fatal("building a link must not open anything")
	}
}

func TestDesktopLinkUnknownProjectRejected(t *testing.T) {
	f := setupDesktop(t, true)
	env := invoke(t, f.g, localOperator(), ToolBuildDesktop, `{"product":"trimble-connect-desktop","project_id":"invented-123"}`)
	if env.Error == nil || env.Error.Code != errs.ProjectNotFound {
		t.Fatalf("fabricated project ID must be rejected: %+v", env)
	}
}

func TestDesktopLinkWithoutAPIIsUnverified(t *testing.T) {
	f := setupDesktop(t, false)
	env := invoke(t, f.g, localOperator(), ToolBuildDesktop, `{"product":"trimble-connect-desktop","project_id":"abc"}`)
	b, _ := json.Marshal(env)
	if env.Status != "ok" || !strings.Contains(string(b), `"verified":false`) || !strings.Contains(string(b), "Project not verified") {
		t.Fatalf("%s", b)
	}
}

func TestDesktopLaunchVisibility(t *testing.T) {
	f := setupDesktop(t, true)
	has := func(p *authz.Principal) bool {
		for _, n := range toolNames(f.g, p) {
			if n == ToolOpenDesktop {
				return true
			}
		}
		return false
	}
	if has(localOperator()) {
		t.Error("launch tool visible without trimble:desktop:launch")
	}
	remote := localOperator(authz.ScopeDesktopLaunch)
	remote.Local = false
	if has(remote) {
		t.Error("launch tool visible to a remote principal")
	}
	if !has(localOperator(authz.ScopeDesktopLaunch)) {
		t.Error("launch tool hidden from scoped local operator")
	}
	if _, rpcErr := f.g.CallTool(context.Background(), remote, ToolOpenDesktop,
		json.RawMessage(`{"product":"trimble-connect-desktop","project_id":"mock-prj-001","reason":"x","dry_run":false}`)); rpcErr == nil {
		t.Error("remote principal could call the launch tool")
	}
}

func TestDesktopLaunchDryRunThenLaunch(t *testing.T) {
	f := setupDesktop(t, true)
	p := localOperator(authz.ScopeDesktopLaunch)
	args := `{"product":"trimble-connect-desktop","project_id":"mock-prj-001","view":"3D","panel":"ToDos","reason":"review open ToDos"}`
	env := invoke(t, f.g, p, ToolOpenDesktop, args)
	if env.Status != "ok" || len(*f.opened) != 0 || env.Result.(map[string]any)["dry_run"] != true {
		t.Fatalf("default must be a dry run: %+v %v", env, *f.opened)
	}
	env = invoke(t, f.g, p, ToolOpenDesktop, strings.Replace(args, `"reason"`, `"dry_run":false,"reason"`, 1))
	if env.Status != "ok" || len(*f.opened) != 1 || (*f.opened)[0] != "trimbleconnect:/projects/mock-prj-001?show=3D,ToDos" {
		t.Fatalf("launch: %+v %v", env.Error, *f.opened)
	}
	last := f.mem.Records[len(f.mem.Records)-1]
	if last.Operation != ToolOpenDesktop || last.Project != "mock-prj-001" || last.Result != "ok" {
		t.Fatalf("audit %+v", last)
	}
}

func TestDesktopLaunchRefusals(t *testing.T) {
	f := setupDesktop(t, true)
	p := localOperator(authz.ScopeDesktopLaunch)
	cases := map[string]struct {
		args string
		code errs.Code
	}{
		"no reason":          {`{"product":"trimble-connect-desktop","project_id":"mock-prj-001","dry_run":false}`, errs.Validation},
		"blank reason":       {`{"product":"trimble-connect-desktop","project_id":"mock-prj-001","dry_run":false,"reason":"  "}`, errs.Validation},
		"fabricated project": {`{"product":"trimble-connect-desktop","project_id":"made-up","dry_run":false,"reason":"x"}`, errs.ProjectNotFound},
		"injection":          {`{"product":"trimble-connect-desktop","project_id":"mock-prj-001?show=3D","dry_run":false,"reason":"x"}`, errs.Validation},
		"bad view":           {`{"product":"trimble-connect-desktop","project_id":"mock-prj-001","view":"4D","dry_run":false,"reason":"x"}`, errs.Validation},
		"extra arg":          {`{"product":"trimble-connect-desktop","project_id":"mock-prj-001","uri":"file:///c:/x","dry_run":false,"reason":"x"}`, errs.Validation},
	}
	for name, c := range cases {
		env := invoke(t, f.g, p, ToolOpenDesktop, c.args)
		if env.Error == nil || env.Error.Code != c.code {
			t.Errorf("%s: got %+v", name, env.Error)
		}
	}
	if len(*f.opened) != 0 {
		t.Fatalf("refused calls opened %v", *f.opened)
	}
	// Project grants apply to launching too.
	limited := localOperator(authz.ScopeDesktopLaunch)
	limited.Projects = []domain.ProjectID{"mock-prj-002"}
	env := invoke(t, f.g, limited, ToolOpenDesktop, `{"product":"trimble-connect-desktop","project_id":"mock-prj-001","dry_run":false,"reason":"x"}`)
	if env.Error == nil || env.Error.Code != errs.Authorization || len(*f.opened) != 0 {
		t.Fatalf("grant bypass: %+v", env.Error)
	}
}

func TestDesktopLaunchUnverifiedRefused(t *testing.T) {
	f := setupDesktop(t, false)
	env := invoke(t, f.g, localOperator(authz.ScopeDesktopLaunch), ToolOpenDesktop,
		`{"product":"trimble-connect-desktop","project_id":"abc","dry_run":false,"reason":"x"}`)
	if env.Error == nil || env.Error.Code != errs.PolicyDenied || len(*f.opened) != 0 {
		t.Fatalf("unverified launch: %+v", env.Error)
	}
}

func TestDesktopLaunchRateLimited(t *testing.T) {
	f := setupDesktop(t, true)
	p := localOperator(authz.ScopeDesktopLaunch)
	args := `{"product":"trimble-connect-desktop","project_id":"mock-prj-001","dry_run":false,"reason":"x"}`
	var last *Envelope
	for i := 0; i < 3; i++ {
		last = invoke(t, f.g, p, ToolOpenDesktop, args)
	}
	if last.Error == nil || last.Error.Code != errs.RateLimited || len(*f.opened) != 2 {
		t.Fatalf("launch burst: %+v %d", last.Error, len(*f.opened))
	}
}
