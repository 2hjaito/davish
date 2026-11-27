//go:build !linux && !windows

package conn

import "syscall"

func getsid(pid int) (int, error) { return syscall.Getsid(pid) }
