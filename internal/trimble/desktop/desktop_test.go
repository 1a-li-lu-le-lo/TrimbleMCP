package desktop

import (
	"context"
	"errors"
	"testing"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/domain"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/errs"
	"github.com/1a-li-lu-le-lo/trimblemcp/internal/trimble"
)

func TestBuildURIDocumentedExample(t *testing.T) {
	// The example on the official page.
	l, err := BuildURI("rQR1yhTGj9I", "3D", "ToDos")
	if err != nil || l.URI != "trimbleconnect:/projects/rQR1yhTGj9I?show=3D,ToDos" {
		t.Fatalf("%q %v", l.URI, err)
	}
}

func TestBuildURIVariants(t *testing.T) {
	cases := []struct{ view, panel, want string }{
		{"", "", "trimbleconnect:/projects/p1?show=3D,models"}, // documented default
		{"data", "objects", "trimbleconnect:/projects/p1?show=data,objects"},
		{"3d", "todos", "trimbleconnect:/projects/p1?show=3D,ToDos"}, // case-insensitive, documented spelling out
		{"PROJECTS", "VIEWS", "trimbleconnect:/projects/p1?show=projects,views"},
		{"3D", "Clashes", "trimbleconnect:/projects/p1?show=3D,clashes"},
	}
	for _, c := range cases {
		l, err := BuildURI("p1", c.view, c.panel)
		if err != nil || l.URI != c.want {
			t.Errorf("%q/%q: got %q %v", c.view, c.panel, l.URI, err)
		}
	}
}

func TestBuildURIRejectsInjectionAndUndocumentedValues(t *testing.T) {
	bad := []struct {
		project     domain.ProjectID
		view, panel string
	}{
		{"", "", ""},
		{"p1?show=3D", "", ""},
		{"p1&x=1", "", ""},
		{"../settings", "", ""},
		{"p1 /c calc", "", ""},
		{"p1\"", "", ""},
		{"p1%20", "", ""},
		{"p\u00e91", "", ""},
		{"p1", "4D", ""},
		{"p1", "3D", "clash"},
		{"p1", "3D,ToDos", ""},
		{"p1", "", "ToDos"}, // single-value forms are undocumented
		{"p1", "data", ""},
		{"p1", "3D", "ToDos&cmd"},
	}
	for _, b := range bad {
		if _, err := BuildURI(b.project, b.view, b.panel); !errs.Is(err, errs.Validation) {
			t.Errorf("accepted %+v", b)
		}
	}
}

func FuzzBuildURIOnlyEmitsSafeURIs(f *testing.F) {
	f.Add("rQR1yhTGj9I", "3D", "ToDos")
	f.Add("a&b", "3d,x", "")
	f.Fuzz(func(t *testing.T, p, v, pn string) {
		l, err := BuildURI(domain.ProjectID(p), v, pn)
		if err != nil {
			return
		}
		for _, r := range l.URI {
			ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == ':' || r == '/' || r == '?' || r == '=' || r == ',' || r == '-' || r == '_'
			if !ok {
				t.Fatalf("unsafe character %q in %q", r, l.URI)
			}
		}
	})
}

func TestLaunchGating(t *testing.T) {
	var opened []string
	open := func(_ context.Context, uri string) error { opened = append(opened, uri); return nil }
	link, _ := BuildURI("p1", "3D", "views")
	if link.URI != "trimbleconnect:/projects/p1?show=3D,views" {
		t.Fatal(link.URI)
	}

	off := New(Config{LaunchEnabled: false, GOOS: "windows", Open: open})
	if off.Describe().Supports(trimble.CapDesktopLaunch) || off.Launch(context.Background(), link) == nil {
		t.Fatal("launch must be disabled by default")
	}
	linux := New(Config{LaunchEnabled: true, GOOS: "linux", Open: open})
	if linux.Describe().Supports(trimble.CapDesktopLaunch) || linux.Launch(context.Background(), link) == nil {
		t.Fatal("launch must be unavailable off Windows")
	}
	win := New(Config{LaunchEnabled: true, GOOS: "windows", Open: open})
	if !win.Describe().Supports(trimble.CapDesktopLaunch) {
		t.Fatal("launch capability missing")
	}
	if err := win.Launch(context.Background(), link); err != nil || len(opened) != 1 || opened[0] != link.URI {
		t.Fatalf("launch: %v %v", err, opened)
	}
	tampered := link
	tampered.URI = "trimbleconnect:/projects/other"
	if err := win.Launch(context.Background(), tampered); err == nil || len(opened) != 1 {
		t.Fatal("a URI that does not match its parts must not be opened")
	}
	failing := New(Config{LaunchEnabled: true, GOOS: "windows", Open: func(context.Context, string) error { return errors.New("no handler") }})
	err := failing.Launch(context.Background(), link)
	if !errs.Is(err, errs.UpstreamUnavailable) || errs.As(err).Message == "no handler" {
		t.Fatalf("launch failure mapping: %v", err)
	}
}

func TestDescriptor(t *testing.T) {
	d := New(Config{}).Describe()
	if d.Product != Product || len(d.Sources) != 1 || d.Sources[0] != Source || !d.ReadOnly || !d.Supports(trimble.CapDesktopLink) {
		t.Fatalf("%+v", d)
	}
}
