// Package audit writes an append-only, hash-chained JSON Lines log of every
// tool invocation. Records carry hashes of inputs and outputs, never the raw
// values, so secrets and sensitive payloads cannot leak into the log.
package audit

import (
	"bytes"
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
	// Writer identifies the process instance that appended the record.
	Writer   string `json:"writer"`
	PrevHash string `json:"prev_sha256"`
}

// Sink appends records.
type Sink interface {
	Append(r *Record) error
}

// Log is a file- or writer-backed Sink.
//
// Each process appending to a log is a separate writer with its own hash
// chain: every record carries the SHA-256 of that writer's previous record,
// and a writer's first record carries the hash of the log's last line when
// it opened the file (its anchor). Several processes, such as trimble-mcp
// and trimblectl, can therefore share one log, and restarts do not break
// verification. Each record is written with a single append.
//
// Verify detects edited, reordered, or removed records wherever a later
// record depends on them. Removing records from the very end of the log (or
// the final records of a writer that nothing anchors to) cannot be detected
// from the file alone; ship logs to append-only storage for that.
type Log struct {
	mu     sync.Mutex
	w      io.Writer
	prev   string
	writer string
}

// NewLog writes a fresh chain to w.
func NewLog(w io.Writer) *Log { return &Log{w: w, prev: zeroHash, writer: NewID("w")} }

const zeroHash = "0000000000000000000000000000000000000000000000000000000000000000"

// OpenFile opens path for append-only writing with owner-only permissions
// and anchors this writer's chain to the file's current last record.
func OpenFile(path string) (*Log, *os.File, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return nil, nil, err
	}
	anchor, err := lastLineHash(f)
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	l := NewLog(f)
	l.prev = anchor
	return l, f, nil
}

// lastLineHash returns the SHA-256 of the last complete line of f, or the
// zero hash for an empty file.
func lastLineHash(f *os.File) (string, error) {
	fi, err := f.Stat()
	if err != nil {
		return "", err
	}
	size := fi.Size()
	if size == 0 {
		return zeroHash, nil
	}
	const window = 1 << 20
	start := max(size-window, 0)
	buf := make([]byte, size-start)
	if _, err := f.ReadAt(buf, start); err != nil && err != io.EOF {
		return "", err
	}
	buf = bytes.TrimRight(buf, "\n")
	if i := bytes.LastIndexByte(buf, '\n'); i >= 0 {
		buf = buf[i+1:]
	} else if start > 0 {
		return "", fmt.Errorf("audit log last record exceeds %d bytes", window)
	}
	sum := sha256.Sum256(buf)
	return hex.EncodeToString(sum[:]), nil
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
	r.Writer = l.writer
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

// Verify checks every writer's hash chain in a JSON Lines audit stream and
// returns the number of records.
func Verify(r io.Reader) (int, error) {
	dec := json.NewDecoder(r)
	seen := map[string]bool{zeroHash: true}
	last := map[string]string{}
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
		if prev, ok := last[rec.Writer]; ok {
			if rec.PrevHash != prev {
				return n, fmt.Errorf("audit chain broken at record %d (%s): previous record of writer %s missing or altered", n+1, rec.ID, rec.Writer)
			}
		} else if !seen[rec.PrevHash] {
			return n, fmt.Errorf("audit chain broken at record %d (%s): writer %s is anchored to a missing or altered record", n+1, rec.ID, rec.Writer)
		}
		sum := sha256.Sum256(raw)
		h := hex.EncodeToString(sum[:])
		seen[h] = true
		last[rec.Writer] = h
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
