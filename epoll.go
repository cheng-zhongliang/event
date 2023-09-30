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

type poller struct {
	fd       int
	fdEvs    map[int]*fdEvent
	epollEvs []syscall.EpollEvent
}

func newPoller() (*poller, error) {
	fd, err := syscall.EpollCreate1(0)
	if err != nil {
		return nil, err
	}

	return &epoll{
		fd:       fd,
		fdEvs:    make(map[int]*fdEvent),
		epollEvs: make([]syscall.EpollEvent, initialNEvent),
	}, nil
}

func (ep *poller) add(ev *Event) error {
	op := syscall.EPOLL_CTL_ADD
	es, ok := ep.fdEvs[ev.fd]
	if ok {
		op = syscall.EPOLL_CTL_MOD
	} else {
		es = newFdEvent()
		ep.fdEvs[ev.fd] = es
	}

	changed := es.add(ev)
	if !changed {
		return nil
	}

	epEv := &syscall.EpollEvent{Events: es.evs}

	*(**fdEvent)(unsafe.Pointer(&epEv.Fd)) = es

	return syscall.EpollCtl(ep.fd, op, ev.fd, epEv)
}

func (ep *poller) del(ev *Event) error {
	es := ep.fdEvs[ev.fd]

	changed := es.del(ev)
	if !changed {
		return nil
	}

	op := syscall.EPOLL_CTL_DEL
	if es.evs == 0 {
		delete(ep.fdEvs, ev.fd)
	} else {
		op = syscall.EPOLL_CTL_MOD
	}

	epEv := &syscall.EpollEvent{Events: es.evs}

	*(**fdEvent)(unsafe.Pointer(&epEv.Fd)) = es

	return syscall.EpollCtl(ep.fd, op, ev.fd, epEv)
}

func (ep *poller) polling(cb func(ev *Event, res uint32), timeout int) error {
	n, err := syscall.EpollWait(ep.fd, ep.epollEvs, timeout)
	if err != nil && !temporaryErr(err) {
		return err
	}

	for i := 0; i < n; i++ {
		which := uint32(0)
		what := ep.epollEvs[i].Events

		if what&syscall.EPOLLERR != 0 {
			which |= EvRead | EvWrite
		} else if what&syscall.EPOLLHUP != 0 && what&syscall.EPOLLRDHUP == 0 {
			which |= EvRead | EvWrite
		} else {
			if what&syscall.EPOLLIN != 0 {
				which |= EvRead
			}
			if what&syscall.EPOLLOUT != 0 {
				which |= EvWrite
			}
			if what&syscall.EPOLLRDHUP != 0 {
				which |= EvClosed
			}
		}

		if which != 0 {
			es := *(**fdEvent)(unsafe.Pointer(&ep.epollEvs[i].Fd))
			es.foreach(func(ev *Event) {
				if ev.events&which != 0 {
					cb(ev, ev.events&(which|EvET))
				}
			})
		}
	}

	if n == len(ep.epollEvs) && n < maxNEvent {
		ep.epollEvs = make([]syscall.EpollEvent, n*2)
	}

	return nil
}

func (ep *poller) close() error {
	return syscall.Close(ep.fd)
}

// fdEvent is the event of a file descriptor.
type fdEvent struct {
	list   *list
	nread  uint8
	nwrite uint8
	nclose uint8
	nET    uint8
	evs    uint32
}

func newFdEvent() *fdEvent {
	return &fdEvent{list: newList()}
}

func (fdEv *fdEvent) add(ev *Event) bool {
	oldEvs := fdEv.evs
	if ev.events&EvRead != 0 {
		fdEv.nread++
	}
	if ev.events&EvWrite != 0 {
		fdEv.nwrite++
	}
	if ev.events&EvClosed != 0 {
		fdEv.nclose++
	}
	if ev.events&EvET != 0 {
		fdEv.nET++
	}
	ev.fdEle = fdEv.list.pushBack(ev)
	fdEv.toEpollEvents()
	return oldEvs != fdEv.evs
}

func (fdEv *fdEvent) del(ev *Event) bool {
	oldEvs := fdEv.evs
	if ev.events&EvRead != 0 {
		fdEv.nread--
	}
	if ev.events&EvWrite != 0 {
		fdEv.nwrite--
	}
	if ev.events&EvClosed != 0 {
		fdEv.nclose--
	}
	if ev.events&EvET != 0 {
		fdEv.nET--
	}
	fdEv.list.remove(ev.fdEle)
	fdEv.toEpollEvents()
	return oldEvs != fdEv.evs
}

func (fdEv *fdEvent) toEpollEvents() {
	fdEv.evs = 0
	if fdEv.nread > 0 {
		fdEv.evs |= syscall.EPOLLIN
	}
	if fdEv.nwrite > 0 {
		fdEv.evs |= syscall.EPOLLOUT
	}
	if fdEv.nclose > 0 {
		fdEv.evs |= syscall.EPOLLRDHUP
	}
	if fdEv.nET > 0 {
		fdEv.evs |= syscall.EPOLLET & maxUint32
	}
}

func (fdEv *fdEvent) foreach(cb func(ev *Event)) {
	for ele := fdEv.list.front(); ele != nil; ele = ele.nextEle() {
		cb(ele.value.(*Event))
	}
}
