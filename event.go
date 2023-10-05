// Copyright (c) 2023 cheng-zhongliang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"time"
)

const (
	// EvRead is readable event.
	EvRead = 1 << iota
	// EvWrite is writable event.
	EvWrite = 1 << iota
	// EvTimeout is timeout event.
	EvTimeout = 1 << iota

	// EvPersist is persistent behavior option.
	EvPersist = 1 << iota
	// EvET is edge-triggered behavior option.
	EvET = 1 << iota

	// EvLoopOnce is the flag to control event base loop just once.
	EvLoopOnce = 001
	// EvLoopNoblock is the flag to control event base loop not block.
	EvLoopNoblock = 002

	// evListInserted is the flag to indicate the event is in the event list.
	evListInserted = 0x01
	// evListActive is the flag to indicate the event is in the active event list.
	evListActive = 0x02
	// evListTimeout is the flag to indicate the event is in the timeout event heap.
	evListTimeout = 0x04

	// HPri is the high priority.
	HPri eventPriority = 0b00
	// MPri is the middle priority.
	MPri eventPriority = 0b01
	// LPri is the low priority.
	LPri eventPriority = 0b10
)

// eventPriority is the priority of the event.
type eventPriority uint8

// Event is the event to watch.
type Event struct {
	// base is the event base of the event.
	base *EventBase

	// ele is the element in the total event list.
	ele *element
	// activeEle is the element in the active event list.
	activeEle *element
	// index is the index in the event heap.
	index int

	// fd is the file descriptor to watch.
	fd int
	// events is the events to watch. Such as EvRead or EvWrite.
	events uint32

	// cb is the callback function when the event is triggered.
	cb func(fd int, events uint32, arg any)
	// arg is the argument passed to the callback function.
	arg any

	// res is the result passed to the callback function.
	res uint32
	// flags is the status of the event in the event list.
	flags int

	// timeout is the timeout of the event.
	timeout time.Duration
	// deadline is the deadline for the event.
	deadline time.Time

	// priority is the priority of the event.
	priority eventPriority
}

// New creates a new event.
func New(base *EventBase, fd int, events uint32, callback func(fd int, events uint32, arg any), arg any) *Event {
	ev := new(Event)
	ev.Assign(base, fd, events, callback, arg, MPri)
	return ev
}

// Assign assigns the event.
// It is used to reuse the event.
func (ev *Event) Assign(base *EventBase, fd int, events uint32, callback func(fd int, events uint32, arg any), arg any, priority eventPriority) {
	ev.base = base
	ev.fd = fd
	ev.events = events
	ev.cb = callback
	ev.arg = arg
	ev.priority = priority
	ev.res = 0
	ev.flags = 0
	ev.timeout = 0
	ev.deadline = time.Time{}
	ev.ele = nil
	ev.activeEle = nil
	ev.index = -1
}

// Attach adds the event to the event base.
// Timeout is the timeout of the event. Default is 0, which means no timeout.
// But if EvTimeout is set in the event, the 0 represents expired immediately.
func (ev *Event) Attach(timeout time.Duration) error {
	if ev.events&(EvRead|EvWrite|EvClosed|EvTimeout) == 0 {
		return ErrEventInvalid
	}

	if ev.flags&evListInserted != 0 {
		return ErrEventExists
	}

	ev.timeout = timeout

	return ev.base.addEvent(ev)
}

// Detach deletes the event from the event base.
// The event will not be triggered after it is detached.
func (ev *Event) Detach() error {
	if ev.flags&evListInserted == 0 {
		return ErrEventNotExists
	}

	return ev.base.delEvent(ev)
}

// Base returns the event base of the event.
func (ev *Event) Base() *EventBase {
	return ev.base
}

// Fd returns the file descriptor of the event.
func (ev *Event) Fd() int {
	return ev.fd
}

// Events returns the events of the event.
func (ev *Event) Events() uint32 {
	return ev.events
}

// Timeout returns the timeout of the event.
func (ev *Event) Timeout() time.Duration {
	return ev.timeout
}

// Priority returns the priority of the event.
func (ev *Event) Priority() eventPriority {
	return ev.priority
}

// SetPriority sets the priority of the event.
func (ev *Event) SetPriority(priority eventPriority) {
	ev.priority = priority
}

// EventBase is the base of all events.
type EventBase struct {
	// poller is the event poller to watch events.
	poller *poller
	// evList is the list of all events.
	evList *list
	// activeEvList is the list of active events.
	activeEvLists []*list
	// eventHeap is the min heap of timeout events.
	evHeap *eventHeap
	// nowTimeCache is the cache of now time.
	nowTimeCache time.Time
}

