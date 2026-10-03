package audit

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChainAndTamperDetection(t *testing.T) {
	var buf bytes.Buffer
	l := NewLog(&buf)
	for i := 0; i < 3; i++ {
		if err := l.Append(&Record{Actor: "a", Operation: "op", RequestID: "r", InputHash: Hash(map[string]string{"token": "secret-value"}), Result: "ok"}); err != nil {
			t.Fatal(err)
		}
	}
	if strings.Contains(buf.String(), "secret-value") {
		t.Fatal("raw input stored in audit log")
	}
	n, err := Verify(bytes.NewReader(buf.Bytes()))
	if err != nil || n != 3 {
		t.Fatalf("verify %d %v", n, err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	tampered := strings.Join([]string{lines[0], strings.Replace(lines[1], `"result":"ok"`, `"result":"denied"`, 1), lines[2]}, "\n")
	if _, err := Verify(strings.NewReader(tampered)); err == nil {
		t.Fatal("edit not detected")
	}
	if _, err := Verify(strings.NewReader(lines[0] + "\n" + lines[2])); err == nil {
		t.Fatal("deletion not detected")
	}
	// Truncating the tail cannot be detected from the file alone (documented).
	if _, err := Verify(strings.NewReader(lines[0] + "\n" + lines[1])); err != nil {
		t.Fatalf("tail truncation unexpectedly flagged: %v", err)
	}
}

// Regression: a restarted process (or a second process) appending to the
// same file must not break verification (security audit finding H2).
func TestReopenAndSharedWriters(t *testing.T) {
	p := filepath.Join(t.TempDir(), "audit.jsonl")
	for run := 0; run < 3; run++ {
		l, f, err := OpenFile(p)
		if err != nil {
			t.Fatal(err)
		}
		l.Append(&Record{Operation: "op", RequestID: "r"})
		l.Append(&Record{Operation: "op", RequestID: "r"})
		f.Close()
	}
	// Two writers interleaving in one file.
	a, fa, _ := OpenFile(p)
	b, fb, _ := OpenFile(p)
	a.Append(&Record{Operation: "a1"})
	b.Append(&Record{Operation: "b1"})
	a.Append(&Record{Operation: "a2"})
	b.Append(&Record{Operation: "b2"})
	fa.Close()
	fb.Close()
	data, _ := os.ReadFile(p)
	n, err := Verify(bytes.NewReader(data))
	if err != nil || n != 10 {
		t.Fatalf("verify %d %v", n, err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	// Removing a middle record is detected.
	cut := strings.Join(append(append([]string{}, lines[:3]...), lines[4:]...), "\n")
	if _, err := Verify(strings.NewReader(cut)); err == nil {
		t.Fatal("middle deletion not detected")
	}
	// Editing an anchor record is detected.
	edited := append([]string{}, lines...)
	edited[1] = strings.Replace(edited[1], `"operation":"op"`, `"operation":"xx"`, 1)
	if _, err := Verify(strings.NewReader(strings.Join(edited, "\n"))); err == nil {
		t.Fatal("edit not detected")
	}
}
