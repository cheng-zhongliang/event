// Copyright (c) 2023 cheng-zhongliang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin || dragonfly || freebsd || netbsd || openbsd
// +build darwin dragonfly freebsd netbsd openbsd

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

type poll struct {
	fd      int
	changes []syscall.Kevent_t
	events  []syscall.Kevent_t
}

func openPoll() (*poll, error) {
	kq := new(poll)
	fd, err := syscall.Kqueue()
	if err != nil {
		return nil, err
	}
	kq.fd = fd
	kq.changes = make([]syscall.Kevent_t, initialNEvent)
	kq.events = make([]syscall.Kevent_t, initialNEvent)
	return kq, nil
}

func (kq *poll) add(ev *Event) error {
	if ev.events&EvRead != 0 {
		kq.changes = append(kq.changes, syscall.Kevent_t{
			Ident:  uint64(ev.fd),
			Filter: syscall.EVFILT_READ,
			Flags:  syscall.EV_ADD,
			Udata:  (*byte)(unsafe.Pointer(ev)),
		})
	}
	if ev.events&EvWrite != 0 {
		kq.changes = append(kq.changes, syscall.Kevent_t{
			Ident:  uint64(ev.fd),
			Filter: syscall.EVFILT_WRITE,
			Flags:  syscall.EV_ADD,
			Udata:  (*byte)(unsafe.Pointer(ev)),
		})
	}
	return nil
}

func (kq *poll) del(ev *Event) error {
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

func (kq *poll) wait(cb func(ev *Event, res uint32), timeout time.Duration) error {
	ts := syscall.NsecToTimespec(timeout.Nanoseconds())
	n, err := syscall.Kevent(kq.fd, kq.changes, kq.events, &ts)
	if err != nil && !temporaryErr(err) {
		return err
	}
	kq.changes = kq.changes[:0]
	for i := 0; i < n; i++ {
		flags := kq.events[i].Flags
		if flags&syscall.EV_ERROR != 0 {
			errno := syscall.Errno(kq.events[i].Data)
			if errno&(syscall.EBADF|syscall.ENOENT|syscall.EINVAL) != 0 {
				continue
			}
			return errno
		}
		which := uint32(0)
		what := kq.events[i].Filter
		ev := (*Event)(unsafe.Pointer(kq.events[i].Udata))
		if what&syscall.EVFILT_READ != 0 {
			which |= EvRead
		} else if what&syscall.EVFILT_WRITE != 0 {
			which |= EvWrite
		}
		cb(ev, ev.events&which)
	}
	if n == len(kq.events) && n < maxNEvent {
		kq.events = make([]syscall.Kevent_t, n<<1)
	}
	return nil
}

func (kq *poll) close() error {
	return syscall.Close(kq.fd)
}
