package event

import (
	"syscall"
)

type Epoll struct {
	Fd  int
	Evs map[int]*struct {
		R *Event
		W *Event
	}
	EpollEvs []syscall.EpollEvent
}

func NewEpoll() (*Epoll, error) {
	fd, err := syscall.EpollCreate1(syscall.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}

	return &Epoll{
		Fd:       fd,
		Evs:      make(map[int]*struct{ R, W *Event }),
		EpollEvs: make([]syscall.EpollEvent, 1024),
	}, nil
}

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
	}

	if ev.Events&EvRead != 0 {
		epEv.Events |= syscall.EPOLLIN
		ep.Evs[ev.Fd].R = ev
	}
	if ev.Events&EvWrite != 0 {
		epEv.Events |= syscall.EPOLLOUT
		ep.Evs[ev.Fd].W = ev
	}

	return syscall.EpollCtl(ep.Fd, op, ev.Fd, epEv)
}

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

func (ep *Epoll) Polling(cb func(ev *Event, activeEvs uint32)) error {
	n, err := syscall.EpollWait(ep.Fd, ep.EpollEvs, -1)
	if err != nil {
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
