package event

import (
	"syscall"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
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

	r1, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, 0, 0, 0)
	if errno != 0 {
		t.Fatal(errno)
	}

	r2, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, 0, 0, 0)
	if errno != 0 {
		t.Fatal(errno)
	}

	ev1 := New(int(r1), EvRead|EvTimeout, func(fd int, events uint32, arg interface{}) {
		if fd != int(r1) {
			t.Fatal("fd not equal")
		}
		if events != EvTimeout {
			t.Fatal("events not equal")
		}
		if arg != "hello" {
			t.Fatal("arg not equal")
		}
	}, "hello")

	ev2 := New(int(r2), EvRead|EvTimeout, func(fd int, events uint32, arg interface{}) {
		if fd != int(r2) {
			t.Fatal("fd not equal")
		}
		if events != EvTimeout {
			t.Fatal("events not equal")
		}
		if arg != "world" {
			t.Fatal("arg not equal")
		}
		err = base.Exit()
		if err != nil {
			t.Fatal(err)
		}
	}, "world")

	err = base.AddEvent(ev1, 1500*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	err = base.AddEvent(ev2, 2000*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	err = base.Dispatch()
	if err != nil && err != ErrBadFileDescriptor {
		t.Fatal(err)
	}

	syscall.Close(int(r1))
	syscall.Close(int(r2))
}

func TestTimer(t *testing.T) {
	base, err := NewBase()
	if err != nil {
		t.Fatal(err)
	}

	n := 0
	ev1 := New(-1, EvTimeout, func(fd int, events uint32, arg interface{}) {
		n++
		if events != EvTimeout {
			t.Fatal("events not equal")
		}
		if arg != "hello" {
			t.Fatal("arg not equal")
		}
	}, "hello")

	nn := 0
	ev2 := New(-2, EvTimeout|EvPersist, func(fd int, events uint32, arg interface{}) {
		nn++
		if events != EvTimeout {
			t.Fatal("events not equal")
		}
		if arg != "world" {
			t.Fatal("arg not equal")
		}
		if nn == 3 {
			err = base.Exit()
			if err != nil {
				t.Fatal(err)
			}
		}
	}, "world")

	err = base.AddEvent(ev1, 1000*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	err = base.AddEvent(ev2, 1000*time.Millisecond)
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
