package unicorn

type EventCallbackFn func(fd int, flags EventFlag, arg interface{})

type EventFlag uint32

const (
	EventRead  EventFlag = 0x02
	EventWrite EventFlag = 0x04
)

type Event struct {
	fd    int
	flags EventFlag
	cb    EventCallbackFn
	arg   interface{}
}

func NewEvent(fd int, flags EventFlag, cb EventCallbackFn, arg interface{}) Event {
	return Event{fd, flags, cb, arg}
}
