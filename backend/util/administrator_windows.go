//go:build windows

package util

import "golang.org/x/sys/windows"

func IsRunningAsAdministrator() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}
