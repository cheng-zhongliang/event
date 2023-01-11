package unicorn

type EventFlag uint32

const (
	Readable EventFlag = 0x01
	Writable EventFlag = 0x02
)

type Event struct {
	fd   int
	flag EventFlag
}

func NewEvent(fd int, flag EventFlag) (ev Event) {
	return Event{fd, flag}
}
