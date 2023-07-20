package event

import (
	"syscall"
)

var (
	// ErrBadFileDescriptor is returned when a bad file descriptor is passed.
	ErrBadFileDescriptor = syscall.EBADF
)

// TemporaryErr checks if an error is temporary such as EAGAIN
func TemporaryErr(err error) bool {
	errno, ok := err.(syscall.Errno)
	if !ok {
		return false
	}
	return errno.Temporary()
}
