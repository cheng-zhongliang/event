// Copyright (c) 2023 cheng-zhongliang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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

	if err := base.Shutdown(); err != nil {
		t.Fatal(err)
	}
}

func TestAddEvent(t *testing.T) {
	base, err := NewBase()
	if err != nil {
		t.Fatal(err)
	}

	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Fatal(err)
	}

	ev := New(base, fds[0], EvRead, func(fd int, events uint32, arg interface{}) {}, nil)

	err = ev.Attach(0)
	if err != nil {
		t.Fatal(err)
	}

	if err := base.Shutdown(); err != nil {
		t.Fatal(err)
	}

	syscall.Close(fds[0])
	syscall.Close(fds[1])
}

func TestDelEvent(t *testing.T) {
	base, err := NewBase()
	if err != nil {
		t.Fatal(err)
	}

	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Fatal(err)
	}

	ev := New(base, fds[0], EvRead, func(fd int, events uint32, arg interface{}) {}, nil)

	err = ev.Attach(0)
	if err != nil {
		t.Fatal(err)
	}

	err = ev.Detach()
	if err != nil {
		t.Fatal(err)
	}

	if err := base.Shutdown(); err != nil {
		t.Fatal(err)
	}

	syscall.Close(fds[0])
	syscall.Close(fds[1])
}

func TestEventLoop(t *testing.T) {
	base, err := NewBase()
	if err != nil {
		t.Fatal(err)
	}

	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Fatal(err)
	}

	ev := New(base, fds[0], EvRead, func(fd int, events uint32, arg interface{}) {
		if fd != fds[0] {
			t.Fatal("fd not equal")
		}
		if events != EvRead {
			t.Fatal("events not equal")
		}
		if arg != "hello" {
			t.Fatal("arg not equal")
		}
		syscall.Read(fds[0], make([]byte, 8))
		if err := base.Shutdown(); err != nil {
			t.Fatal(err)
		}
	}, "hello")

	err = ev.Attach(0)
	if err != nil {
		t.Fatal(err)
	}

	_, err = syscall.Write(fds[1], []byte{0, 0, 0, 0, 0, 0, 0, 1})
	if err != nil {
		t.Fatal(err)
	}

	err = base.Dispatch()
	if err != nil && err != syscall.EBADF {
		t.Fatal(err)
	}

	syscall.Close(fds[0])
	syscall.Close(fds[1])
}

func TestEventTimeout(t *testing.T) {
	base, err := NewBase()
	if err != nil {
		t.Fatal(err)
	}

	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Fatal(err)
	}

	n := 0
	ev := New(base, fds[0], EvRead|EvTimeout, func(fd int, events uint32, arg interface{}) {
		if fd != fds[0] {
			t.Fatal("fd not equal")
		}
		if events != EvTimeout {
			t.Fatal("events not equal")
		}
		if arg != "hello" {
			t.Fatal("arg not equal")
		}
		if err := base.Shutdown(); err != nil {
			t.Fatal(err)
		}
		n++
	}, "hello")

	err = ev.Attach(10 * time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	err = base.Dispatch()
	if err != nil && err != syscall.EBADF {
		t.Fatal(err)
	}

	if n != 1 {
		t.FailNow()
	}

	syscall.Close(fds[0])
	syscall.Close(fds[1])
}

func TestTimer(t *testing.T) {
	base, err := NewBase()
	if err != nil {
		t.Fatal(err)
	}

	n := 0
	ev := NewTimer(base, func(fd int, events uint32, arg interface{}) {
		if events != EvTimeout {
			t.Fatal("events not equal")
		}
		if arg != "hello" {
			t.Fatal("arg not equal")
		}
		if err := base.Shutdown(); err != nil {
			t.Fatal(err)
		}
		n++
	}, "hello")

	err = ev.Attach(10 * time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	err = base.Dispatch()
	if err != nil && err != syscall.EBADF {
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
	ev := NewTicker(base, func(fd int, events uint32, arg interface{}) {
		if events != EvTimeout {
			t.Fatal("events not equal")
		}
		if arg != "hello" {
			t.Fatal("arg not equal")
		}
		n++
		if n == 3 {
			if err := base.Shutdown(); err != nil {
				t.Fatal(err)
			}
		}
	}, "hello")

	err = ev.Attach(5 * time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	err = base.Dispatch()
	if err != nil && err != syscall.EBADF {
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

	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Fatal(err)
	}

	fds1, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Fatal(err)
	}

	_, err = syscall.Write(fds[1], []byte{0, 0, 0, 0, 0, 0, 0, 1})
	if err != nil {
		t.Fatal(err)
	}

	_, err = syscall.Write(fds1[1], []byte{0, 0, 0, 0, 0, 0, 0, 1})
	if err != nil {
		t.Fatal(err)
	}

	triggerTime0 := 0
	ev0 := New(base, fds[0], EvRead, func(fd int, events uint32, arg interface{}) {
		if fd != fds[0] {
			t.Fatal("fd not equal")
		}
		if events != EvRead {
			t.Fatal("events not equal")
		}
		if arg != "hello" {
			t.Fatal("arg not equal")
		}
		syscall.Read(fds[0], make([]byte, 8))
		triggerTime0 = int(time.Now().UnixMicro())
		if err := base.Shutdown(); err != nil {
			t.Fatal(err)
		}
	}, "hello")

	triggerTime1 := 0
	ev1 := New(base, fds1[0], EvRead, func(fd int, events uint32, arg interface{}) {
		if fd != fds1[0] {
			t.Fatal("fd not equal")
		}
		if events != EvRead {
			t.Fatal("events not equal")
		}
		if arg != "hello" {
			t.Fatal("arg not equal")
		}
		syscall.Read(fds1[0], make([]byte, 8))
		triggerTime1 = int(time.Now().UnixMicro())
	}, "hello")
	ev1.SetPriority(HP)

	err = ev0.Attach(0)
	if err != nil {
		t.Fatal(err)
	}

	err = ev1.Attach(0)
	if err != nil {
		t.Fatal(err)
	}

	err = base.Dispatch()
	if err != nil && err != syscall.EBADF {
		t.Fatal(err)
	}

	if triggerTime0 <= triggerTime1 {
		t.FailNow()
	}

	syscall.Close(fds[0])
	syscall.Close(fds[1])
	syscall.Close(fds1[0])
	syscall.Close(fds1[1])
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
		ev := New(base, receivers[i], EvRead, func(fd int, events uint32, arg interface{}) {}, nil)
		err = ev.Attach(0)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()

	if err := base.Shutdown(); err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		syscall.Close(receivers[i])
	}
}

func BenchmarkEventDel(b *testing.B) {
	receivers := make([]int, b.N)
	events := make([]*Event, b.N)
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

		ev := New(base, receivers[i], EvRead, func(fd int, events uint32, arg interface{}) {}, nil)
		err = ev.Attach(0)
		if err != nil {
			b.Fatal(err)
		}
		events[i] = ev
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = events[i].Detach()
		if err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()

	if err := base.Shutdown(); err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		syscall.Close(receivers[i])
	}
}

func BenchmarkEventLoop(b *testing.B) {
	receivers := make([]int, b.N)
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
		receivers[i] = fds[0]
		senders[i] = fds[1]

		ev := New(base, fds[0], EvRead|EvPersist, func(fd int, events uint32, arg interface{}) {
			syscall.Read(fd, buf)
			fires++
		}, nil)
		err = ev.Attach(0)
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
		syscall.Close(receivers[i])
		syscall.Close(senders[i])
	}

	if err := base.Shutdown(); err != nil {
		b.Fatal(err)
	}
}
