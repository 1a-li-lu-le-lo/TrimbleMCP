//go:build !windows

package desktop

import (
	"context"
	"errors"
)

// platformOpen is unavailable off Windows: Trimble Connect for Windows and
// its trimbleconnect: protocol handler exist only there.
func platformOpen(context.Context, string) error {
	return errors.New("trimbleconnect: links can only be opened on Windows")
}
