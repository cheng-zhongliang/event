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

type Base struct {
	// Poller is the event poller to watch events.
	Poller *Epoll
	// EvList is the list of all events.
	EvList *list.List
	// ActiveEvList is the list of active events.
	ActiveEvList *list.List
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

// NewBase creates a new event base.
func NewBase() (*Base, error) {
	poller, err := NewEpoll()
	if err != nil {
		return nil, err
	}

	return &Base{
		Poller:       poller,
		EvList:       list.New(),
		ActiveEvList: list.New(),
	}, nil
}

// AddEvent adds an event to the event base.
func (bs *Base) AddEvent(ev *Event) error {
	if ev.Flags&(EvListInserted|EvListActive) != 0 {
		return ErrEventAlreadyAdded
	}

	ev.Ele = bs.EvList.PushBack(ev)

	return bs.Poller.Add(ev)
}

// DelEvent deletes an event from the event base.
func (bs *Base) DelEvent(ev *Event) error {
	if ev.Flags&(EvListInserted|EvListActive) == 0 {
		return ErrEventNotAdded
	}

	if ev.Flags&EvListActive != 0 {
		bs.ActiveEvList.Remove(ev.ActiveEle)
		ev.ActiveEle = nil
		ev.Flags &^= EvListActive
	}

	if ev.Flags&EvListInserted != 0 {
		bs.EvList.Remove(ev.Ele)
		ev.Ele = nil
		ev.Flags &^= EvListInserted
	}

	return bs.Poller.Del(ev)
}

// OnEvent adds an event to the active event list.
func (bs *Base) OnEvent(ev *Event, res uint32) {
	if ev.Flags&EvListActive != 0 {
		ev.Res |= res
		return
	}

	ev.Res = res
	ev.Flags |= EvListActive
	ev.ActiveEle = bs.ActiveEvList.PushBack(ev)
}

// Dispatch waits for events and dispatches them.
func (bs *Base) Dispatch() error {
	for {
		if err := bs.Poller.Polling(bs.OnEvent); err != nil {
			return err
		}

		for e := bs.ActiveEvList.Front(); e != nil; e = e.Next() {
			bs.ActiveEvList.Remove(e)
			ev := e.Value.(*Event)
			ev.Flags &^= EvListActive
			ev.ActiveEle = nil
			ev.Cb(ev.Fd, ev.Res, ev.Arg)
		}
	}
}

// Exit closes the event base.
func (bs *Base) Exit() error {
	return bs.Poller.Close()
}
