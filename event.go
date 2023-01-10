package unicorn

type EventFlag uint32

const (
	Readable EventFlag = 0x01
	Writable EventFlag = 0x02
)

type Event struct {
	fd       int
	flag     EventFlag
	callback EventCallback
}

func NewEvent(fd int, flag EventFlag, fn EventCallback) (ev Event) {
	return Event{fd, flag, fn}
}
