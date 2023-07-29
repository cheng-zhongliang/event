package event

import (
	"syscall"
	"unsafe"
)

// FdEvent is the event of a file descriptor.
type FdEvent struct {
	// R is the read event.
	R *Event
	// W is the write event.
	W *Event
	// C is the close event.
	C *Event
	// E is edge-triggered behavior.
	E bool
}

// Epoll is the epoll poller implementation.
type Epoll struct {
	// Fd is the file descriptor of epoll.
	Fd int
	// Evs is the fd to event map.
	FdEvs map[int]*FdEvent
	// EpollEvs is the epoll events.
	EpollEvs []syscall.EpollEvent
	// SignalEvs is the signal events.
	SignalEvs map[int]*Event
	// SignalFd0 is the read end of the signal pipe.
	SignalFd0 int
	// SignalFd1 is the write end of the signal pipe.
	SignalFd1 int
	// SignalPoller is the signal poller.
	SignalPoller *SignalPoller
}

// NewEpoll creates a new epoll poller.
func NewEpoll() (*Epoll, error) {
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

	ep := &Epoll{
		Fd:        fd,
		FdEvs:     make(map[int]*FdEvent),
		EpollEvs:  make([]syscall.EpollEvent, 0xFF),
		SignalEvs: map[int]*Event{},
		SignalFd0: signalFds[0],
		SignalFd1: signalFds[1],
	}

	ep.SignalPoller = NewSignalPoller(ep.TriggerSignal)

	return ep, nil
}

// Add adds an event to the epoll poller.
// If the event is already added, it will be modified.
func (ep *Epoll) Add(ev *Event) error {
	if ev.Events&EvSignal != 0 {
		ep.SignalEvs[ev.Fd] = ev
		ep.SignalPoller.SubscribeSignal(ev.Fd)
		return nil
	}

	epEv := &syscall.EpollEvent{}
	op := syscall.EPOLL_CTL_ADD

	es, ok := ep.FdEvs[ev.Fd]
	if ok {
		op = syscall.EPOLL_CTL_MOD
	} else {
		es = &FdEvent{}
		ep.FdEvs[ev.Fd] = es
	}

	if ev.Events&EvRead != 0 {
		es.R = ev
	}
	if ev.Events&EvWrite != 0 {
		es.W = ev
	}
	if ev.Events&EvClosed != 0 {
		es.C = ev
	}
	if ev.Events&EvET != 0 {
		es.E = true
	}

	if es.R != nil {
		epEv.Events |= syscall.EPOLLIN
	}
	if es.W != nil {
		epEv.Events |= syscall.EPOLLOUT
	}
	if es.C != nil {
		epEv.Events |= syscall.EPOLLRDHUP
	}
	if es.E {
		epEv.Events |= syscall.EPOLLET
	}

	*(**FdEvent)(unsafe.Pointer(&epEv.Fd)) = es

	return syscall.EpollCtl(ep.Fd, op, ev.Fd, epEv)
}

// Del deletes an event from the epoll poller.
// If the event is not added, it will return an error.
// If the event is added twice, it will be modified.
func (ep *Epoll) Del(ev *Event) error {
	if ev.Events&EvSignal != 0 {
		delete(ep.SignalEvs, ev.Fd)
		ep.SignalPoller.UnsubscribeSignal(ev.Fd)
		return nil
	}

	es := ep.FdEvs[ev.Fd]

	if ev.Events&EvRead != 0 {
		es.R = nil
	}
	if ev.Events&EvWrite != 0 {
		es.W = nil
	}
	if ev.Events&EvClosed != 0 {
		es.C = nil
	}

	epEv := &syscall.EpollEvent{}
	op := syscall.EPOLL_CTL_DEL

	if es.R != nil {
		epEv.Events |= syscall.EPOLLIN
	}
	if es.W != nil {
		epEv.Events |= syscall.EPOLLOUT
	}
	if es.C != nil {
		epEv.Events |= syscall.EPOLLRDHUP
	}
	if es.E {
		epEv.Events |= syscall.EPOLLET
	}

	if epEv.Events == 0 {
		delete(ep.FdEvs, ev.Fd)
	} else {
		op = syscall.EPOLL_CTL_MOD
	}

	*(**FdEvent)(unsafe.Pointer(&epEv.Fd)) = es

	return syscall.EpollCtl(ep.Fd, op, ev.Fd, epEv)
}

// Polling polls the epoll poller.
// It will call the callback function when an event is ready.
// It will block until an event is ready.
func (ep *Epoll) Polling(cb func(ev *Event, res uint32), timeout int) error {
	n, err := syscall.EpollWait(ep.Fd, ep.EpollEvs, timeout)
	if err != nil && !TemporaryErr(err) {
		return err
	}

	for i := 0; i < n; i++ {
		if ep.EpollEvs[i].Fd == int32(ep.SignalFd0) {
			ep.OnSignal(cb)
			continue
		}

		var evRead, evWrite, evClosed *Event
		what := ep.EpollEvs[i].Events
		es := *(**FdEvent)(unsafe.Pointer(&ep.EpollEvs[i].Fd))

		if what&syscall.EPOLLERR != 0 {
			evRead = es.R
			evWrite = es.W
		} else if what&syscall.EPOLLHUP != 0 && what&syscall.EPOLLRDHUP == 0 {
			evRead = es.R
			evWrite = es.W
		} else {
			if what&syscall.EPOLLIN != 0 {
				evRead = es.R
			}
			if what&syscall.EPOLLOUT != 0 {
				evWrite = es.W
			}
			if what&syscall.EPOLLRDHUP != 0 {
				evClosed = es.C
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

// Close closes the epoll poller.
func (ep *Epoll) Close() error {
	ep.SignalPoller.Close()
	syscall.Close(ep.SignalFd0)
	syscall.Close(ep.SignalFd1)
	return syscall.Close(ep.Fd)
}

// Trigger triggers a signal.
func (ep *Epoll) TriggerSignal(signal int) {
	syscall.Write(ep.SignalFd1, []byte{byte(signal)})
}

// OnSignal is the callback function when a signal is received.
func (ep *Epoll) OnSignal(cb func(ev *Event, res uint32)) {
	buf := make([]byte, 1)
	syscall.Read(ep.SignalFd0, buf)

	ev, ok := ep.SignalEvs[int(buf[0])]
	if !ok {
		return
	}

	cb(ev, EvSignal)
}
