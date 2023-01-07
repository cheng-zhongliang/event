package unicorn

type EventDemultiplexer interface {
	AddEvent(ev Event) error
	DelEvent(ev Event) error
	WaitEvents() ([]Event, error)
}
