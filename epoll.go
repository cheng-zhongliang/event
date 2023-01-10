package unicorn

import "golang.org/x/sys/unix"

type Epoll struct {
	Fd int
}

func NewEpoll() (*Epoll, error) {
	fd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}
	return &Epoll{
		Fd: fd,
	}, nil
}

func (ep *Epoll) AddEvent(ev Event) error {
	return unix.EpollCtl(ep.Fd, unix.EPOLL_CTL_ADD, ev.fd, &unix.EpollEvent{Events: uint32(ev.flag), Fd: int32(ev.fd)})
}

func (ep *Epoll) DelEvent(ev Event) error {
	return unix.EpollCtl(ep.Fd, unix.EPOLL_CTL_DEL, ev.fd, nil)
}

func (ep *Epoll) ModEvent(ev Event) error {
	return unix.EpollCtl(ep.Fd, unix.EPOLL_CTL_MOD, ev.fd, &unix.EpollEvent{Events: uint32(ev.flag), Fd: int32(ev.fd)})
}

func (ep *Epoll) WaitActiveEvents(activeEvents []Event) (n int, err error) {
	var events []unix.EpollEvent
	if nn := len(activeEvents); nn > 0 {
		events = make([]unix.EpollEvent, nn)
	} else {
		return 0, nil
	}

	n, err = unix.EpollWait(ep.Fd, events, -1)
	if err != nil {
		return 0, err
	}

	for i := 0; i < n; i++ {
		activeEvents[i] = Event{
			fd:   int(events[i].Fd),
			flag: EventFlag(events[i].Events),
		}
	}

	return n, nil
}
