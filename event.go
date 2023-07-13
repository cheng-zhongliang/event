package event

import (
	"container/list"
	"unsafe"
)

const (
	// EvRead is readable event.
	EvRead = 0x01
	// EvWrite is writable event.
	EvWrite = 0x02

	// EvListInserted is the flag to indicate the event is in the event list.
	EvListInserted = 0x01
	// EvListActive is the flag to indicate the event is in the active event list.
	EvListActive = 0x02
)

type Event struct {
	// Ele is the element in the total event list.
	Ele *list.Element
	// ActiveEle is the element in the active event list.
	ActiveEle *list.Element

	// Fd is the file descriptor to watch.
	Fd int
	// Events is the events to watch. It can be EvRead or EvWrite.
	Events uint32

	// Cb is the callback function when the event is triggered.
	Cb func(fd int, events uint32, arg unsafe.Pointer)
	// Arg is the argument passed to the callback function.
	Arg unsafe.Pointer

	// Res is the result passed to the callback function.
	Res uint32
	// Flags is the status of the event in the event list. It can be EvListInserted or EvListActive.
	Flags int
}

// New creates a new event.
func New(fd int, events uint32, callback func(fd int, events uint32, arg unsafe.Pointer), arg unsafe.Pointer) *Event {
	return &Event{
		Fd:     fd,
		Events: events,
		Cb:     callback,
		Arg:    arg,
	}
}
