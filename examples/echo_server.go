package main

import (
	"event"
	"syscall"
	"time"
)

func main() {
	base, err := event.NewBase()
	if err != nil {
		panic(err)
	}

	fd := Socket()
	ev := event.New(fd, event.EvRead, Accept, base)
	if err := base.AddEvent(ev, 0); err != nil {
		panic(err)
	}

	exitEv := event.New(-1, event.EvTimeout, Exit, base)
	if err := base.AddEvent(exitEv, 1*time.Minute); err != nil {
		panic(err)
	}

	if err := base.Dispatch(); err != nil && err != event.ErrBadFileDescriptor {
		panic(err)
	}
}

func Socket() int {
	addr := syscall.SockaddrInet4{Port: 1246, Addr: [4]byte{0, 0, 0, 0}}
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM|syscall.SOCK_NONBLOCK, syscall.IPPROTO_TCP)
	if err != nil {
		panic(err)
	}
	if err := syscall.Bind(fd, &addr); err != nil {
		panic(err)
	}
	err = syscall.Listen(fd, syscall.SOMAXCONN)
	if err != nil {
		panic(err)
	}
	return fd
}

func Accept(fd int, events uint32, arg interface{}) {
	base := arg.(*event.EventBase)

	clientFd, _, err := syscall.Accept(fd)
	if err != nil {
		panic(err)
	}

	ev := event.New(clientFd, event.EvRead|event.EvPersist, Echo, nil)
	if err := base.AddEvent(ev, 0); err != nil {
		panic(err)
	}
}

func Echo(fd int, events uint32, arg interface{}) {
	buf := make([]byte, 0xFFF)
	n, err := syscall.Read(fd, buf)
	if err != nil {
		panic(err)
	}
	if _, err := syscall.Write(fd, buf[:n]); err != nil {
		panic(err)
	}
}

func Exit(fd int, events uint32, arg interface{}) {
	base := arg.(*event.EventBase)

	if err := base.Exit(); err != nil {
		panic(err)
	}
}
