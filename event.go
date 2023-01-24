package unicorn

type EventFlag uint32

const (
	EventRead  EventFlag = 0x01
	EventWrite EventFlag = 0x02
)

type Event struct {
	Fd   int
	Flag EventFlag
}
