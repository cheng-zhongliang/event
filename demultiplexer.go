package unicorn

type Demultiplexer interface {
	Add(ev Event) error
	Del(ev Event) error
	Wait(evs []Event) (n int, err error)
}
