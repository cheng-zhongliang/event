package unicorn

import "golang.org/x/sys/unix"

type Epoll struct {
	BaseFd int
}

func NewEpoll() (*Epoll, error) {
	fd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}
	return &Epoll{
		BaseFd: fd,
	}, nil
}

func (ep *Epoll) AddEvent(ev Event) error {
	return nil
}

func (ep *Epoll) DelEvent(ev Event) error {
	return nil
}

func (ep *Epoll) ModEvent(ev Event) error {
	return nil
}

func (ep *Epoll) WaitActiveEvents() ([]Event, error) {
	return nil, nil
}
