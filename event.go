package unicorn

type EventCallbackFn func(fd int, what EventType, arg interface{})

type EventType uint32

const (
	EventRead  EventType = 0x02
	EventWrite EventType = 0x04
)

type Event struct {
	fd   int
	what EventType
	cb   EventCallbackFn
	arg  interface{}
}

func NewEvent(fd int, what EventType, cb EventCallbackFn, arg interface{}) *Event {
	return &Event{fd, what, cb, arg}
}
