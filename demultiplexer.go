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
	WaitActiveEvents(activeEvents []Event) (n int, err error)
	NormalizeEvent(interface{}) Event      // convert from platform specific event to Event
	UnNormalizeEvent(ev Event) interface{} // convert from Event to platform specific event
}
