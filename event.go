package unicorn

type EventType uint32

const (
	EventTimeout EventType = iota
	EventRead
	EventWrite
	EventPersist
	EventET
	EventLT
)

type EventCallbackFn func(fd int, what EventType, arg interface{})

type Event struct {
	fd   int
	what EventType
	cb   EventCallbackFn
	arg  interface{}
}

func NewEvent(fd int, what EventType, cb EventCallbackFn, arg interface{}) *Event {
	return &Event{fd, what, cb, arg}
}
