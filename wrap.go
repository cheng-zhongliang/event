package event

import (
	"os"
	"syscall"
)

// NewSignal creates a new signal event.
func NewSignal(signal os.Signal, callback func(fd int, events uint32, arg interface{}), arg interface{}) *Event {
	return New(int(signal.(syscall.Signal)), EvSignal, callback, arg)
}

// NewTimer creates a new timer event.
func NewTimer(callback func(fd int, events uint32, arg interface{}), arg interface{}) *Event {
	return New(-1, EvTimeout, callback, arg)
}

// NewTicker creates a new ticker event.
func NewTicker(callback func(fd int, events uint32, arg interface{}), arg interface{}) *Event {
	return New(-1, EvTimeout|EvPersist, callback, arg)
}
