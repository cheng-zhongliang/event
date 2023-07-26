package event

import (
	"encoding/binary"
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
	// SignalEvs is the signal events.
	SignalEvs map[int]*Event
	// SignalFd0 is the read end of the signal pipe.
	SignalFd0 int
	// SignalFd1 is the write end of the signal pipe.
	SignalFd1 int
	// OnActive is the callback function when event trigger.
	OnActive func(ev *Event, res uint32)
	// ExitCh is the exit channel.
	ExitCh chan struct{}
	// Wg is the wait group.
	Wg *sync.WaitGroup
}

// NewEpoll creates a new epoll poller.
func NewEpoll(onActive func(ev *Event, res uint32)) (*Epoll, error) {
	fd, err := syscall.EpollCreate1(syscall.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}

	ep := &Epoll{
		Fd:        fd,
		FdEvs:     make(map[int]*FdEvent),
		EpollEvs:  make([]syscall.EpollEvent, 0xFF),
		OnActive:  onActive,
		SignalEvs: map[int]*Event{},
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
		ep.SignalEvs[ev.Fd] = ev
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
	if ev.Events&EvSignal != 0 {
		delete(ep.SignalEvs, ev.Fd)
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
func (ep *Epoll) Polling(timeout int) error {
	n, err := syscall.EpollWait(ep.Fd, ep.EpollEvs, timeout)
	if err != nil && !TemporaryErr(err) {
		return err
	}

	for i := 0; i < n; i++ {
		if ep.EpollEvs[i].Fd == int32(ep.SignalFd0) {
			ep.OnSignal(ep.SignalFd0)
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
			ep.OnActive(evRead, EvRead)
		}
		if evWrite != nil {
			ep.OnActive(evWrite, EvWrite)
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
		ch := make(chan os.Signal, 1)
		signal.Notify(ch)
		for {
			select {
			case sig := <-ch:
				sigNum := sig.(syscall.Signal)
				buf := make([]byte, binary.MaxVarintLen64)
				n := binary.PutUvarint(buf, uint64(sigNum))
				syscall.Write(ep.SignalFd1, buf[:n])
			case <-ep.ExitCh:
				return
			}
		}
	}()

	return syscall.EpollCtl(ep.Fd, syscall.EPOLL_CTL_ADD, ep.SignalFd0, &syscall.EpollEvent{Events: syscall.EPOLLIN})
}

// OnSignal is the callback function when a signal is received.
func (ep *Epoll) OnSignal(fd int) {
	buf := make([]byte, binary.MaxVarintLen64)
	syscall.Read(fd, buf)

	sigNum, _ := binary.Uvarint(buf)
	ev, ok := ep.SignalEvs[int(sigNum)]
	if !ok {
		return
	}

	ep.OnActive(ev, EvSignal)
}
