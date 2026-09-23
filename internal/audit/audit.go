// Package audit writes an append-only, hash-chained JSON Lines log of every
// tool invocation. Records carry hashes of inputs and outputs, never the raw
// values, so secrets and sensitive payloads cannot leak into the log.
package audit

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Record is one audit event.
type Record struct {
	ID                string    `json:"audit_id"`
	Time              time.Time `json:"time"`
	Actor             string    `json:"actor"`
	Client            string    `json:"client"`
	Tenant            string    `json:"tenant"`
	Product           string    `json:"product,omitempty"`
	Project           string    `json:"project,omitempty"`
	Operation         string    `json:"operation"`
	Resource          string    `json:"resource,omitempty"`
	Scope             string    `json:"scope,omitempty"`
	Approval          string    `json:"approval,omitempty"`
	IdempotencyKey    string    `json:"idempotency_key,omitempty"`
	RequestID         string    `json:"request_id"`
	UpstreamRequestID string    `json:"upstream_request_id,omitempty"`
	InputHash         string    `json:"input_sha256"`
	OutputHash        string    `json:"output_sha256,omitempty"`
	Result            string    `json:"result"` // ok | denied | error
	ErrorCode         string    `json:"error_code,omitempty"`
	DurationMS        int64     `json:"duration_ms"`
	PrevHash          string    `json:"prev_sha256"`
}

// Sink appends records.
type Sink interface {
	Append(r *Record) error
}

// Log is a file- or writer-backed Sink. Each record includes the SHA-256 of
// the previous serialized record so truncation or edits are detectable.
type Log struct {
	mu   sync.Mutex
	w    io.Writer
	prev string
}

// NewLog writes to w.
func NewLog(w io.Writer) *Log { return &Log{w: w, prev: zeroHash} }

const zeroHash = "0000000000000000000000000000000000000000000000000000000000000000"

// OpenFile opens path for append-only writing with owner-only permissions.
func OpenFile(path string) (*Log, *os.File, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return nil, nil, err
	}
	return NewLog(f), f, nil
}

// Append assigns an ID if missing, chains, and writes r as one JSON line.
func (l *Log) Append(r *Record) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if r.ID == "" {
		r.ID = NewID("aud")
	}
	if r.Time.IsZero() {
		r.Time = time.Now().UTC()
	}
	r.PrevHash = l.prev
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(b)
	if _, err := l.w.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("audit write failed: %w", err)
	}
	l.prev = hex.EncodeToString(sum[:])
	return nil
}

// Verify checks the hash chain of a JSON Lines audit stream.
func Verify(r io.Reader) (int, error) {
	dec := json.NewDecoder(r)
	prev := zeroHash
	n := 0
	for dec.More() {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return n, err
		}
		var rec Record
		if err := json.Unmarshal(raw, &rec); err != nil {
			return n, err
		}
		if rec.PrevHash != prev {
			return n, fmt.Errorf("audit chain broken at record %d (%s)", n+1, rec.ID)
		}
		sum := sha256.Sum256(raw)
		prev = hex.EncodeToString(sum[:])
		n++
	}
	return n, nil
}

// Hash returns the hex SHA-256 of v's canonical JSON encoding.
func Hash(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// NewID returns a random identifier with prefix.
func NewID(prefix string) string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}

// Discard is a Sink that drops records; tests only.
type Discard struct{}

func (Discard) Append(*Record) error { return nil }

// Memory collects records; tests only.
type Memory struct {
	mu      sync.Mutex
	Records []Record
}

func (m *Memory) Append(r *Record) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r.ID == "" {
		r.ID = NewID("aud")
	}
	m.Records = append(m.Records, *r)
	return nil
}
