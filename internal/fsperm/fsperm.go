// Package fsperm checks that files holding secrets are private to their
// owner.
//
// On Unix-like systems that means no group or other permission bits. On
// Windows, Go synthesises a mode of 0666 or 0444 for every file
// (os/types_windows.go), so mode bits carry no access-control information;
// privacy there comes from NTFS ACLs, such as the per-user protection of
// %LOCALAPPDATA%. The check is therefore skipped on Windows, and operators
// must keep secret files in a user-private directory
// (see docs/operations/configuration.md).
package fsperm

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

// CheckPrivate returns an error when path is missing or, on systems with
// meaningful mode bits, readable or writable by group or others.
func CheckPrivate(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	return check(runtime.GOOS, filepath.Base(path), fi.Mode())
}

// Private reports whether mode is acceptable for a secret file on goos.
func Private(goos string, mode fs.FileMode) bool {
	return check(goos, "", mode) == nil
}

func check(goos, name string, mode fs.FileMode) error {
	if goos == "windows" {
		return nil
	}
	if mode.Perm()&0o077 != 0 {
		return fmt.Errorf("%s must not be accessible by group or others (chmod 600)", name)
	}
	return nil
}
