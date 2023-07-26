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
	// E is the Edge-triggered behavior.
	E uint32
}

// Epoll is the epoll poller implementation.
type Epoll struct {
	// Fd is the file descriptor of epoll.
	Fd int
	// Evs is the fd to event map.
	// It is used to find the event by fd.
	// It is also used to prevent the event from being garbage collected.
	FdEvs map[int]*FdEvent
	// EpollEvs is the epoll events.
	EpollEvs []syscall.EpollEvent
}

// NewEpoll creates a new epoll poller.
func NewEpoll() (*Epoll, error) {
	fd, err := syscall.EpollCreate1(syscall.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}

	return &Epoll{
		Fd:       fd,
		FdEvs:    make(map[int]*FdEvent),
		EpollEvs: make([]syscall.EpollEvent, 0xFF),
	}, nil
}

// Add adds an event to the epoll poller.
// If the event is already added, it will be modified.
func (ep *Epoll) Add(ev *Event) error {
	epEv := &syscall.EpollEvent{}
	op := syscall.EPOLL_CTL_ADD

	es, ok := ep.FdEvs[ev.Fd]
	if ok {
		op = syscall.EPOLL_CTL_MOD
		if es.R != nil {
			epEv.Events |= syscall.EPOLLIN
		}
		if es.W != nil {
			epEv.Events |= syscall.EPOLLOUT
		}
	} else {
		es = &FdEvent{}
		ep.FdEvs[ev.Fd] = es
		if ev.Events&EvET != 0 {
			// see go issue 832
			es.E = syscall.EPOLLET & 0xFFFFFFFF
		}
	}

	*(**FdEvent)(unsafe.Pointer(&epEv.Fd)) = es

	epEv.Events |= es.E

	if ev.Events&EvRead != 0 {
		epEv.Events |= syscall.EPOLLIN
		es.R = ev
	}
	if ev.Events&EvWrite != 0 {
		epEv.Events |= syscall.EPOLLOUT
		es.W = ev
	}

	return syscall.EpollCtl(ep.Fd, op, ev.Fd, epEv)
}

// Del deletes an event from the epoll poller.
// If the event is not added, it will return an error.
// If the event is added twice, it will be modified.
func (ep *Epoll) Del(ev *Event) error {
	epEv := &syscall.EpollEvent{}
	op := syscall.EPOLL_CTL_DEL

	if ev.Events&EvRead != 0 {
		epEv.Events |= syscall.EPOLLIN
	}
	if ev.Events&EvWrite != 0 {
		epEv.Events |= syscall.EPOLLOUT
	}

	es, ok := ep.FdEvs[ev.Fd]
	if !ok {
		return syscall.EpollCtl(ep.Fd, op, ev.Fd, epEv)
	}

	*(**FdEvent)(unsafe.Pointer(&epEv.Fd)) = es

	epEv.Events |= es.E

	if epEv.Events&(syscall.EPOLLIN|syscall.EPOLLOUT) != (syscall.EPOLLIN | syscall.EPOLLOUT) {
		if epEv.Events&syscall.EPOLLIN != 0 && ep.FdEvs[ev.Fd].W != nil {
			op = syscall.EPOLL_CTL_MOD
			epEv.Events = syscall.EPOLLOUT
			ep.FdEvs[ev.Fd].R = nil
		} else if epEv.Events&syscall.EPOLLOUT != 0 && ep.FdEvs[ev.Fd].R != nil {
			op = syscall.EPOLL_CTL_MOD
			epEv.Events = syscall.EPOLLIN
			ep.FdEvs[ev.Fd].W = nil
		}
	}

	if op == syscall.EPOLL_CTL_DEL {
		delete(ep.FdEvs, ev.Fd)
	}

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
		var evRead, evWrite *Event
		what := ep.EpollEvs[i].Events
		es := *(**FdEvent)(unsafe.Pointer(&ep.EpollEvs[i].Fd))

		if what&syscall.EPOLLIN != 0 {
			evRead = es.R
		}
		if what&syscall.EPOLLOUT != 0 {
			evWrite = es.W
		}

		if evRead != nil {
			cb(evRead, EvRead)
		}
		if evWrite != nil {
			cb(evWrite, EvWrite)
		}
	}

	return nil
}

// Close closes the epoll poller.
func (ep *Epoll) Close() error {
	return syscall.Close(ep.Fd)
}
