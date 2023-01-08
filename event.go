package unicorn

type EventFlag uint32

const (
	Readable EventFlag = 0x01
	Writable EventFlag = 0x02
)

type Event struct {
	fd     int
	flags  uint32
	handleFn HandleFunc
}

func NewEvent(fd int, flags []EventFlag, fn HandleFunc) (ev Event) {
	ev.fd = fd
	ev.handleFn = fn
	for _, flag := range flags {
		ev.flags |= uint32(flag)
	}
	return
}
