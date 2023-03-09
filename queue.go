package unicorn

type EventQueue struct {
	ch chan Event
}

func NewEventQueue(size int) *EventQueue {
	return &EventQueue{
		ch: make(chan Event, size),
	}
}

func (eq *EventQueue) Push(ev Event) error {
	select {
	case eq.ch <- ev:
		return nil
	default:
		return ErrQueueFull
	}
}

func (eq *EventQueue) Pop() (Event, error) {
	select {
	case ev := <-eq.ch:
		return ev, nil
	default:
		return Event{}, ErrQueueEmpty
	}
}
