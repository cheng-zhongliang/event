package unicorn

type EventCallbackFn func(fd int, flags EventFlags, arg interface{})

type EventFlags uint32

const (
	EventRead  EventFlags = 0x02
	EventWrite EventFlags = 0x04
)

type Event struct {
	fd    int
	flags EventFlags
	cb    EventCallbackFn
	arg   interface{}
}

func NewEvent(fd int, flags EventFlags, cb EventCallbackFn, arg interface{}) Event {
	return Event{fd, flags, cb, arg}
}
