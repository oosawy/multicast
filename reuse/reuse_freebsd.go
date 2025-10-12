//go:build freebsd

package reuse

import (
	"golang.org/x/sys/unix"
)

func ReuseAddr(fd uintptr) (err error) {
	err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
	return
}

func ReusePort(fd uintptr) (err error) {
	err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
	if err != nil {
		return
	}
	err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT_LB, 1)
	return
}
