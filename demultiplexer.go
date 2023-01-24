package unicorn

type EventDemultiplexerType int

const (
	Epoll EventDemultiplexerType = iota
	Kqueue
)

type EventDemultiplexer interface {
	AddEvent(ev Event) error
	DelEvent(ev Event) error
	ModEvent(ev Event) error
	WaitActiveEvents(evs []Event) (n int, err error)
	Close() error
}
