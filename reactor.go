package unicorn

import "sync"

type EventReactorConfig struct {
	Capacity          int
	DemultiplexerType EventDemultiplexerType
}

type EventReactor struct {
	C             EventReactorConfig
	Demultiplexer EventDemultiplexer
	Hanlder       EventHandler
	sync.RWMutex
}

func NewEventReactor(c EventReactorConfig) (evReactor *EventReactor, err error) {
	var demultiplexer EventDemultiplexer
	switch c.DemultiplexerType {
	case Epoll:
		demultiplexer, err = NewEpoller()
	case Kqueue:
		fallthrough
	default:
		err = ErrInvalidDemultiplexerType
	}
	if err != nil {
		return
	}

	evReactor = &EventReactor{
		C:             c,
		Demultiplexer: demultiplexer,
		Hanlder:       make(EventHandler, c.Capacity),
	}

	return
}

func (r *EventReactor) RegisterEvent(ev Event, fn EventCallback) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.Hanlder[ev.fd]; ok {
		return ErrEventExists
	}
	r.Hanlder[ev.fd] = fn

	return r.Demultiplexer.AddEvent(ev)
}

func (r *EventReactor) UnregisterEvent(ev Event) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.Hanlder[ev.fd]; !ok {
		return ErrEventNotExists
	}
	delete(r.Hanlder, ev.fd)

	return r.Demultiplexer.DelEvent(ev)
}

func (r *EventReactor) ModifyEvent(ev Event, fn EventCallback) error {
	r.RLock()
	defer r.RUnlock()

	if _, ok := r.Hanlder[ev.fd]; !ok {
		return ErrEventNotExists
	}
	r.Hanlder[ev.fd] = fn

	return r.Demultiplexer.ModEvent(ev)
}

func (r *EventReactor) React() error {
	activeEvents := make([]Event, r.C.Capacity)
	callbacks := make([]EventCallback, r.C.Capacity)
	for {
		n, err := r.Demultiplexer.WaitActiveEvents(activeEvents)
		if err != nil {
			return err
		}

		r.RLock()
		for i := 0; i < n; i++ {
			callbacks[i] = r.Hanlder[activeEvents[i].fd]
		}
		r.RUnlock()

		for i := 0; i < n; i++ {
			callbacks[i](activeEvents[i])
		}
	}
}
