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

	err = base.Exit()
	if err != nil {
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

	err = base.Exit()
	if err != nil {
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

	err = base.Exit()
	if err != nil {
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
		err = base.Exit()
		if err != nil {
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
		err = base.Exit()
		if err != nil {
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
		err = base.Exit()
		if err != nil {
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
			err = base.Exit()
			if err != nil {
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
		err = base.Exit()
		if err != nil {
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
	ev1.SetPriority(High)

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
			err = base.Exit()
			if err != nil {
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
