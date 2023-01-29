<p align="center">
<img 
    src="logo.png" 
    width="213" height="75" border="0" alt="unicorn">
<br>
<a href="https://godoc.org/github.com/cheng-zhongliang/unicorn"><img src="https://img.shields.io/badge/go-reference-blue" alt="GoDoc"></a>
<a href="https://github.com/cheng-zhongliang/unicorn/blob/master/LICENSE"><img src="https://img.shields.io/badge/license-BSD--3--Clause-brightgreen"></a>
</p>

`unicorn` is a network I/O event notification library for Go. It uses epoll ~and kqueue~ to monitor I/O events. So it is very efficient and low memory usage. What's more, the API is simple and easy to use.

*Note: All development is done on a Raspberry Pi 4B.*

## Features

- Simple API
- High performance
- Low memory usage
- Lock-free
- ~Cross-platform~

## Getting Started

### Installing
To start using `unicorn`, just run `go get`:

```sh
$ go get -u github.com/cheng-zhongliang/unicorn
```

### Usage

It's easy to use `unicorn` to monitor I/O events. Just need to create an event reactor. Register for events you are interested in. Then, call the `React` method to start the event monitoring loop. When the event fires, the callback function will be called.

```go
package main

import (
	"log"
	"syscall"

	"github.com/cheng-zhongliang/unicorn"
)

func main() {
	EvReactor, err := unicorn.NewEventReactor(10, unicorn.Epoll)
	if err != nil {
		log.Fatal(err)
	}
	defer EvReactor.CoolOff()

	r0, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, 0, 0, 0)
	if errno != 0 {
		log.Println(err)
		return
	}
	defer syscall.Close(int(r0))

	event := unicorn.Event{
		Fd:   int(r0),
		Flag: unicorn.EventRead,
	}
	
	cb := func(ev unicorn.Event) {
		log.Println("Event triggered")
	}
	
	if err := EvReactor.RegisterEvent(event, cb); err != nil {
		log.Println(err)
		return
	}

	if err := EvReactor.React(); err != nil {
		log.Println(err)
		return
	}
}
```
