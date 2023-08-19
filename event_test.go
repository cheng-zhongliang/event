/*
 * Copyright (c) 2023 cheng-zhongliang. All rights reserved.
 *
 * This source code is licensed under the 3-Clause BSD License, which can be
 * found in the accompanying "LICENSE" file, or at:
 * https://opensource.org/licenses/BSD-3-Clause
 *
 * By accessing, using, copying, modifying, or distributing this software,
 * you agree to the terms and conditions of the BSD 3-Clause License.
 */

package event_test

import (
	"syscall"
	"testing"
	"time"

	. "github.com/cheng-zhongliang/event"
)

func TestNewBase(t *testing.T) {
	base, err := NewBase()
	if err != nil {
		t.Fatal(err)
	}

	if err := base.Exit(); err != nil {
		t.Fatal(err)
	}
}

func TestAddEvent(t *testing.T) {
	base, err := NewBase()
	if err != nil {
		t.Fatal(err)
	}

	r0, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, 0, 0, 0)
	if errno != 0 {
		t.Fatal(errno)
	}

	ev := New(int(r0), EvRead, func(fd int, events uint32, arg interface{}) {}, nil)

	err = base.AddEvent(ev, 0)
	if err != nil {
		t.Fatal(err)
	}

	if err := base.Exit(); err != nil {
		t.Fatal(err)
	}

	syscall.Close(int(r0))
}

func TestDelEvent(t *testing.T) {
	base, err := NewBase()
	if err != nil {
		t.Fatal(err)
	}

	r0, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, 0, 0, 0)
	if errno != 0 {
		t.Fatal(errno)
	}

	ev := New(int(r0), EvRead, func(fd int, events uint32, arg interface{}) {}, nil)

	err = base.AddEvent(ev, 0)
	if err != nil {
		t.Fatal(err)
	}

	err = base.DelEvent(ev)
	if err != nil {
		t.Fatal(err)
	}

	if err := base.Exit(); err != nil {
		t.Fatal(err)
	}

	syscall.Close(int(r0))
}

func TestEventDispatch(t *testing.T) {
	base, err := NewBase()
	if err != nil {
		t.Fatal(err)
	}

	r0, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, 0, 0, 0)
	if errno != 0 {
		t.Fatal(errno)
	}

	ev := New(int(r0), EvRead, func(fd int, events uint32, arg interface{}) {
		if fd != int(r0) {
			t.Fatal("fd not equal")
		}
		if events != EvRead {
			t.Fatal("events not equal")
		}
		if arg != "hello" {
			t.Fatal("arg not equal")
		}
		syscall.Read(int(r0), make([]byte, 8))
		if err := base.Exit(); err != nil {
			t.Fatal(err)
		}
	}, "hello")

	err = base.AddEvent(ev, 0)
	if err != nil {
		t.Fatal(err)
	}

	_, err = syscall.Write(int(r0), []byte{0, 0, 0, 0, 0, 0, 0, 1})
	if err != nil {
		t.Fatal(err)
	}

	err = base.Dispatch()
	if err != nil && err != ErrBadFileDescriptor {
		t.Fatal(err)
	}

	syscall.Close(int(r0))
}

func TestEventTimeout(t *testing.T) {
	base, err := NewBase()
	if err != nil {
		t.Fatal(err)
	}

	r0, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, 0, 0, 0)
	if errno != 0 {
		t.Fatal(errno)
	}

	n := 0
	ev := New(int(r0), EvRead|EvTimeout, func(fd int, events uint32, arg interface{}) {
		if fd != int(r0) {
			t.Fatal("fd not equal")
		}
		if events != EvTimeout {
			t.Fatal("events not equal")
		}
		if arg != "hello" {
			t.Fatal("arg not equal")
		}
		if err := base.Exit(); err != nil {
			t.Fatal(err)
		}
		n++
	}, "hello")

	err = base.AddEvent(ev, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	err = base.Dispatch()
	if err != nil && err != ErrBadFileDescriptor {
		t.Fatal(err)
	}

	if n != 1 {
		t.FailNow()
	}

	syscall.Close(int(r0))
}

