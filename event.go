package event

import (
	"time"
)

// EventPriority is the priority of the event.
type EventPriority int

const (
	// EvRead is readable event.
	EvRead = 1 << iota
	// EvWrite is writable event.
	EvWrite = 1 << iota
	// EvTimeout is timeout event
	EvTimeout = 1 << iota
	// EvSignal is signal event.
	EvSignal = 1 << iota

	// EvPersist is persistent event.
	EvPersist = 0x10

	// EvListInserted is the flag to indicate the event is in the event list.
	EvListInserted = 0x01
	// EvListActive is the flag to indicate the event is in the active event list.
	EvListActive = 0x02
	// EvListTimeout is the flag to indicate the event is in the timeout event list.
	EvListTimeout = 0x04

	// High is the high priority.
	High EventPriority = 0b00
	// Middle is the middle priority.
	Middle EventPriority = 0b01
	// Low is the low priority.
	Low EventPriority = 0b10
)

// Event is the event to watch.
type Event struct {
	// Ele is the element in the total event list.
	Ele *EventListEle
	// ActiveEle is the element in the active event list.
	ActiveEle *EventListEle
	// Index is the index in the event heap.
	Index int

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
	Timeout time.Duration
	// Deadline is the deadline in milliseconds.
	Deadline int64

	// Priority is the priority of the event.
	Priority EventPriority
}

// New creates a new event.
func New(fd int, events uint32, callback func(fd int, events uint32, arg interface{}), arg interface{}) *Event {
	return &Event{
		Fd:       fd,
		Events:   events,
		Cb:       callback,
		Arg:      arg,
		Priority: Middle,
	}
}

// SetPriority sets the priority of the event.
func (ev *Event) SetPriority(priority EventPriority) {
	ev.Priority = priority
}

// EventBase is the base of all events.
type EventBase struct {
	// Poller is the event poller to watch events.
	Poller *Epoll
	// EvList is the list of all events.
	EvList *EventList
	// ActiveEvList is the list of active events.
	ActiveEvLists []*EventList
	// EventHeap is the min heap of timeout events.
	EvHeap *EventHeap
}

// NewBase creates a new event base.
func NewBase() (*EventBase, error) {
	poller, err := NewEpoll()
	if err != nil {
		return nil, err
	}

	return &EventBase{
		Poller:        poller,
		EvList:        NewEventList(),
		ActiveEvLists: []*EventList{NewEventList(), NewEventList(), NewEventList()},
		EvHeap:        NewEventHeap(),
	}, nil
}

// AddEvent adds an event to the event base.
func (bs *EventBase) AddEvent(ev *Event, timeout time.Duration) error {
	if timeout > 0 && ev.Flags&EvListTimeout == 0 {
		ev.Timeout = timeout
		ev.Deadline = time.Now().Add(timeout).UnixMilli()
		bs.EventQueueInsert(ev, EvListTimeout)
	}

	if ev.Flags&EvListInserted == 0 {
		bs.EventQueueInsert(ev, EvListInserted)
		if ev.Events&(EvRead|EvWrite|EvSignal) != 0 {
			return bs.Poller.Add(ev)
		}
	}

	return nil
}

// DelEvent deletes an event from the event base.
func (bs *EventBase) DelEvent(ev *Event) error {
	if ev.Flags&EvListTimeout != 0 {
		bs.EventQueueRemove(ev, EvListTimeout)
	}

	if ev.Flags&EvListActive != 0 {
		bs.EventQueueRemove(ev, EvListActive)
	}

	if ev.Flags&EvListInserted != 0 {
		bs.EventQueueRemove(ev, EvListInserted)
		if ev.Events&(EvRead|EvWrite|EvSignal) != 0 {
			return bs.Poller.Del(ev)
		}
	}

	return nil
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

		bs.DelEvent(ev)

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
	bs.EventQueueInsert(ev, EvListActive)
}

// HandleActiveEvents handles active events.
func (bs *EventBase) HandleActiveEvents() {
	for i := range bs.ActiveEvLists {
		for e := bs.ActiveEvLists[i].Front(); e != nil; {
			next := e.Next()
			ev := e.Value
			if ev.Events&EvPersist != 0 {
				bs.EventQueueRemove(ev, EvListActive)
			} else {
				bs.DelEvent(ev)
			}
			e = next

			if ev.Events&EvTimeout != 0 && ev.Events&EvPersist != 0 {
				bs.AddEvent(ev, ev.Timeout)
			}

			ev.Cb(ev.Fd, ev.Res, ev.Arg)
		}
	}
}

// EventQueueInsert inserts an event into the event list.
// Double insertion is possible for active events.
func (bs *EventBase) EventQueueInsert(ev *Event, which int) {
	if ev.Flags&which != 0 && ev.Flags&EvListActive != 0 {
		return
	}

	ev.Flags |= which
	switch which {
	case EvListInserted:
		ev.Ele = bs.EvList.PushBack(ev)
	case EvListActive:
		ev.ActiveEle = bs.ActiveEvLists[ev.Priority].PushBack(ev)
	case EvListTimeout:
		ev.Index = bs.EvHeap.PushEvent(ev)
	}
}

// EventQueueRemove removes an event from the event list.
func (bs *EventBase) EventQueueRemove(ev *Event, which int) {
	if ev.Flags&which == 0 {
		return
	}

	ev.Flags &^= which
	switch which {
	case EvListInserted:
		bs.EvList.Remove(ev.Ele)
		ev.Ele = nil
	case EvListActive:
		bs.ActiveEvLists[ev.Priority].Remove(ev.ActiveEle)
		ev.ActiveEle = nil
	case EvListTimeout:
		bs.EvHeap.RemoveEvent(ev.Index)
		ev.Index = -1
	}
}
