package event

import (
	"errors"
	"syscall"
)

var (
	// ErrEventAlreadyAdded is returned when an event is already added.
	ErrEventAlreadyAdded = errors.New("event already added")
	// ErrEventNotAdded is returned when an event is not added.
	ErrEventNotAdded = errors.New("event not added")
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
