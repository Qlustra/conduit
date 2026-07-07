//go:build unix

package layout

import (
	"errors"
	"syscall"
)

func isCrossDeviceRenameError(err error) bool {
	return errors.Is(err, syscall.EXDEV)
}
