//go:build !unix && !windows

package reuse

func ReuseAddr(fd uintptr) error {
	return ErrReuseAddrNotSupported
}

func ReusePort(fd uintptr) error {
	return ErrReusePortNotSupported
}
