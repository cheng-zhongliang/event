//go:build darwin || dragonfly || freebsd || netbsd || openbsd
// +build darwin dragonfly freebsd netbsd openbsd

// Copyright (c) 2023 cheng-zhongliang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"syscall"
	"time"
	"unsafe"
)

const (
	initialNEvent = 0x20
	maxNEvent     = 0x1000
)

type poller struct {
	fd      int
	changes []syscall.Kevent_t
	events  []syscall.Kevent_t
}

func newPoller() (*poller, error) {
	fd, err := syscall.Kqueue()
	if err != nil {
		return nil, err
	}

	return &poller{
		fd:      fd,
		changes: make([]syscall.Kevent_t, initialNEvent),
		events:  make([]syscall.Kevent_t, initialNEvent),
	}, nil
}

func (kq *poller) add(ev *Event) error {
	ET := uint16(0)
	if ev.events&EvET != 0 {
		ET = syscall.EV_CLEAR
	}
	if ev.events&EvRead != 0 {
		kq.changes = append(kq.changes, syscall.Kevent_t{
			Ident:  uint64(ev.fd),
			Filter: syscall.EVFILT_READ,
			Flags:  syscall.EV_ADD | syscall.EV_ENABLE | ET,
			Udata:  (*byte)(unsafe.Pointer(ev)),
		})
	}
	if ev.events&EvWrite != 0 {
		kq.changes = append(kq.changes, syscall.Kevent_t{
			Ident:  uint64(ev.fd),
			Filter: syscall.EVFILT_WRITE,
			Flags:  syscall.EV_ADD | syscall.EV_ENABLE | ET,
			Udata:  (*byte)(unsafe.Pointer(ev)),
		})
	}
	return nil
}

func (kq *poller) del(ev *Event) error {
	if ev.events&EvRead != 0 {
		kq.changes = append(kq.changes, syscall.Kevent_t{
			Ident:  uint64(ev.fd),
			Filter: syscall.EVFILT_READ,
			Flags:  syscall.EV_DELETE,
		})
	}
	if ev.events&EvWrite != 0 {
		kq.changes = append(kq.changes, syscall.Kevent_t{
			Ident:  uint64(ev.fd),
			Filter: syscall.EVFILT_WRITE,
			Flags:  syscall.EV_DELETE,
		})
	}
	return nil
}

func (kq *poller) polling(cb func(ev *Event, res uint32), timeout time.Duration) error {
	timespec := syscall.NsecToTimespec(timeout.Nanoseconds())
	n, err := syscall.Kevent(kq.fd, kq.changes, kq.events, &timespec)
	if err != nil && !temporaryErr(err) {
		return err
	}

	kq.changes = kq.changes[:0]

	for i := 0; i < n; i++ {
		which := uint32(0)
		what := kq.events[i].Filter
		ev := (*Event)(unsafe.Pointer(kq.events[i].Udata))

		switch what {
		case syscall.EVFILT_READ:
			which |= EvRead
		case syscall.EVFILT_WRITE:
			which |= EvWrite
		}

		cb(ev, ev.events&(which|EvET))
	}

	if n == len(kq.events) && n < maxNEvent {
		kq.events = make([]syscall.Kevent_t, n<<1)
	}

	return nil
}

func (kq *poller) close() error {
	return syscall.Close(kq.fd)
}
