package event

import (
	"container/list"
	"time"
)

const (
	// EvRead is readable event.
	EvRead = 0x01
	// EvWrite is writable event.
	EvWrite = 0x02
	// EvTimeout is timeout event
	EvTimeout = 0x04

	// EvListInserted is the flag to indicate the event is in the event list.
	EvListInserted = 0x01
	// EvListActive is the flag to indicate the event is in the active event list.
	EvListActive = 0x02
)

// Event is the event to watch.
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
	Cb func(fd int, events uint32, arg interface{})
	// Arg is the argument passed to the callback function.
	Arg interface{}

	// Res is the result passed to the callback function.
	Res uint32
	// Flags is the status of the event in the event list. It can be EvListInserted or EvListActive.
	Flags int

	// Timeout is the timeout in milliseconds.
	Timeout int64
	// Deadline is the deadline in milliseconds.
	Deadline int64
}

// EventBase is the base of all events.
type EventBase struct {
	// Poller is the event poller to watch events.
	Poller *Epoll
	// EvList is the list of all events.
	EvList *list.List
	// ActiveEvList is the list of active events.
	ActiveEvList *list.List
	// EventHeap is the min heap of timeout events.
	EvHeap *EventHeap
}

// New creates a new event.
func New(fd int, events uint32, callback func(fd int, events uint32, arg interface{}), arg interface{}) *Event {
	return &Event{
		Fd:     fd,
		Events: events,
		Cb:     callback,
		Arg:    arg,
	}
}

// NewBase creates a new event base.
func NewBase() (*EventBase, error) {
	poller, err := NewEpoll()
	if err != nil {
		return nil, err
	}

	return &EventBase{
		Poller:       poller,
		EvList:       list.New(),
		ActiveEvList: list.New(),
		EvHeap:       NewEventHeap(),
	}, nil
}

// AddEvent adds an event to the event base.
func (bs *EventBase) AddEvent(ev *Event, timeout time.Duration) error {
	if ev.Flags&EvListInserted != 0 {
		return ErrEventAlreadyAdded
	}

	if ev.Events&EvTimeout != 0 {
		ev.Timeout = timeout.Milliseconds()
		ev.Deadline = time.Now().Add(timeout).UnixMilli()
		bs.EvHeap.PushEvent(ev)
	}

	bs.EventListInsert(ev, EvListInserted)

	return bs.Poller.Add(ev)
}

// DelEvent deletes an event from the event base.
func (bs *EventBase) DelEvent(ev *Event) error {
	if ev.Flags&EvListInserted == 0 {
		return ErrEventNotAdded
	}

	if ev.Flags&EvListActive != 0 {
		bs.EventListRemove(ev, EvListActive)
	}

	if ev.Flags&EvListInserted != 0 {
		bs.EventListRemove(ev, EvListInserted)
	}

	return bs.Poller.Del(ev)
}

// Dispatch waits for events and dispatches them.
func (bs *EventBase) Dispatch() error {
	for {
		err := bs.Poller.Polling(bs.OnActive, bs.WaitTime())
		if err != nil {
			return err
		}

		bs.OnTimeout()

		bs.HandleActiveEvents()
	}
}

// Exit closes the event base.
func (bs *EventBase) Exit() error {
	return bs.Poller.Close()
}

// WaitTime returns the time to wait for events.
func (bs *EventBase) WaitTime() int {
	if !bs.EvHeap.Empty() {
		now := time.Now().UnixMilli()
		ev := bs.EvHeap.PeekEvent()
		if ev.Deadline > now {
			return int(ev.Deadline - now)
		}
	}

	return -1
}

// OnTimeout handles timeout events.
func (bs *EventBase) OnTimeout() {
	now := time.Now().UnixMilli()
	for !bs.EvHeap.Empty() {
		ev := bs.EvHeap.PeekEvent()
		if ev.Deadline > now {
			break
		}
		bs.EvHeap.PopEvent()
		bs.OnActive(ev, EvTimeout)
	}
}

// OnEvent adds an event to the active event list.
func (bs *EventBase) OnActive(ev *Event, res uint32) {
	if ev.Flags&EvListActive != 0 {
		ev.Res |= res
		return
	}

	ev.Res = res
	bs.EventListInsert(ev, EvListActive)
}

// HandleActiveEvents handles active events.
func (bs *EventBase) HandleActiveEvents() {
	for e := bs.ActiveEvList.Front(); e != nil; e = e.Next() {
		ev := e.Value.(*Event)
		bs.EventListRemove(ev, EvListActive)
		if ev.Events&EvTimeout != 0 && ev.Flags&EvListInserted != 0 {
			now := time.Now().UnixMilli()
			ev.Deadline = now + ev.Timeout
			bs.EvHeap.PushEvent(ev)
		}
		ev.Cb(ev.Fd, ev.Res, ev.Arg)
	}
}

// EventListInsert inserts an event into the event list.
// Double insertion is possible for active events.
func (bs *EventBase) EventListInsert(ev *Event, which int) {
	if ev.Flags&which != 0 {
		if ev.Flags&EvListActive != 0 {
			return
		}
	}

	ev.Flags |= which
	switch which {
	case EvListInserted:
		ev.Ele = bs.EvList.PushBack(ev)
	case EvListActive:
		ev.ActiveEle = bs.ActiveEvList.PushBack(ev)
	}
}

// EventListRemove removes an event from the event list.
func (bs *EventBase) EventListRemove(ev *Event, which int) {
	if ev.Flags&which == 0 {
		return
	}

	ev.Flags &^= which
	switch which {
	case EvListInserted:
		bs.EvList.Remove(ev.Ele)
		ev.Ele = nil
	case EvListActive:
		bs.ActiveEvList.Remove(ev.ActiveEle)
		ev.ActiveEle = nil
	}
}
