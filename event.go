package unicorn

type EventFlag uint32

const (
	Readable EventFlag = 0x02
	Writable EventFlag = 0x04
)

type Event struct {
	fd    int
	flags EventFlag
}

func NewEvent(fd int, flags EventFlag) Event {
	return Event{fd, flags}
}
