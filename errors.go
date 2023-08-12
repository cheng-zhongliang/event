package event

import (
	"errors"
	"syscall"
)

var (
	ErrBadFileDescriptor = syscall.EBADF
	ErrEventExists       = errors.New("event exists")
	ErrEventNotExists    = errors.New("event does not exist")
)

func temporaryErr(err error) bool {
	errno, ok := err.(syscall.Errno)
	if !ok {
		return false
	}
	return errno.Temporary()
}
