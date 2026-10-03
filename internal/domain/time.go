package domain

import "time"

// Observed records when and by which clock a value was produced. All instants
// are UTC; the source time zone, when known, is kept separately so local time
// can be reconstructed without guessing across daylight-saving transitions.
type Observed struct {
	At          time.Time `json:"at"`
	SourceClock string    `json:"source_clock"`
	SourceTZ    string    `json:"source_time_zone,omitempty"`
}

// Provenance describes where a returned value came from.
type Provenance struct {
	Product           Product   `json:"product"`
	APIVersion        string    `json:"api_version"`
	Region            Region    `json:"region,omitempty"`
	Source            string    `json:"source"`
	ObservedAt        time.Time `json:"observed_at"`
	UpstreamRequestID string    `json:"upstream_request_id,omitempty"`
	Cached            bool      `json:"cached"`
}

// UTC returns t converted to UTC, or the zero time unchanged.
func UTC(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return t.UTC()
}
