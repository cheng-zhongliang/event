package unicorn

import "golang.org/x/sys/unix"

type Epoller struct {
	Fd int
}

func NewEpoller() (*Epoller, error) {
	fd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}
	return &Epoller{
		Fd: fd,
	}, nil
}

func (ep *Epoller) AddEvent(ev Event) error {
	return unix.EpollCtl(ep.Fd, unix.EPOLL_CTL_ADD, ev.Fd, ep.stdEv2epEv(ev))
}

func (ep *Epoller) DelEvent(ev Event) error {
	return unix.EpollCtl(ep.Fd, unix.EPOLL_CTL_DEL, ev.Fd, nil)
}

func (ep *Epoller) ModEvent(ev Event) error {
	return unix.EpollCtl(ep.Fd, unix.EPOLL_CTL_MOD, ev.Fd, ep.stdEv2epEv(ev))
}

func (ep *Epoller) WaitActiveEvents(activeEvents []Event) (n int, err error) {
	var events []unix.EpollEvent
	if nn := len(activeEvents); nn > 0 {
		events = make([]unix.EpollEvent, nn)
	} else {
		return
	}

	n, err = unix.EpollWait(ep.Fd, events, -1)
	if err != nil && !TemporaryErr(err) {
		return
	}

	for i := 0; i < n; i++ {
		activeEvents[i] = ep.epEv2StdEv(events[i])
	}

	return
}

func (ep *Epoller) epEv2StdEv(epEv unix.EpollEvent) (stdEv Event) {
	stdEv.Fd = int(epEv.Fd)
	if epEv.Events&unix.EPOLLIN != 0 {
		stdEv.Flag |= EventRead
	}
	if epEv.Events&unix.EPOLLOUT != 0 {
		stdEv.Flag |= EventWrite
	}
	return
}

func (ep *Epoller) stdEv2epEv(stdEv Event) (epEv *unix.EpollEvent) {
	if stdEv.Flag&EventRead != 0 {
		epEv.Events |= unix.EPOLLIN
	}
	if stdEv.Flag&EventWrite != 0 {
		epEv.Events |= unix.EPOLLOUT
	}
	epEv.Fd = int32(stdEv.Fd)
	return
}
