package unicorn

import (
	"syscall"
	"testing"
)

func TestNewEpoller(t *testing.T) {
	ep, err := NewEpoller(10)
	if err != nil {
		t.Error(err)
	}
	if err := ep.Close(); err != nil {
		t.Error(err)
	}
}

func TestEpollerAddEvent(t *testing.T) {
	r0, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, 0, 0, 0)
	if errno != 0 {
		t.Error(errno)
	}

	ep, err := NewEpoller(10)
	if err != nil {
		t.Error(err)
	}
	ev := Event{
		Fd:   int(r0),
		Flag: EventRead,
	}
	if err := ep.AddEvent(ev); err != nil {
		t.Error(err)
	}

	if err := ep.Close(); err != nil {
		t.Error(err)
	}
	if err := syscall.Close(int(r0)); err != nil {
		t.Error(err)
	}
}

func TestEpollerDelEvent(t *testing.T) {
	r0, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, 0, 0, 0)
	if errno != 0 {
		t.Error(errno)
	}

	ep, err := NewEpoller(10)
	if err != nil {
		t.Error(err)
	}
	ev := Event{
		Fd:   int(r0),
		Flag: EventRead,
	}
	if err := ep.AddEvent(ev); err != nil {
		t.Error(err)
	}
	if err := ep.DelEvent(ev); err != nil {
		t.Error(err)
	}

	if err := ep.Close(); err != nil {
		t.Error(err)
	}
	if err := syscall.Close(int(r0)); err != nil {
		t.Error(err)
	}
}

func TestEpollerModEvent(t *testing.T) {
	r0, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, 0, 0, 0)
	if errno != 0 {
		t.Error(errno)
	}

	ep, err := NewEpoller(10)
	if err != nil {
		t.Error(err)
	}
	ev := Event{
		Fd:   int(r0),
		Flag: EventRead,
	}
	if err := ep.AddEvent(ev); err != nil {
		t.Error(err)
	}
	ev.Flag = EventWrite
	if err := ep.ModEvent(ev); err != nil {
		t.Error(err)
	}

	if err := ep.Close(); err != nil {
		t.Error(err)
	}
	if err := syscall.Close(int(r0)); err != nil {
		t.Error(err)
	}
}

func TestEpollerWaitActiveEvents(t *testing.T) {
	r0, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, 0, 0, 0)
	if errno != 0 {
		t.Error(errno)
	}

	ep, err := NewEpoller(10)
	if err != nil {
		t.Error(err)
	}
	ev := Event{
		Fd:   int(r0),
		Flag: EventRead,
	}
	if err := ep.AddEvent(ev); err != nil {
		t.Error(err)
	}

	go func() {
		n, err := syscall.Write(int(r0), []byte{0, 0, 0, 0, 0, 0, 0, 1})
		if err != nil {
			t.Error(err)
		}
		if n != 8 {
			t.Error("n != 8")
		}
	}()

	evs := make([]Event, 10)
	n, err := ep.WaitActiveEvents(evs)
	if err != nil {
		t.Error(err)
	}
	if n != 1 {
		t.Error("n != 1")
	}
	if evs[0].Fd != int(r0) {
		t.Error("evs[0].Fd != int(r0)")
	}
	if evs[0].Flag != EventRead {
		t.Error("evs[0].Flag != EventRead")
	}

	if err := ep.Close(); err != nil {
		t.Error(err)
	}
	if err := syscall.Close(int(r0)); err != nil {
		t.Error(err)
	}
}
