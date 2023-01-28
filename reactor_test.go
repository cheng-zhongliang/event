package unicorn

import (
	"syscall"
	"testing"
)

func TestNewReactor(t *testing.T) {
	r, err := NewEventReactor(10, Epoll)
	if err != nil {
		t.Error(err)
	}
	if err := r.CoolOff(); err != nil {
		t.Error(err)
	}
}

func TestReactorRegisterEvent(t *testing.T) {
	r, err := NewEventReactor(10, Epoll)
	if err != nil {
		t.Error(err)
	}
	r0, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, 0, 0, 0)
	if errno != 0 {
		t.Error(errno)
	}
	ev := Event{
		Fd:   int(r0),
		Flag: EventRead,
	}
	if err := r.RegisterEvent(ev, func(ev Event) {}); err != nil {
		t.Error(err)
	}
	if err := r.CoolOff(); err != nil {
		t.Error(err)
	}
	if err := syscall.Close(int(r0)); err != nil {
		t.Error(err)
	}
}

func TestReactorUnregisterEvent(t *testing.T) {
	r, err := NewEventReactor(10, Epoll)
	if err != nil {
		t.Error(err)
	}
	r0, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, 0, 0, 0)
	if errno != 0 {
		t.Error(errno)
	}
	ev := Event{
		Fd:   int(r0),
		Flag: EventRead,
	}
	if err := r.RegisterEvent(ev, func(ev Event) {}); err != nil {
		t.Error(err)
	}
	if err := r.UnregisterEvent(ev); err != nil {
		t.Error(err)
	}
	if err := r.CoolOff(); err != nil {
		t.Error(err)
	}
	if err := syscall.Close(int(r0)); err != nil {
		t.Error(err)
	}
}

func TestReact(t *testing.T) {
	r, err := NewEventReactor(10, Epoll)
	if err != nil {
		t.Error(err)
	}
	r0, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, 0, 0, 0)
	if errno != 0 {
		t.Error(errno)
	}
	ev := Event{
		Fd:   int(r0),
		Flag: EventRead,
	}
	if err := r.RegisterEvent(ev, func(ev Event) {
		if ev.Flag != EventRead {
			t.Error("ev.Flag != EventRead")
		}
		if ev.Fd != int(r0) {
			t.Errorf("ev.Fd != %d", r0)
		}
		n, err := syscall.Read(int(r0), make([]byte, 8))
		if err != nil {
			t.Error(err)
		}
		if n != 8 {
			t.Error("read: n != 8")
		}
		if err := r.CoolOff(); err != nil {
			t.Error(err)
		}
	}); err != nil {
		t.Error(err)
	}
	n, err := syscall.Write(int(r0), []byte{0, 0, 0, 0, 0, 0, 0, 1})
	if err != nil {
		t.Error(err)
	}
	if n != 8 {
		t.Error("write: n != 8")
	}
	if err := r.React(); err != nil && err.Error() != "bad file descriptor" {
		t.Error(err)
	}
	if err := syscall.Close(int(r0)); err != nil {
		t.Error(err)
	}
}
