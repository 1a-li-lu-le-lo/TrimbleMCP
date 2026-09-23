//go:build windows

package desktop

import (
	"context"
	"fmt"
	"syscall"
	"unsafe"
)

var (
	shell32       = syscall.NewLazyDLL("shell32.dll")
	shellExecuteW = shell32.NewProc("ShellExecuteW")
)

// platformOpen hands the URI to the registered protocol handler through the
// documented ShellExecuteW API. No command interpreter is involved, so shell
// metacharacters have no meaning; BuildURI has already restricted the URI to
// a fixed prefix, an alphanumeric project ID, and enumerated parameters.
func platformOpen(ctx context.Context, uri string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	verb, err := syscall.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	file, err := syscall.UTF16PtrFromString(uri)
	if err != nil {
		return err
	}
	const swShowNormal = 1
	r, _, _ := shellExecuteW.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(file)), 0, 0, swShowNormal)
	// ShellExecuteW returns a value greater than 32 on success.
	if r <= 32 {
		return fmt.Errorf("ShellExecuteW failed with code %d", r)
	}
	return nil
}
