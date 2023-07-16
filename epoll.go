package event

import (
	"syscall"
)

// Epoll is the epoll poller implementation.
type Epoll struct {
	// Fd is the file descriptor of epoll.
	Fd int
	// Evs is the fd to event map.
	Evs map[int]*struct {
		// R is the read event.
		R *Event
		// W is the write event.
		W *Event
	}
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
		Evs:      make(map[int]*struct{ R, W *Event }),
		EpollEvs: make([]syscall.EpollEvent, 0xFF),
	}, nil
}

// Add adds an event to the epoll poller.
// If the event is already added, it will be modified.
func (ep *Epoll) Add(ev *Event) error {
	epEv := &syscall.EpollEvent{Fd: int32(ev.Fd)}
	op := syscall.EPOLL_CTL_ADD

	es, ok := ep.Evs[ev.Fd]
	if ok {
		op = syscall.EPOLL_CTL_MOD
		if es.R != nil {
			epEv.Events |= syscall.EPOLLIN
		}
		if es.W != nil {
			epEv.Events |= syscall.EPOLLOUT
		}
	} else {
		es = &struct{ R, W *Event }{}
		ep.Evs[ev.Fd] = es
	}

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
	epEv := &syscall.EpollEvent{Fd: int32(ev.Fd)}
	op := syscall.EPOLL_CTL_DEL

	if ev.Events&EvRead != 0 {
		epEv.Events |= syscall.EPOLLIN
	}
	if ev.Events&EvWrite != 0 {
		epEv.Events |= syscall.EPOLLOUT
	}

	if epEv.Events&(syscall.EPOLLIN|syscall.EPOLLOUT) != (syscall.EPOLLIN | syscall.EPOLLOUT) {
		op = syscall.EPOLL_CTL_MOD
		if epEv.Events&syscall.EPOLLIN != 0 && ep.Evs[ev.Fd].W != nil {
			epEv.Events = syscall.EPOLLOUT
			ep.Evs[ev.Fd].R = nil
		} else if epEv.Events&syscall.EPOLLOUT != 0 && ep.Evs[ev.Fd].R != nil {
			epEv.Events = syscall.EPOLLIN
			ep.Evs[ev.Fd].W = nil
		}
	} else {
		delete(ep.Evs, ev.Fd)
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
		es, ok := ep.Evs[int(ep.EpollEvs[i].Fd)]
		if !ok {
			continue
		}

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
