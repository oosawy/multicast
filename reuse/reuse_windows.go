package reuse

import "golang.org/x/sys/windows"

func ReuseAddr(fd uintptr) (err error) {
	err = windows.SetsockoptInt(windows.Handle(fd), windows.SOL_SOCKET, windows.SO_REUSEADDR, 1)
	return
}

func ReusePort(fd uintptr) (err error) {
	return ErrReusePortNotSupported
}
