//go:build windows

package util

import (
	"ai-disk-cleanner/backend/model"
	"fmt"
	"syscall"
	"unsafe"
)

const (
	driveRemovable = 2
	driveFixed     = 3
)

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	getLogicalDrives     = kernel32.NewProc("GetLogicalDrives")
	getDriveType         = kernel32.NewProc("GetDriveTypeW")
	getDiskFreeSpaceEx   = kernel32.NewProc("GetDiskFreeSpaceExW")
	getVolumeInformation = kernel32.NewProc("GetVolumeInformationW")
)

func ListDisks() ([]model.DiskInfo, error) {
	mask, _, callErr := getLogicalDrives.Call()
	if mask == 0 {
		return nil, fmt.Errorf("get logical drives: %w", callErr)
	}

	disks := make([]model.DiskInfo, 0, 4)
	for index := 0; index < 26; index++ {
		if mask&(1<<index) == 0 {
			continue
		}

		letter := byte('A' + index)
		path := fmt.Sprintf("%c:\\", letter)
		pathPtr, err := syscall.UTF16PtrFromString(path)
		if err != nil {
			continue
		}

		driveType, _, _ := getDriveType.Call(uintptr(unsafe.Pointer(pathPtr)))
		if driveType != driveFixed && driveType != driveRemovable {
			continue
		}

		var available, total, free uint64
		ok, _, _ := getDiskFreeSpaceEx.Call(
			uintptr(unsafe.Pointer(pathPtr)),
			uintptr(unsafe.Pointer(&available)),
			uintptr(unsafe.Pointer(&total)),
			uintptr(unsafe.Pointer(&free)),
		)
		if ok == 0 {
			continue
		}

		label := volumeLabel(pathPtr)
		if label == "" {
			if driveType == driveRemovable {
				label = "移动磁盘"
			} else {
				label = "本地磁盘"
			}
		}

		disks = append(disks, model.DiskInfo{
			Name:  fmt.Sprintf("%s (%c:)", label, letter),
			Path:  path,
			Total: total,
			Free:  free,
			Used:  total - free,
		})
	}

	return disks, nil
}

func volumeLabel(path *uint16) string {
	buffer := make([]uint16, syscall.MAX_PATH+1)
	ok, _, _ := getVolumeInformation.Call(
		uintptr(unsafe.Pointer(path)),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
		0,
		0,
		0,
		0,
		0,
	)
	if ok == 0 {
		return ""
	}
	return syscall.UTF16ToString(buffer)
}
