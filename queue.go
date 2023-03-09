package unicorn

type EventQueue struct {
	ch chan Event
}

func NewEventQueue(size int) *EventQueue {
	return &EventQueue{
		ch: make(chan Event, size),
	}
}

func (eq *EventQueue) Enqueue(ev Event) error {
	select {
	case eq.ch <- ev:
		return nil
	default:
		return ErrQueueFull
	}
}

func (eq *EventQueue) Dequeue() (Event, error) {
	select {
	case ev := <-eq.ch:
		return ev, nil
	default:
		return Event{}, ErrQueueEmpty
	}
}
