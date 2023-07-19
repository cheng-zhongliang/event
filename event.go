package event

import (
	"time"
)

const (
	// EvRead is readable event.
	EvRead = 1 << iota
	// EvWrite is writable event.
	EvWrite = 1 << iota
	// EvTimeout is timeout event
	EvTimeout = 1 << iota
	// EvPersist is persistent event.
	EvPersist = 1 << iota

	// EvListInserted is the flag to indicate the event is in the event list.
	EvListInserted = 0x01
	// EvListActive is the flag to indicate the event is in the active event list.
	EvListActive = 0x02
)

// Event is the event to watch.
type Event struct {
	// Ele is the element in the total event list.
	Ele *EventElement
	// ActiveEle is the element in the active event list.
	ActiveEle *EventElement

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
	EvList *EventList
	// ActiveEvList is the list of active events.
	ActiveEvList *EventList
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
		EvList:       NewEventList(),
		ActiveEvList: NewEventList(),
		EvHeap:       NewEventHeap(),
	}, nil
}

// AddEvent adds an event to the event base.
func (bs *EventBase) AddEvent(ev *Event, timeout time.Duration) error {
	if ev.IsExist() {
		return ErrEventAlreadyAdded
	}

	if ev.IsTimeout() {
		ev.Timeout = timeout.Milliseconds()
		ev.Deadline = time.Now().Add(timeout).UnixMilli()
		bs.EvHeap.PushEvent(ev)
	}

	bs.EventListInsert(ev, EvListInserted)

	if ev.IsRead() || ev.IsWrite() {
		return bs.Poller.Add(ev)
	} else {
		return nil
	}
}

// DelEvent deletes an event from the event base.
func (bs *EventBase) DelEvent(ev *Event) error {
	if !ev.IsExist() {
		return ErrEventNotAdded
	}

	if ev.IsActive() {
		bs.EventListRemove(ev, EvListActive)
	}

	if ev.IsExist() {
		bs.EventListRemove(ev, EvListInserted)
	}

	if ev.IsRead() || ev.IsWrite() {
		return bs.Poller.Del(ev)
	} else {
		return nil
	}
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
		if !ev.IsExist() {
			bs.EvHeap.PopEvent()
			continue
		}
		if ev.Deadline > now {
			break
		}
		bs.EvHeap.PopEvent()
		bs.OnActive(ev, EvTimeout)
	}
}

// OnEvent adds an event to the active event list.
func (bs *EventBase) OnActive(ev *Event, res uint32) {
	if ev.IsActive() {
		ev.Res |= res
		return
	}

	ev.Res = res
	bs.EventListInsert(ev, EvListActive)
}

// HandleActiveEvents handles active events.
func (bs *EventBase) HandleActiveEvents() {
	for e := bs.ActiveEvList.Front(); e != nil; {
		next := e.Next()
		ev := e.Value
		if !ev.IsPersist() {
			bs.DelEvent(ev)
		} else {
			bs.EventListRemove(ev, EvListActive)
		}
		e = next

		if ev.IsTimeout() && ev.IsExist() {
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
	if ev.Flags&which != 0 && ev.IsActive() {
		return
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

// IsExist return true if the event already inserted.
func (ev *Event) IsExist() bool {
	return ev.Flags&EvListInserted != 0
}

// IsActive return true if the event active.
func (ev *Event) IsActive() bool {
	return ev.Flags&EvListActive != 0
}

// IsTimeout return true if the event is timeout.
func (ev *Event) IsTimeout() bool {
	return ev.Events&EvTimeout != 0
}

// IsRead return true if the event is read.
func (ev *Event) IsRead() bool {
	return ev.Events&EvRead != 0
}

// IsWrite return true if the event is write.
func (ev *Event) IsWrite() bool {
	return ev.Events&EvWrite != 0
}

// IsPersist return true if the event is persist.
func (ev *Event) IsPersist() bool {
	return ev.Events&EvPersist != 0
}
