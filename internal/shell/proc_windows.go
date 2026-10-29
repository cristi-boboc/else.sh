//go:build windows

package shell

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// parent returns the parent pid and executable name of pid.
func parent(pid int) (int, string, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0, "", err
	}
	defer windows.CloseHandle(snap)
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	for err = windows.Process32First(snap, &e); err == nil; err = windows.Process32Next(snap, &e) {
		if int(e.ProcessID) == pid {
			return int(e.ParentProcessID), windows.UTF16ToString(e.ExeFile[:]), nil
		}
	}
	return 0, "", fmt.Errorf("process %d not found", pid)
}
