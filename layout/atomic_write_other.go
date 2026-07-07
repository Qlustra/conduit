//go:build !unix

package layout

func isCrossDeviceRenameError(error) bool {
	return false
}
