package unicorn

type EventDemultiplexerType int

const (
	Epoll EventDemultiplexerType = iota + 1
	Kqueue
)

type EventDemultiplexer interface {
	AddEvent(ev Event) error
	DelEvent(ev Event) error
	ModEvent(ev Event) error
	WaitActiveEvents(evs []Event) (n int, err error)
	Close() error
}

func NewEventDemultiplexer(t EventDemultiplexerType, capacity int) (demultiplexer EventDemultiplexer, err error) {
	switch t {
	case Epoll:
		demultiplexer, err = NewEpoller(capacity)
	case Kqueue:
		fallthrough
	default:
		err = ErrInvalidDemultiplexerType
	}
	return
}
