package event

import (
	"syscall"
	"testing"
)

func TestNew(t *testing.T) {
	base, err := NewEventBase()
	if err != nil {
		t.Fatal(err)
	}

	err = base.Exit()
	if err != nil {
		t.Fatal(err)
	}
}

func TestAddEvent(t *testing.T) {
	base, err := NewEventBase()
	if err != nil {
		t.Fatal(err)
	}

	r0, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, 0, 0, 0)
	if errno != 0 {
		t.Fatal(errno)
	}

	ev := New(int(r0), EvRead, func(fd int, events uint32, arg interface{}) {
		t.Log("callback called")
	}, nil)

	err = base.AddEvent(ev)
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
	base, err := NewEventBase()
	if err != nil {
		t.Fatal(err)
	}

	r0, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, 0, 0, 0)
	if errno != 0 {
		t.Fatal(errno)
	}

	ev := New(int(r0), EvRead, func(fd int, events uint32, arg interface{}) {
		t.Log("callback called")
	}, nil)

	err = base.AddEvent(ev)
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
	base, err := NewEventBase()
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

	err = base.AddEvent(ev)
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