func TestTimer(t *testing.T) {
	base, err := NewBase()
	if err != nil {
		t.Fatal(err)
	}

	n := 0
	ev := New(-1, EvTimeout, func(fd int, events uint32, arg interface{}) {
		if events != EvTimeout {
			t.Fatal("events not equal")
		}
		if arg != "hello" {
			t.Fatal("arg not equal")
		}
		if err := base.Exit(); err != nil {
			t.Fatal(err)
		}
		n++
	}, "hello")

	err = base.AddEvent(ev, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	err = base.Dispatch()
	if err != nil && err != ErrBadFileDescriptor {
		t.Fatal(err)
	}

	if n != 1 {
		t.FailNow()
	}
}

func TestTicker(t *testing.T) {
	base, err := NewBase()
	if err != nil {
		t.Fatal(err)
	}

	n := 0
	ev := New(-1, EvTimeout|EvPersist, func(fd int, events uint32, arg interface{}) {
		if events != EvTimeout {
			t.Fatal("events not equal")
		}
		if arg != "hello" {
			t.Fatal("arg not equal")
		}
		n++
		if n == 3 {
			if err := base.Exit(); err != nil {
				t.Fatal(err)
			}
		}
	}, "hello")

	err = base.AddEvent(ev, 5*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	err = base.Dispatch()
	if err != nil && err != ErrBadFileDescriptor {
		t.Fatal(err)
	}

	if n != 3 {
		t.FailNow()
	}
}

func TestPriority(t *testing.T) {
	base, err := NewBase()
	if err != nil {
		t.Fatal(err)
	}

	r0, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, 0, 0, 0)
	if errno != 0 {
		t.Fatal(errno)
	}

	r1, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, 0, 0, 0)
	if errno != 0 {
		t.Fatal(errno)
	}

	_, err = syscall.Write(int(r0), []byte{0, 0, 0, 0, 0, 0, 0, 1})
	if err != nil {
		t.Fatal(err)
	}

	_, err = syscall.Write(int(r1), []byte{0, 0, 0, 0, 0, 0, 0, 1})
	if err != nil {
		t.Fatal(err)
	}

	triggerTime0 := 0
	ev0 := New(int(r0), EvRead, func(fd int, events uint32, arg interface{}) {
		if fd != int(r0) {
			t.Fatal("fd not equal")
		}
		if events != EvRead {
			t.Fatal("events not equal")
		}
		if arg != "hello" {
			t.Fatal("arg not equal")
		}
		syscall.Read(int(r0), make([]byte, 8))
		triggerTime0 = int(time.Now().UnixMicro())
		if err := base.Exit(); err != nil {
			t.Fatal(err)
		}
	}, "hello")

	triggerTime1 := 0
	ev1 := New(int(r1), EvRead, func(fd int, events uint32, arg interface{}) {
		if fd != int(r1) {
			t.Fatal("fd not equal")
		}
		if events != EvRead {
			t.Fatal("events not equal")
		}
		if arg != "hello" {
			t.Fatal("arg not equal")
		}
		syscall.Read(int(r1), make([]byte, 8))
		triggerTime1 = int(time.Now().UnixMicro())
	}, "hello")
	ev1.SetPriority(HPriority)

	err = base.AddEvent(ev0, 0)
	if err != nil {
		t.Fatal(err)
	}

	err = base.AddEvent(ev1, 0)
	if err != nil {
		t.Fatal(err)
	}

	err = base.Dispatch()
	if err != nil && err != ErrBadFileDescriptor {
		t.Fatal(err)
	}

	if triggerTime0 <= triggerTime1 {
		t.FailNow()
	}

	syscall.Close(int(r0))
	syscall.Close(int(r1))
}

func TestEdgeTrigger(t *testing.T) {
	base, err := NewBase()
	if err != nil {
		t.Fatal(err)
	}

	r0, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, 0, 0, 0)
	if errno != 0 {
		t.Fatal(errno)
	}

	_, err = syscall.Write(int(r0), []byte{0, 0, 0, 0, 0, 0, 0, 1})
	if err != nil {
		t.Fatal(err)
	}

	n := 0
	ev := New(int(r0), EvRead|EvTimeout|EvPersist|EvET, func(fd int, events uint32, arg interface{}) {
		if events&EvTimeout != 0 {
			if err := base.Exit(); err != nil {
				t.Fatal(err)
			}
			return
		}
		if events&EvET == 0 {
			t.FailNow()
		}
		n++
	}, "hello")

	err = base.AddEvent(ev, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	err = base.Dispatch()
	if err != nil && err != ErrBadFileDescriptor {
		t.Fatal(err)
	}

	if n != 1 {
		t.FailNow()
	}

	syscall.Close(int(r0))
}

func BenchmarkEventAdd(b *testing.B) {
	receivers := make([]int, b.N)
	base, err := NewBase()
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
		if err != nil {
			b.Fatal(err)
		}
		receivers[i] = fds[0]
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ev := New(receivers[i], EvRead, func(fd int, events uint32, arg interface{}) {}, nil)
		err = base.AddEvent(ev, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()

	if err := base.Exit(); err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		syscall.Close(receivers[i])
	}
}

func BenchmarkEventLoop(b *testing.B) {
	receiver := make([]int, b.N)
	senders := make([]int, b.N)

	base, err := NewBase()
	if err != nil {
		b.Fatal(err)
	}

	buf := make([]byte, 1)
	fires := 0

	for i := 0; i < b.N; i++ {
		fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
		if err != nil {
			b.Fatal(err)
		}
		receiver[i] = fds[0]
		senders[i] = fds[1]

		ev := New(fds[0], EvRead|EvPersist, func(fd int, events uint32, arg interface{}) {
			syscall.Read(fd, buf)
			fires++
		}, nil)
		err = base.AddEvent(ev, 0)
		if err != nil {
			b.Fatal(err)
		}

		_, err = syscall.Write(fds[1], []byte{'e'})
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for fires < b.N {
		err = base.Loop(EvLoopOnce | EvLoopNoblock)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()

	for i := 0; i < b.N; i++ {
		syscall.Close(receiver[i])
		syscall.Close(senders[i])
	}

	if err := base.Exit(); err != nil {
		b.Fatal(err)
	}
}
