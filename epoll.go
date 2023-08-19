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
	maxUint32     = 0xFFFFFFFF
)

// fdEvent is the event of a file descriptor.
type fdEvent struct {
	// r is the read event.
	r *Event
	// w is the write event.
	w *Event
	// c is the close event.
	c *Event
	// e is the edge-triggered behavior.
	e bool
}

func (fdEv *fdEvent) toEpollEvents() uint32 {
	epEvents := uint32(0)
	if fdEv.r != nil {
		epEvents |= syscall.EPOLLIN
	}
	if fdEv.w != nil {
		epEvents |= syscall.EPOLLOUT
	}
	if fdEv.c != nil {
		epEvents |= syscall.EPOLLRDHUP
	}
	if fdEv.e {
		epEvents |= syscall.EPOLLET & maxUint32
	}
	return epEvents
}

type epoll struct {
	fd        int
	fdEvs     map[int]*fdEvent
	epollEvs  []syscall.EpollEvent
	signalEvs map[int]*Event
	signalFd0 int
	signalFd1 int
	signaler  *signaler
}

func newEpoll() (*epoll, error) {
	fd, err := syscall.EpollCreate1(0)
	if err != nil {
		return nil, err
	}

	signalFds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_SEQPACKET, 0)
	if err != nil {
		syscall.Close(fd)
		return nil, err
	}

	err = syscall.EpollCtl(fd, syscall.EPOLL_CTL_ADD, signalFds[0], &syscall.EpollEvent{
		Events: syscall.EPOLLIN,
		Fd:     int32(signalFds[0]),
	})
	if err != nil {
		syscall.Close(fd)
		syscall.Close(signalFds[0])
		syscall.Close(signalFds[1])
		return nil, err
	}

	ep := &epoll{
		fd:        fd,
		fdEvs:     make(map[int]*fdEvent),
		epollEvs:  make([]syscall.EpollEvent, initialNEvent),
		signalEvs: map[int]*Event{},
		signalFd0: signalFds[0],
		signalFd1: signalFds[1],
	}

	ep.signaler = newSignaler(ep.triggerSignal)

	return ep, nil
}

func (ep *epoll) add(ev *Event) error {
	if ev.events&EvSignal != 0 {
		ep.signalEvs[ev.fd] = ev
		return ep.signaler.subscribe(ev.fd)
	}

	op := syscall.EPOLL_CTL_ADD
	es, ok := ep.fdEvs[ev.fd]
	if ok {
		op = syscall.EPOLL_CTL_MOD
	} else {
		es = &fdEvent{}
		ep.fdEvs[ev.fd] = es
	}

	if ev.events&EvRead != 0 {
		es.r = ev
	}
	if ev.events&EvWrite != 0 {
		es.w = ev
	}
	if ev.events&EvClosed != 0 {
		es.c = ev
	}
	es.e = ev.events&EvET != 0

	epEv := &syscall.EpollEvent{Events: es.toEpollEvents()}

	*(**fdEvent)(unsafe.Pointer(&epEv.Fd)) = es

	return syscall.EpollCtl(ep.fd, op, ev.fd, epEv)
}

func (ep *epoll) del(ev *Event) error {
	if ev.events&EvSignal != 0 {
		delete(ep.signalEvs, ev.fd)
		return ep.signaler.unsubscribe(ev.fd)
	}

	es := ep.fdEvs[ev.fd]

	if ev.events&EvRead != 0 {
		es.r = nil
	}
	if ev.events&EvWrite != 0 {
		es.w = nil
	}
	if ev.events&EvClosed != 0 {
		es.c = nil
	}

	op := syscall.EPOLL_CTL_DEL
	if es.toEpollEvents()&^EvET == 0 {
		delete(ep.fdEvs, ev.fd)
	} else {
		op = syscall.EPOLL_CTL_MOD
	}

	epEv := &syscall.EpollEvent{Events: es.toEpollEvents()}

	*(**fdEvent)(unsafe.Pointer(&epEv.Fd)) = es

	return syscall.EpollCtl(ep.fd, op, ev.fd, epEv)
}

func (ep *epoll) polling(cb func(ev *Event, res uint32), timeout int) error {
	n, err := syscall.EpollWait(ep.fd, ep.epollEvs, timeout)
	if err != nil && !temporaryErr(err) {
		return err
	}

	for i := 0; i < n; i++ {
		if ep.epollEvs[i].Fd == int32(ep.signalFd0) {
			ep.onSignal(cb)
			continue
		}

		var evRead, evWrite, evClosed *Event
		what := ep.epollEvs[i].Events
		es := *(**fdEvent)(unsafe.Pointer(&ep.epollEvs[i].Fd))

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

		ET := uint32(0)
		if es.e {
			ET = EvET
		}
		if evRead != nil {
			cb(evRead, EvRead|ET)
		}
		if evWrite != nil {
			cb(evWrite, EvWrite|ET)
		}
		if evClosed != nil {
			cb(evClosed, EvClosed|ET)
		}
	}

	if n == len(ep.epollEvs) && n < maxNEvent {
		ep.epollEvs = make([]syscall.EpollEvent, n*2)
	}

	return nil
}

func (ep *epoll) close() error {
	ep.signaler.close()
	if err := syscall.Close(ep.signalFd0); err != nil {
		return err
	}
	if err := syscall.Close(ep.signalFd1); err != nil {
		return err
	}
	if err := syscall.Close(ep.fd); err != nil {
		return err
	}
	return nil
}

func (ep *epoll) triggerSignal(signal int) {
	syscall.Write(ep.signalFd1, []byte{byte(signal)})
}

func (ep *epoll) onSignal(cb func(ev *Event, res uint32)) {
	buf := make([]byte, 1)
	syscall.Read(ep.signalFd0, buf)
	ev, ok := ep.signalEvs[int(buf[0])]
	if !ok {
		return
	}
	cb(ev, EvSignal)
}
