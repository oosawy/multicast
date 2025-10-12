package reuse

import "errors"

var (
	ErrReuseAddrNotSupported = errors.New("reuseaddr: not supported on this platform")
	ErrReusePortNotSupported = errors.New("reuseport: not supported on this platform")
)
