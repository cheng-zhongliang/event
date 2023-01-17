<p align="center">
<img 
    src="logo.png" 
    width="213" height="75" border="0" alt="unicorn">
<br>
<a href="https://godoc.org/github.com/cheng-zhongliang/unicorn"><img src="https://img.shields.io/badge/go-reference-blue" alt="GoDoc"></a>
<a href="https://github.com/cheng-zhongliang/unicorn/blob/master/LICENSE"><img src="https://img.shields.io/badge/license-BSD--3--Clause-brightgreen" alt="GoDoc"></a>
</p>

`unicorn` is a network I/O event notification library for Go.

`unicorn` adopts reactor architecture.

`unicorn` supports epoll on Linux ~~and kqueue on Unix~~.

*Note: Unicorn is more suitable for beginners who want to learn I/O multiplexing.*

*Note: All development is done on a Raspberry Pi 4B.*

## Features

- Simple API
- Standard reactor architecture
- Low memory usage
- Cross-platform
