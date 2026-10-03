//go:build windows

package desktop

import (
	"context"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	shell32        = syscall.NewLazyDLL("shell32.dll")
	ole32          = syscall.NewLazyDLL("ole32.dll")
	shellExecuteW  = shell32.NewProc("ShellExecuteW")
	coInitializeEx = ole32.NewProc("CoInitializeEx")
	coUninitialize = ole32.NewProc("CoUninitialize")
)

const (
	coinitApartmentThreaded = 0x2
	coinitDisableOLE1DDE    = 0x4
	sOK                     = 0
	sFalse                  = 1
	swShowNormal            = 1
)

// platformOpen hands the URI to the registered protocol handler through the
// documented ShellExecuteW API, after initialising COM as Microsoft
// recommends (COINIT_APARTMENTTHREADED | COINIT_DISABLE_OLE1DDE). No command
// interpreter is involved, lpParameters and lpDirectory are NULL, and
// BuildURI has already restricted the URI to a fixed prefix, an alphanumeric
// project ID, and enumerated values, so it contains no spaces, quotes, or
// backslashes that a handler's command line could reinterpret.
// Sources: https://learn.microsoft.com/en-us/windows/win32/api/shellapi/nf-shellapi-shellexecutew
func platformOpen(ctx context.Context, uri string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := coInitializeEx.Call(0, coinitApartmentThreaded|coinitDisableOLE1DDE)
	if hr == sOK || hr == sFalse {
		defer coUninitialize.Call()
	}
	// Any other HRESULT (e.g. RPC_E_CHANGED_MODE) means COM is already
	// initialised differently on this thread; ShellExecuteW still works.
	verb, err := syscall.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	file, err := syscall.UTF16PtrFromString(uri)
	if err != nil {
		return err
	}
	r, _, _ := shellExecuteW.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(file)), 0, 0, swShowNormal)
	// ShellExecuteW returns a value greater than 32 on success; 31
	// (SE_ERR_NOASSOC) means no handler is registered for trimbleconnect:.
	if r <= 32 {
		if r == 31 {
			return fmt.Errorf("no application is registered for trimbleconnect: (code %d)", r)
		}
		return fmt.Errorf("ShellExecuteW failed with code %d", r)
	}
	return nil
}
