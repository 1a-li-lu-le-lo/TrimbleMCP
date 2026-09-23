package domain

import (
	"math"
	"regexp"

	"github.com/1a-li-lu-le-lo/trimblemcp/internal/errs"
)

// CRS identifies a coordinate reference system by authority and code, e.g.
// EPSG:4326. It has no zero-value default: an empty CRS is invalid, and the
// bridge never assumes WGS 84 or infers a code from coordinate ranges.
type CRS struct {
	Authority string    `json:"authority"`
	Code      string    `json:"code"`
	AxisOrder AxisOrder `json:"axis_order"`
}

// AxisOrder must be stated explicitly because EPSG:4326 is formally
// latitude-first while many APIs and GeoJSON are longitude-first.
type AxisOrder string

const (
	AxisLatLon   AxisOrder = "lat_lon"
	AxisLonLat   AxisOrder = "lon_lat"
	AxisEastNort AxisOrder = "east_north"
	AxisNortEast AxisOrder = "north_east"
)

var crsCode = regexp.MustCompile(`^[A-Za-z0-9._-]{1,32}$`)

func (c CRS) Validate() error {
	if c.Authority == "" || c.Code == "" {
		return errs.New(errs.CoordinateSystemNeeded)
	}
	if !crsCode.MatchString(c.Authority) || !crsCode.MatchString(c.Code) {
		return errs.Newf(errs.Validation, "crs authority and code must be short alphanumeric identifiers")
	}
	switch c.AxisOrder {
	case AxisLatLon, AxisLonLat, AxisEastNort, AxisNortEast:
	case "":
		return errs.Newf(errs.CoordinateSystemNeeded, "axis_order must be stated explicitly")
	default:
		return errs.Newf(errs.Validation, "unknown axis_order %q", c.AxisOrder)
	}
	return nil
}

func (c CRS) String() string { return c.Authority + ":" + c.Code }

// LengthUnit enumerates supported linear units. International and US survey
// feet are distinct and are never conflated.
type LengthUnit string

const (
	Metre             LengthUnit = "metre"
	Kilometre         LengthUnit = "kilometre"
	InternationalFoot LengthUnit = "international_foot"
	USSurveyFoot      LengthUnit = "us_survey_foot"
	StatuteMile       LengthUnit = "statute_mile"
)

// metresPer holds exact definitions: international foot = 0.3048 m exactly;
// US survey foot = 1200/3937 m exactly (NIST).
var metresPer = map[LengthUnit]float64{
	Metre:             1,
	Kilometre:         1000,
	InternationalFoot: 0.3048,
	USSurveyFoot:      1200.0 / 3937.0,
	StatuteMile:       1609.344,
}

// Distance is a magnitude with an explicit unit.
type Distance struct {
	Value float64    `json:"value"`
	Unit  LengthUnit `json:"unit"`
}

func (d Distance) Validate() error {
	if _, ok := metresPer[d.Unit]; !ok {
		return errs.Newf(errs.Unit, "unknown or missing length unit %q", d.Unit)
	}
	if math.IsNaN(d.Value) || math.IsInf(d.Value, 0) {
		return errs.Newf(errs.Validation, "distance must be finite")
	}
	return nil
}

// Convert returns d expressed in unit. Conversion is exact up to float64
// precision; no geodetic transformation is involved.
func (d Distance) Convert(unit LengthUnit) (Distance, error) {
	if err := d.Validate(); err != nil {
		return Distance{}, err
	}
	to, ok := metresPer[unit]
	if !ok {
		return Distance{}, errs.Newf(errs.Unit, "unknown target unit %q", unit)
	}
	return Distance{Value: d.Value * metresPer[d.Unit] / to, Unit: unit}, nil
}

// GeographicPosition is a position in a geographic (angular) CRS whose
// coordinates are degrees. Callers must supply the CRS; range checks apply
// only because the caller asserted a geographic CRS, not to infer one.
type GeographicPosition struct {
	CRS       CRS     `json:"crs"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	// Height is optional; if present its reference must be stated.
	Height          *float64   `json:"height,omitempty"`
	HeightUnit      LengthUnit `json:"height_unit,omitempty"`
	HeightReference HeightRef  `json:"height_reference,omitempty"`
	Accuracy        *Distance  `json:"horizontal_accuracy,omitempty"`
	Source          string     `json:"source"`
	Observed        *Observed  `json:"observed,omitempty"`
}

// HeightRef distinguishes ellipsoidal from orthometric heights.
type HeightRef string

const (
	HeightEllipsoidal HeightRef = "ellipsoidal"
	HeightOrthometric HeightRef = "orthometric"
)

func (p GeographicPosition) Validate() error {
	if err := p.CRS.Validate(); err != nil {
		return err
	}
	if math.IsNaN(p.Latitude) || p.Latitude < -90 || p.Latitude > 90 {
		return errs.Newf(errs.Validation, "latitude must be within [-90, 90] degrees")
	}
	if math.IsNaN(p.Longitude) || p.Longitude < -180 || p.Longitude > 180 {
		return errs.Newf(errs.Validation, "longitude must be within [-180, 180] degrees")
	}
	if p.Height != nil {
		if _, ok := metresPer[p.HeightUnit]; !ok {
			return errs.Newf(errs.Unit, "height requires an explicit height_unit")
		}
		if p.HeightReference != HeightEllipsoidal && p.HeightReference != HeightOrthometric {
			return errs.Newf(errs.Validation, "height requires height_reference ellipsoidal or orthometric")
		}
	}
	if p.Accuracy != nil {
		if err := p.Accuracy.Validate(); err != nil {
			return err
		}
	}
	if p.Source == "" {
		return errs.Newf(errs.Validation, "position source is required")
	}
	return nil
}
