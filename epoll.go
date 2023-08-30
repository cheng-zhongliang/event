/*
 * Copyright (c) 2023 cheng-zhongliang. All rights reserved.
 *
 * This source code is licensed under the 3-Clause BSD License, which can be
 * found in the accompanying "LICENSE" file, or at:
 * https://opensource.org/licenses/BSD-3-Clause
 *
 * By accessing, using, copying, modifying, or distributing this software,
 * you agree to the terms and conditions of the BSD 3-Clause License.
 */

package event

import (
	"syscall"
	"unsafe"
)

const (
	initialNEvent = 0x20
	maxNEvent     = 0x1000
)

type epoll struct {
	fd       int
	fdEvs    map[int]*fdEvent
	epollEvs []syscall.EpollEvent
}

func newEpoll() (*epoll, error) {
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

func (ep *epoll) add(ev *Event) error {
	op := syscall.EPOLL_CTL_ADD
	es, ok := ep.fdEvs[ev.fd]
	if ok {
		op = syscall.EPOLL_CTL_MOD
	} else {
		es = newFdEvent()
		ep.fdEvs[ev.fd] = es
	}

	es.add(ev)

	epEv := &syscall.EpollEvent{Events: es.toEpollEvents()}

	*(**fdEvent)(unsafe.Pointer(&epEv.Fd)) = es

	return syscall.EpollCtl(ep.fd, op, ev.fd, epEv)
}

func (ep *epoll) del(ev *Event) error {
	es := ep.fdEvs[ev.fd]

	es.del(ev)

	epEvs := es.toEpollEvents()

	op := syscall.EPOLL_CTL_DEL
	if epEvs == 0 {
		delete(ep.fdEvs, ev.fd)
	} else {
		op = syscall.EPOLL_CTL_MOD
	}

	epEv := &syscall.EpollEvent{Events: epEvs}

	*(**fdEvent)(unsafe.Pointer(&epEv.Fd)) = es

	return syscall.EpollCtl(ep.fd, op, ev.fd, epEv)
}

func (ep *epoll) polling(cb func(ev *Event, res uint32), timeout int) error {
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
			es.active(which|EvET, cb)
		}
	}

	if n == len(ep.epollEvs) && n < maxNEvent {
		ep.epollEvs = make([]syscall.EpollEvent, n*2)
	}

	return nil
}

func (ep *epoll) close() error {
	return syscall.Close(ep.fd)
}
