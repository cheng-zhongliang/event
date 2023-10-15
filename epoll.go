// Copyright (c) 2023 cheng-zhongliang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux
// +build linux

package event

import (
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	initialNEvent = 0x20
	maxNEvent     = 0x1000
	maxUint32     = 0xFFFFFFFF
)

var evPool = sync.Pool{
	New: func() any {
		return new(fdEvent)
	},
}

type fdEvent struct {
	r   *Event
	w   *Event
	et  uint8
	evs uint32
}

type poll struct {
	fd       int
	fdEvents map[int]*fdEvent
	events   []syscall.EpollEvent
}

func openPoll() (*poll, error) {
	fd, err := syscall.EpollCreate1(0)
	if err != nil {
		return nil, err
	}

	return &poll{
		fd:       fd,
		fdEvents: make(map[int]*fdEvent),
		events:   make([]syscall.EpollEvent, initialNEvent),
	}, nil
}

func (ep *poll) add(ev *Event) error {
	op := syscall.EPOLL_CTL_ADD
	es, ok := ep.fdEvents[ev.fd]
	if ok {
		op = syscall.EPOLL_CTL_MOD
	} else {
		es = evPool.Get().(*fdEvent)
		ep.fdEvents[ev.fd] = es
	}

	if ev.events&EvRead != 0 {
		es.r = ev
		es.evs |= syscall.EPOLLIN
	}
	if ev.events&EvWrite != 0 {
		es.w = ev
		es.evs |= syscall.EPOLLOUT
	}
	if ev.events&EvET != 0 {
		es.et++
		es.evs |= syscall.EPOLLET & maxUint32
	}

	epEv := syscall.EpollEvent{Events: es.evs}

	*(**fdEvent)(unsafe.Pointer(&epEv.Fd)) = es

	return syscall.EpollCtl(ep.fd, op, ev.fd, &epEv)
}

func (ep *poll) del(ev *Event) error {
	es := ep.fdEvents[ev.fd]

	if ev.events&EvRead != 0 {
		es.r = nil
		es.evs &^= syscall.EPOLLIN
	}
	if ev.events&EvWrite != 0 {
		es.w = nil
		es.evs &^= syscall.EPOLLOUT
	}
	if ev.events&EvET != 0 {
		if es.et--; es.et == 0 {
			es.evs &^= syscall.EPOLLET & maxUint32
		}
	}

	op := syscall.EPOLL_CTL_DEL
	if es.evs&(syscall.EPOLLIN|syscall.EPOLLOUT) == 0 {
		delete(ep.fdEvents, ev.fd)
		evPool.Put(es)
	} else {
		op = syscall.EPOLL_CTL_MOD
	}

	epEv := syscall.EpollEvent{Events: es.evs}

	*(**fdEvent)(unsafe.Pointer(&epEv.Fd)) = es

	return syscall.EpollCtl(ep.fd, op, ev.fd, &epEv)
}

func (ep *poll) wait(cb func(ev *Event, res uint32), timeout time.Duration) error {
	n, err := syscall.EpollWait(ep.fd, ep.events, int(timeout.Milliseconds()))
	if err != nil && !temporaryErr(err) {
		return err
	}

	for i := 0; i < n; i++ {
		var evRead, evWrite *Event
		what := ep.events[i].Events
		es := *(**fdEvent)(unsafe.Pointer(&ep.events[i].Fd))

		if what&(syscall.EPOLLERR|syscall.EPOLLHUP) != 0 {
			what |= syscall.EPOLLIN | syscall.EPOLLOUT
		}

		if what&syscall.EPOLLIN != 0 {
			evRead = es.r
		}
		if what&syscall.EPOLLOUT != 0 {
			evWrite = es.w
		}

		if evRead != nil {
			cb(evRead, evRead.events&(EvRead|EvET))
		}
		if evWrite != nil {
			cb(evWrite, evWrite.events&(EvWrite|EvET))
		}
	}

	if n == len(ep.events) && n < maxNEvent {
		ep.events = make([]syscall.EpollEvent, n<<1)
	}

	return nil
}

func (ep *poll) close() error {
	return syscall.Close(ep.fd)
}
