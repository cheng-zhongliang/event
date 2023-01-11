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
	return unix.EpollCtl(ep.Fd, unix.EPOLL_CTL_ADD, ev.fd, ep.UnNormalizeEvent(ev).(*unix.EpollEvent))
}

func (ep *Epoll) DelEvent(ev Event) error {
	return unix.EpollCtl(ep.Fd, unix.EPOLL_CTL_DEL, ev.fd, nil)
}

func (ep *Epoll) ModEvent(ev Event) error {
	return unix.EpollCtl(ep.Fd, unix.EPOLL_CTL_MOD, ev.fd, ep.UnNormalizeEvent(ev).(*unix.EpollEvent))
}

func (ep *Epoll) WaitActiveEvents(activeEvents []Event) (n int, err error) {
	var events []unix.EpollEvent
	if nn := len(activeEvents); nn > 0 {
		events = make([]unix.EpollEvent, nn)
	} else {
		return 0, nil
	}

	n, err = unix.EpollWait(ep.Fd, events, -1)
	if err != nil && !TemporaryErr(err) {
		return 0, err
	}

	for i := 0; i < n; i++ {
		activeEvents[i] = ep.NormalizeEvent(&events[i])
	}

	return n, nil
}

func (ep *Epoll) NormalizeEvent(epollEv interface{}) (ev Event) {
	epEv, ok := epollEv.(*unix.EpollEvent)
	if !ok {
		return
	}

	ev.fd = int(epEv.Fd)

	if epEv.Events&unix.EPOLLIN != 0 {
		ev.flag |= EventRead
	}
	if epEv.Events&unix.EPOLLOUT != 0 {
		ev.flag |= EventWrite
	}

	return
}

func (ep *Epoll) UnNormalizeEvent(ev Event) (epollEv interface{}) {
	epEv := &unix.EpollEvent{}
	if ev.flag&EventRead != 0 {
		epEv.Events |= unix.EPOLLIN
	}
	if ev.flag&EventWrite != 0 {
		epEv.Events |= unix.EPOLLOUT
	}
	epEv.Fd = int32(ev.fd)

	return epEv
}
