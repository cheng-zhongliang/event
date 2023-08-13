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
	// events is events.
	events uint32
	// epEvents is epoll events.
	epEvents uint32
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

	ep.signaler = newSignaler(ep.triggerSignal)

	return ep, nil
}

func (ep *epoll) add(ev *Event) error {
	if ev.events&EvSignal != 0 {
		if _, ok := ep.signalEvs[ev.fd]; !ok {
			ep.signalEvs[ev.fd] = ev
			ep.signaler.subscribe(ev.fd)
			return nil
		}
		return ErrEventExists
	}

	op := syscall.EPOLL_CTL_ADD
	es, ok := ep.fdEvs[ev.fd]
	if ok {
		op = syscall.EPOLL_CTL_MOD
	} else {
		es = &fdEvent{}
		ep.fdEvs[ev.fd] = es
	}

	if es.events&ev.events != 0 {
		return ErrEventExists
	}

	es.events |= ev.events

	if ev.events&EvRead != 0 {
		es.r = ev
		es.epEvents |= syscall.EPOLLIN
	}
	if ev.events&EvWrite != 0 {
		es.w = ev
		es.epEvents |= syscall.EPOLLOUT
	}
	if ev.events&EvClosed != 0 {
		es.c = ev
		es.epEvents |= syscall.EPOLLRDHUP
	}
	if ev.events&EvET != 0 {
		es.epEvents |= syscall.EPOLLET & 0xFFFFFFFF
	}

	epEv := &syscall.EpollEvent{Events: es.epEvents}

	*(**fdEvent)(unsafe.Pointer(&epEv.Fd)) = es

	return syscall.EpollCtl(ep.fd, op, ev.fd, epEv)
}

func (ep *epoll) del(ev *Event) error {
	if ev.events&EvSignal != 0 {
		if _, ok := ep.signalEvs[ev.fd]; ok {
			delete(ep.signalEvs, ev.fd)
			ep.signaler.unsubscribe(ev.fd)
			return nil
		}
		return ErrEventNotExists
	}

	es := ep.fdEvs[ev.fd]

	es.events &^= ev.events

	if ev.events&EvRead != 0 {
		es.r = nil
		es.epEvents &^= syscall.EPOLLIN
	}
	if ev.events&EvWrite != 0 {
		es.w = nil
		es.epEvents &^= syscall.EPOLLOUT
	}
	if ev.events&EvClosed != 0 {
		es.c = nil
		es.epEvents &^= syscall.EPOLLRDHUP
	}
	if ev.events&EvET != 0 {
		es.epEvents &^= syscall.EPOLLET & 0xFFFFFFFF
	}

	op := syscall.EPOLL_CTL_DEL
	if es.epEvents == 0 {
		delete(ep.fdEvs, ev.fd)
	} else {
		op = syscall.EPOLL_CTL_MOD
	}

	epEv := &syscall.EpollEvent{Events: es.epEvents}

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
			cb(evRead, EvRead|EvET)
		}
		if evWrite != nil {
			cb(evWrite, EvWrite|EvET)
		}
		if evClosed != nil {
			cb(evClosed, EvClosed|EvET)
		}
	}

	return nil
}

func (ep *epoll) close() {
	ep.signaler.close()
	syscall.Close(ep.signalFd0)
	syscall.Close(ep.signalFd1)
	syscall.Close(ep.fd)
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
