//go:build !windows

package util

func isRunningAsAdministrator() bool {
	return true
}
