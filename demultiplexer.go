package unicorn

type EventDemultiplexerType int

const (
	EPOLL EventDemultiplexerType = iota
	KQUEUE
)

type EventDemultiplexer interface {
	AddEvent(ev Event) error
	DelEvent(ev Event) error
	ModEvent(ev Event) error
	WaitActiveEvents() ([]Event, error)
}
