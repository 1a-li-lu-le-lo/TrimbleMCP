package audit

import (
	"bytes"
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
}
