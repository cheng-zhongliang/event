package event

import (
	"os"
	"os/signal"
	"sync"
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
	E bool
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
	// SignalEvs is the signal events.
	SignalEvs map[int]*Event
	// SignalCh is the signal channel.
	SignalCh chan os.Signal
	// SignalFd0 is the read end of the signal pipe.
	SignalFd0 int
	// SignalFd1 is the write end of the signal pipe.
	SignalFd1 int
	// ExitCh is the exit channel.
	ExitCh chan struct{}
	// Wg is the wait group.
	Wg *sync.WaitGroup
}

// NewEpoll creates a new epoll poller.
func NewEpoll() (*Epoll, error) {
	fd, err := syscall.EpollCreate1(syscall.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}

	ep := &Epoll{
		Fd:        fd,
		FdEvs:     make(map[int]*FdEvent),
		EpollEvs:  make([]syscall.EpollEvent, 0xFF),
		SignalEvs: map[int]*Event{},
		SignalCh:  make(chan os.Signal, 1),
		ExitCh:    make(chan struct{}),
		Wg:        &sync.WaitGroup{},
	}

	if err := ep.InitSignalPoll(); err != nil {
		return nil, err
	}

	return ep, nil
}

// Add adds an event to the epoll poller.
// If the event is already added, it will be modified.
func (ep *Epoll) Add(ev *Event) error {
	if ev.Events&EvSignal != 0 {
		ep.SubscribeSignal(ev)
		return nil
	}

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
			es.E = true
		}
	}

	*(**FdEvent)(unsafe.Pointer(&epEv.Fd)) = es

	if es.E {
		epEv.Events |= syscall.EPOLLET & 0xFFFFFFFF
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
	if ev.Events&EvSignal != 0 {
		ep.UnsubscribeSignal(ev)
		return nil
	}

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

	if ev.Events&EvET == 0 && es.E {
		epEv.Events |= syscall.EPOLLET & 0xFFFFFFFF
	} else {
		es.E = false
	}

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
		if ep.EpollEvs[i].Fd == int32(ep.SignalFd0) {
			ep.OnSignal(cb, ep.SignalFd0)
			continue
		}

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
	close(ep.ExitCh)
	ep.Wg.Wait()
	return syscall.Close(ep.Fd)
}

// InitSignalPoll initializes the signal poll.
func (ep *Epoll) InitSignalPoll() error {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_SEQPACKET|syscall.SOCK_CLOEXEC, 0)
	if err != nil {
		return err
	}

	ep.SignalFd0 = fds[0]
	ep.SignalFd1 = fds[1]

	ep.Wg.Add(1)
	go func() {
		defer ep.Wg.Done()
		for {
			select {
			case sig := <-ep.SignalCh:
				syscall.Write(ep.SignalFd1, []byte{byte(sig.(syscall.Signal))})
			case <-ep.ExitCh:
				return
			}
		}
	}()

	return syscall.EpollCtl(ep.Fd, syscall.EPOLL_CTL_ADD, ep.SignalFd0, &syscall.EpollEvent{Events: syscall.EPOLLIN, Fd: int32(ep.SignalFd0)})
}

// OnSignal is the callback function when a signal is received.
func (ep *Epoll) OnSignal(cb func(ev *Event, res uint32), fd int) {
	buf := make([]byte, 1)
	syscall.Read(fd, buf)

	ev, ok := ep.SignalEvs[int(buf[0])]
	if !ok {
		return
	}

	cb(ev, EvSignal)
}

// SubscribeSignal subscribes the signal.
func (ep *Epoll) SubscribeSignal(ev *Event) {
	ep.SignalEvs[ev.Fd] = ev
	signal.Stop(ep.SignalCh)
	signals := make([]os.Signal, 0, len(ep.SignalEvs))
	for s := range ep.SignalEvs {
		signals = append(signals, syscall.Signal(s))
	}
	signal.Notify(ep.SignalCh, signals...)
}

// UnsubscribeSignal unsubscribes the signal.
func (ep *Epoll) UnsubscribeSignal(ev *Event) {
	delete(ep.SignalEvs, ev.Fd)
	signal.Stop(ep.SignalCh)
	signals := make([]os.Signal, 0, len(ep.SignalEvs))
	for s := range ep.SignalEvs {
		signals = append(signals, syscall.Signal(s))
	}
	signal.Notify(ep.SignalCh, signals...)
}
