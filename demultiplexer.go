package unicorn

type EventDemultiplexer interface {
	Add(ev Event) error
	Del(ev Event) error
	Wait(evs []Event) (n int, err error)
}
