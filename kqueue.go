//go:build darwin || dragonfly || freebsd || netbsd || openbsd
// +build darwin dragonfly freebsd netbsd openbsd

// Copyright (c) 2023 cheng-zhongliang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"syscall"
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

func (kp *poller) add(ev *Event) error {
	ET := uint16(0)
	if ev.events&EvET != 0 {
		ET = syscall.EV_CLEAR
	}
	if ev.events&EvRead != 0 {
		kp.changes = append(kp.changes, syscall.Kevent_t{
			Ident:  uint64(ev.fd),
			Filter: syscall.EVFILT_READ,
			Flags:  syscall.EV_ADD | syscall.EV_ENABLE | ET,
			Udata:  (*byte)(unsafe.Pointer(ev)),
		})
	}
	if ev.events&EvWrite != 0 {
		kp.changes = append(kp.changes, syscall.Kevent_t{
			Ident:  uint64(ev.fd),
			Filter: syscall.EVFILT_WRITE,
			Flags:  syscall.EV_ADD | syscall.EV_ENABLE | ET,
			Udata:  (*byte)(unsafe.Pointer(ev)),
		})
	}
	return nil
}

func (kp *poller) del(ev *Event) error {
	if ev.events&EvRead != 0 {
		kp.changes = append(kp.changes, syscall.Kevent_t{
			Ident:  uint64(ev.fd),
			Filter: syscall.EVFILT_READ,
		})
	}
	if ev.events&EvWrite != 0 {
		kp.changes = append(kp.changes, syscall.Kevent_t{
			Ident:  uint64(ev.fd),
			Filter: syscall.EVFILT_WRITE,
		})
	}
	return nil
}

func (kp *poller) polling(cb func(ev *Event, res uint32), timeout int) error {
	n, err := syscall.Kevent(kp.fd, kp.changes, kp.events, nil)
	if err != nil && !temporaryErr(err) {
		return err
	}

	kp.changes = kp.changes[:0]

	for i := 0; i < n; i++ {
		which := uint32(0)

		ev := (*Event)(unsafe.Pointer(kp.events[i].Udata))

		switch kp.events[i].Filter {
		case syscall.EVFILT_READ:
			which |= EvRead
		case syscall.EVFILT_WRITE:
			which |= EvWrite
		}

		cb(ev, which)
	}

	if n == len(kp.events) && n < maxNEvent {
		kp.events = make([]syscall.Kevent_t, n*2)
	}

	return nil
}

func (kp *poller) close() error {
	return syscall.Close(kp.fd)
}
