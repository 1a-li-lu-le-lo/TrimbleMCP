package fsperm

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPrivate(t *testing.T) {
	if !Private("linux", 0o600) || Private("linux", 0o644) || Private("darwin", 0o640) {
		t.Fatal("unix mode checks wrong")
	}
	// Go reports 0666/0444 for every Windows file; the check must not
	// reject them, or no secret file could ever be loaded on Windows.
	if !Private("windows", 0o666) || !Private("windows", 0o444) {
		t.Fatal("windows files must not be rejected on synthetic mode bits")
	}
}

func TestCheckPrivateFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CheckPrivate(p); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		os.Chmod(p, 0o644)
		if CheckPrivate(p) == nil {
			t.Fatal("world-readable file accepted")
		}
	}
	if CheckPrivate(p+".missing") == nil {
		t.Fatal("missing file accepted")
	}
}
