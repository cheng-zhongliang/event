package unicorn

type EventDemultiplexer interface {
	AddEvent(ev Event) error
	DelEvent(ev Event) error
	ModEvent(ev Event) error
	WaitEvents([]Event) (n int, err error)
}
