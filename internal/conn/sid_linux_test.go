//go:build linux

package conn

import "syscall"

// getsid is the session of pid; the syscall package has no Getsid on Linux.
func getsid(pid int) (int, error) {
	sid, _, errno := syscall.Syscall(syscall.SYS_GETSID, uintptr(pid), 0, 0)
	if errno != 0 {
		return 0, errno
	}
	return int(sid), nil
}