// NewBase creates a new event base.
func NewBase() (*EventBase, error) {
	poller, err := newPoller()
	if err != nil {
		return nil, err
	}

	return &EventBase{
		poller:        poller,
		evList:        newList(),
		activeEvLists: []*list{newList(), newList(), newList()},
		evHeap:        new(eventHeap),
		nowTimeCache:  time.Time{},
	}, nil
}

// Loop loops events.
// If flags is EvLoopOnce, it will block until an event is triggered. Then it will exit.
// If flags is EvLoopNoblock, it will not block.
func (bs *EventBase) Loop(flags int) error {
	bs.clearTimeCache()

	for {
		err := bs.poller.polling(bs.onActive, bs.waitTime(flags&EvLoopNoblock != 0))
		if err != nil {
			return err
		}

		bs.updateTimeCache()

		bs.onTimeout()

		bs.handleActiveEvents()

		if flags&EvLoopOnce != 0 {
			return nil
		}
	}
}

// Dispatch dispatches events.
// It will block until events trigger.
func (bs *EventBase) Dispatch() error {
	return bs.Loop(0)
}

// Exit exit event loop.
func (bs *EventBase) Exit() error {
	return bs.poller.close()
}

// addEvent adds an event to the event base.
func (bs *EventBase) addEvent(ev *Event) error {
	if ev.events&EvTimeout != 0 {
		ev.deadline = bs.now().Add(ev.timeout)
		bs.eventQueueInsert(ev, evListTimeout)
	}

	bs.eventQueueInsert(ev, evListInserted)

	if ev.events&(EvRead|EvWrite|EvClosed) != 0 {
		return bs.poller.add(ev)
	}

	return nil
}

// delEvent deletes an event from the event base.
func (bs *EventBase) delEvent(ev *Event) error {
	bs.eventQueueRemove(ev, evListTimeout)

	bs.eventQueueRemove(ev, evListActive)

	bs.eventQueueRemove(ev, evListInserted)

	if ev.events&(EvRead|EvWrite|EvClosed) != 0 {
		return bs.poller.del(ev)
	}

	return nil
}

func (bs *EventBase) waitTime(noblock bool) time.Duration {
	if noblock {
		return 0
	}
	if !bs.evHeap.empty() {
		ev := bs.evHeap.peekEvent()
		if d := ev.deadline.Sub(bs.now()); d > 0 {
			return d
		}
		return 0
	}
	return -1
}

func (bs *EventBase) onTimeout() {
	now := bs.now()
	for !bs.evHeap.empty() {
		ev := bs.evHeap.peekEvent()
		if ev.deadline.After(now) {
			break
		}

		bs.eventQueueRemove(ev, evListTimeout)

		bs.onActive(ev, EvTimeout)
	}
}

func (bs *EventBase) onActive(ev *Event, res uint32) {
	if ev.flags&evListActive != 0 {
		ev.res |= res
		return
	}

	ev.res = res
	bs.eventQueueInsert(ev, evListActive)
}

func (bs *EventBase) handleActiveEvents() {
	for i := range bs.activeEvLists {
		for e := bs.activeEvLists[i].front(); e != nil; {
			next := e.nextEle()
			ev := e.value.(*Event)
			if ev.events&EvPersist != 0 {
				bs.eventQueueRemove(ev, evListActive)
			} else {
				bs.delEvent(ev)
			}
			e = next

			if ev.events&EvTimeout != 0 && ev.events&EvPersist != 0 {
				ev.deadline = bs.now().Add(ev.timeout)
				bs.eventQueueInsert(ev, evListTimeout)
			}

			ev.cb(ev.fd, ev.res, ev.arg)
		}
	}
}

func (bs *EventBase) eventQueueInsert(ev *Event, which int) {
	if ev.flags&which != 0 {
		return
	}

	ev.flags |= which
	switch which {
	case evListInserted:
		ev.ele = bs.evList.pushBack(ev)
	case evListActive:
		ev.activeEle = bs.activeEvLists[ev.priority].pushBack(ev)
	case evListTimeout:
		ev.index = bs.evHeap.pushEvent(ev)
	}
}

func (bs *EventBase) eventQueueRemove(ev *Event, which int) {
	if ev.flags&which == 0 {
		return
	}

	ev.flags &^= which
	switch which {
	case evListInserted:
		bs.evList.remove(ev.ele)
	case evListActive:
		bs.activeEvLists[ev.priority].remove(ev.activeEle)
	case evListTimeout:
		bs.evHeap.removeEvent(ev.index)
	}
}

func (bs *EventBase) now() time.Time {
	if !bs.nowTimeCache.IsZero() {
		return bs.nowTimeCache
	}
	return time.Now()
}

func (bs *EventBase) updateTimeCache() {
	bs.nowTimeCache = time.Now()
}

func (bs *EventBase) clearTimeCache() {
	bs.nowTimeCache = time.Time{}
}
