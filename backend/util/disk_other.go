//go:build !windows

package util

import (
	"errors"
)

func listDisks() ([]model.DiskInfo, error) {
	return nil, errors.New("disk enumeration is currently only supported on Windows")
}
