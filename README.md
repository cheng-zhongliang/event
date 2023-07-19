<p align="center">
<img 
    src="logo.png" 
    width="350" height="175" border="0" alt="event">
<br><br>
<a href="https://godoc.org/github.com/cheng-zhongliang/event"><img src="https://img.shields.io/badge/go-reference-blue" alt="GoDoc"></a>
<a href="https://github.com/cheng-zhongliang/event/blob/master/LICENSE"><img src="https://img.shields.io/badge/license-BSD--3--Clause-brightgreen"></a>
</p>

`event` is a network I/O event notification library for Go. It uses epoll to poll I/O events that is fast and low memory usage. It works in a similar manner as [libevent](https://github.com/libevent/libevent).

The goal of `event` is to provide a `BASIC` tool for building high performance network applications.

*Note: All development is done on a Raspberry Pi 4B.*

## Features

- Simple API
- Low memory usage
- Support Read/Write/Timeout events
- Efficient ticker

## Getting Started

### Installing
To start using `event`, just run `go get`:

```sh
$ go get -u github.com/cheng-zhongliang/event
```

### Events

Supports Following events:

- `EvRead` fires when the fd is readable.
- `EvWrite` fires when the fd is writable.
- `EvTimeout` fires when the timeout expires.
- `EvPersist` fires repeatedly until the event is deleted.

These events can be used in combination such as EvRead|EvTimeout. When the event fires, the callback function will be called. 

### Ticker

This is a simple example of how to use the ticker:

```go
ev := event.New(-1, event.EvTimeout|event.EvPersist, Tick, nil)
base.AddEvent(ev, 1*time.Second)
```

The event will be triggered every second.

### Usage

Example echo server that binds to port 1246:

```go
package main

import (
	"syscall"
	"time"

	"github.com/cheng-zhongliang/event"
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
```

Connect to the echo server:

```sh
$ telnet localhost 1246
```
