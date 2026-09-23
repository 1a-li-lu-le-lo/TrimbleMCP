package domain

import (
	"math"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/errs"
)

func TestIDValidation(t *testing.T) {
	good := []string{"abc", "prj-001", "9f3b1c2e-aaaa-bbbb-cccc-1234567890ab", "EUx1_:v2"}
	bad := []string{"", "..", ".", "a/b", `a\b`, "a b", "a?b", "a#b", "a%2Fb", "a\x00b", "a\nb", strings.Repeat("x", 257), "a​b"}
	for _, s := range good {
		if _, err := ParseProjectID(s); err != nil {
			t.Errorf("%q rejected: %v", s, err)
		}
	}
	for _, s := range bad {
		if _, err := ParseFileID(s); !errs.Is(err, errs.Validation) {
			t.Errorf("%q accepted", s)
		}
	}
}

func FuzzIDNeverAllowsPathOrQueryBreakout(f *testing.F) {
	for _, s := range []string{"abc", "../x", "a/b", "a?b=c", "%2e%2e", "\x00"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		id, err := ParseFolderID(s)
		if err != nil {
			return
		}
		v := string(id)
		if !utf8.ValidString(v) || strings.ContainsAny(v, "/\\?#% \t\r\n") || v == ".." || v == "." || len(v) > maxIDLen {
			t.Fatalf("unsafe id accepted: %q", v)
		}
	})
}

func TestPageNormalize(t *testing.T) {
	p, err := PageRequest{}.Normalize()
	if err != nil || p.Size != DefaultPageSize {
		t.Fatal(p, err)
	}
	for _, n := range []int{-1, MaxPageSize + 1} {
		if _, err := (PageRequest{Size: n}).Normalize(); err == nil {
			t.Errorf("size %d accepted", n)
		}
	}
	if _, err := (PageRequest{Token: strings.Repeat("a", 600)}).Normalize(); err == nil {
		t.Error("long token accepted")
	}
}

func TestCRSMustBeExplicit(t *testing.T) {
	if err := (CRS{}).Validate(); !errs.Is(err, errs.CoordinateSystemNeeded) {
		t.Fatal("empty CRS accepted")
	}
	if err := (CRS{Authority: "EPSG", Code: "4326"}).Validate(); !errs.Is(err, errs.CoordinateSystemNeeded) {
		t.Fatal("missing axis order accepted")
	}
	if err := (CRS{Authority: "EPSG", Code: "4326", AxisOrder: AxisLatLon}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (CRS{Authority: "EPSG", Code: "43 26", AxisOrder: AxisLatLon}).Validate(); err == nil {
		t.Fatal("malformed code accepted")
	}
}

func TestSurveyFootIsNotInternationalFoot(t *testing.T) {
	d := Distance{Value: 1_000_000, Unit: USSurveyFoot}
	m, err := d.Convert(Metre)
	if err != nil {
		t.Fatal(err)
	}
	intl, _ := Distance{Value: 1_000_000, Unit: InternationalFoot}.Convert(Metre)
	// The definitions differ by 2 ppm: 0.6096012 m over 1,000,000 ft.
	if diff := m.Value - intl.Value; math.Abs(diff-0.6096012) > 1e-6 {
		t.Fatalf("difference %.7f m", diff)
	}
	if math.Abs(m.Value-304800.6096012) > 1e-6 {
		t.Fatalf("us survey ft to m = %.7f", m.Value)
	}
	if _, err := (Distance{Value: 1, Unit: "feet"}).Convert(Metre); !errs.Is(err, errs.Unit) {
		t.Fatal("ambiguous unit accepted")
	}
	if _, err := (Distance{Value: math.NaN(), Unit: Metre}).Convert(Kilometre); err == nil {
		t.Fatal("NaN accepted")
	}
}

func TestGeographicPosition(t *testing.T) {
	crs := CRS{Authority: "EPSG", Code: "4326", AxisOrder: AxisLatLon}
	h := 10.0
	ok := GeographicPosition{CRS: crs, Latitude: 45, Longitude: -122, Source: "test"}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	cases := []GeographicPosition{
		{Latitude: 45, Longitude: -122, Source: "x"},                                                       // no CRS
		{CRS: crs, Latitude: 91, Longitude: 0, Source: "x"},                                                // lat range
		{CRS: crs, Latitude: 0, Longitude: 181, Source: "x"},                                               // lon range
		{CRS: crs, Latitude: math.NaN(), Longitude: 0, Source: "x"},                                        // NaN
		{CRS: crs, Latitude: 0, Longitude: 0, Height: &h, HeightUnit: Metre, Source: "x"},                  // height ref missing
		{CRS: crs, Latitude: 0, Longitude: 0, Height: &h, HeightReference: HeightEllipsoidal, Source: "x"}, // height unit missing
		{CRS: crs, Latitude: 0, Longitude: 0},                                                              // no source
	}
	for i, c := range cases {
		if err := c.Validate(); err == nil {
			t.Errorf("case %d accepted", i)
		}
	}
}
