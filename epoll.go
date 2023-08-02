package event

import (
	"syscall"
	"unsafe"
)

// fdEvent is the event of a file descriptor.
type fdEvent struct {
	// r is the read event.
	r *Event
	// w is the write event.
	w *Event
	// c is the close event.
	c *Event
	// e is edge-triggered behavior.
	e bool
}

type epoll struct {
	fd           int
	fdEvs        map[int]*fdEvent
	epollEvs     []syscall.EpollEvent
	signalEvs    map[int]*Event
	signalFd0    int
	signalFd1    int
	signalPoller *signalPoller
}

func newEpoll() (*epoll, error) {
	fd, err := syscall.EpollCreate1(syscall.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}

	signalFds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_SEQPACKET|syscall.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}

	err = syscall.EpollCtl(fd, syscall.EPOLL_CTL_ADD, signalFds[0], &syscall.EpollEvent{Events: syscall.EPOLLIN, Fd: int32(signalFds[0])})
	if err != nil {
		return nil, err
	}

	ep := &epoll{
		fd:        fd,
		fdEvs:     make(map[int]*fdEvent),
		epollEvs:  make([]syscall.EpollEvent, 0xFF),
		signalEvs: map[int]*Event{},
		signalFd0: signalFds[0],
		signalFd1: signalFds[1],
	}

	ep.signalPoller = newSignalPoller(ep.triggerSignal)

	return ep, nil
}

func (ep *epoll) add(ev *Event) error {
	if ev.events&EvSignal != 0 {
		ep.signalEvs[ev.fd] = ev
		ep.signalPoller.subscribeSignal(ev.fd)
		return nil
	}

	epEv := &syscall.EpollEvent{}
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
	if ev.events&EvET != 0 {
		es.e = true
	}

	if es.r != nil {
		epEv.Events |= syscall.EPOLLIN
	}
	if es.w != nil {
		epEv.Events |= syscall.EPOLLOUT
	}
	if es.c != nil {
		epEv.Events |= syscall.EPOLLRDHUP
	}
	if es.e {
		epEv.Events |= syscall.EPOLLET & 0xFFFFFFFF
	}

	*(**fdEvent)(unsafe.Pointer(&epEv.Fd)) = es

	return syscall.EpollCtl(ep.fd, op, ev.fd, epEv)
}

func (ep *epoll) del(ev *Event) error {
	if ev.events&EvSignal != 0 {
		delete(ep.signalEvs, ev.fd)
		ep.signalPoller.unsubscribeSignal(ev.fd)
		return nil
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

	epEv := &syscall.EpollEvent{}
	op := syscall.EPOLL_CTL_DEL

	if es.r != nil {
		epEv.Events |= syscall.EPOLLIN
	}
	if es.w != nil {
		epEv.Events |= syscall.EPOLLOUT
	}
	if es.c != nil {
		epEv.Events |= syscall.EPOLLRDHUP
	}
	if es.e {
		epEv.Events |= syscall.EPOLLET & 0xFFFFFFFF
	}

	if epEv.Events == 0 {
		delete(ep.fdEvs, ev.fd)
	} else {
		op = syscall.EPOLL_CTL_MOD
	}

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

		if evRead != nil {
			cb(evRead, EvRead)
		}
		if evWrite != nil {
			cb(evWrite, EvWrite)
		}
		if evClosed != nil {
			cb(evClosed, EvClosed)
		}
	}

	return nil
}

func (ep *epoll) close() error {
	ep.signalPoller.close()
	syscall.Close(ep.signalFd0)
	syscall.Close(ep.signalFd0)
	return syscall.Close(ep.fd)
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
