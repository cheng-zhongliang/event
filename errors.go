package event

import (
	"syscall"
)

var (
	ErrBadFileDescriptor = syscall.EBADF
)

func temporaryErr(err error) bool {
	errno, ok := err.(syscall.Errno)
	if !ok {
		return false
	}
	return errno.Temporary()
}
