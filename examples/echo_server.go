package main

import (
	"syscall"

	"github.com/cheng-zhongliang/event"
)

func main() {
	base, err := event.NewBase()
	if err != nil {
		panic(err)
	}
	fd := socket()
	ev := event.New(base, fd, event.EvRead|event.EvPersist, accept, base)
	if err := ev.Attach(0); err != nil {
		panic(err)
	}
	if err := base.Dispatch(); err != nil && err != syscall.EBADF {
		panic(err)
	}
	syscall.Close(fd)
}

func socket() int {
	addr := syscall.SockaddrInet4{Port: 1246, Addr: [4]byte{0, 0, 0, 0}}
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_TCP)
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

func accept(fd int, events uint32, arg interface{}) {
	base := arg.(*event.EventBase)
	clientFd, _, err := syscall.Accept(fd)
	if err != nil {
		panic(err)
	}
	ev := event.New(base, clientFd, event.EvRead|event.EvPersist, echo, nil)
	if err := ev.Attach(0); err != nil {
		panic(err)
	}
}

func echo(fd int, events uint32, arg interface{}) {
	buf := make([]byte, 0xFFF)
	n, err := syscall.Read(fd, buf)
	if err != nil {
		panic(err)
	}
	if _, err := syscall.Write(fd, buf[:n]); err != nil {
		panic(err)
	}
}
