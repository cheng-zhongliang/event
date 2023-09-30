//go:build linux
// +build linux

// Copyright (c) 2023 cheng-zhongliang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"syscall"
	"unsafe"
)

const (
	initialNEvent = 0x20
	maxNEvent     = 0x1000
	maxUint32     = 0xFFFFFFFF
)

type fdEvent struct {
	r   *Event
	w   *Event
	c   *Event
	et  uint8
	evs uint32
}

type poller struct {
	fd       int
	fdEvents map[int]*fdEvent
	events   []syscall.EpollEvent
}

func newPoller() (*poller, error) {
	fd, err := syscall.EpollCreate1(0)
	if err != nil {
		return nil, err
	}

	return &poller{
		fd:       fd,
		fdEvents: make(map[int]*fdEvent),
		events:   make([]syscall.EpollEvent, initialNEvent),
	}, nil
}

func (ep *poller) add(ev *Event) error {
	op := syscall.EPOLL_CTL_ADD
	es, ok := ep.fdEvents[ev.fd]
	if ok {
		op = syscall.EPOLL_CTL_MOD
	} else {
		es = new(fdEvent)
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
	if ev.events&EvClosed != 0 {
		es.c = ev
		es.evs |= syscall.EPOLLRDHUP
	}
	if ev.events&EvET != 0 {
		es.et++
		es.evs |= syscall.EPOLLET & maxUint32
	}

	epEv := &syscall.EpollEvent{Events: es.evs}

	*(**fdEvent)(unsafe.Pointer(&epEv.Fd)) = es

	return syscall.EpollCtl(ep.fd, op, ev.fd, epEv)
}

func (ep *poller) del(ev *Event) error {
	es := ep.fdEvents[ev.fd]

	if ev.events&EvRead != 0 {
		es.r = nil
		es.evs &^= syscall.EPOLLIN
	}
	if ev.events&EvWrite != 0 {
		es.w = nil
		es.evs &^= syscall.EPOLLOUT
	}
	if ev.events&EvClosed != 0 {
		es.c = nil
		es.evs &^= syscall.EPOLLRDHUP
	}
	if ev.events&EvET != 0 {
		if es.et--; es.et == 0 {
			es.evs &^= syscall.EPOLLET
		}
	}

	op := syscall.EPOLL_CTL_DEL
	if es.evs == 0 {
		delete(ep.fdEvents, ev.fd)
	} else {
		op = syscall.EPOLL_CTL_MOD
	}

	epEv := &syscall.EpollEvent{Events: es.evs}

	*(**fdEvent)(unsafe.Pointer(&epEv.Fd)) = es

	return syscall.EpollCtl(ep.fd, op, ev.fd, epEv)
}

func (ep *poller) polling(cb func(ev *Event, res uint32), timeout int) error {
	n, err := syscall.EpollWait(ep.fd, ep.events, timeout)
	if err != nil && !temporaryErr(err) {
		return err
	}

	for i := 0; i < n; i++ {
		var evRead, evWrite, evClosed *Event
		what := ep.events[i].Events
		es := *(**fdEvent)(unsafe.Pointer(&ep.events[i].Fd))

		if what&syscall.EPOLLERR != 0 {
			evRead = es.r
			evWrite = es.w
		} else if what&syscall.EPOLLHUP != 0 && what&syscall.EPOLLRDHUP == 0 {
			evRead = es.r
			evWrite = es.w
		} else {
			if what&syscall.EPOLLIN != 0 {
				evRead = es.r
			}
			if what&syscall.EPOLLOUT != 0 {
				evWrite = es.w
			}
			if what&syscall.EPOLLRDHUP != 0 {
				evClosed = es.c
			}
		}

		if evRead != nil {
			cb(evRead, evRead.events&(EvRead|EvET))
		}
		if evWrite != nil {
			cb(evWrite, evWrite.events&(EvWrite|EvET))
		}
		if evClosed != nil {
			cb(evClosed, evClosed.events&(EvClosed|EvET))
		}
	}

	if n == len(ep.events) && n < maxNEvent {
		ep.events = make([]syscall.EpollEvent, n<<1)
	}

	return nil
}
