package unicorn

import "sync"

type EventReactorConfig struct {
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
	case EPOLL:
		demultiplexer, err = NewEpoll()
	case KQUEUE:
		fallthrough
	default:
		err = ErrInvalidDemultiplexerType
	}
	if err != nil {
		return nil, err
	}

	evReactor = &EventReactor{
		C:             c,
		Demultiplexer: demultiplexer,
		Hanlder:       make(EventHandler, 1024),
	}

	return
}

func (r *EventReactor) RegisterEvent(ev Event) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.Hanlder[ev.fd]; ok {
		return ErrEventExists
	}
	r.Hanlder[ev.fd] = ev.callback

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

func (r *EventReactor) ModifyEvent(ev Event) error {
	r.RLock()
	defer r.RUnlock()

	if _, ok := r.Hanlder[ev.fd]; !ok {
		return ErrEventNotExists
	}
	r.Hanlder[ev.fd] = ev.callback

	return r.Demultiplexer.ModEvent(ev)
}

func (r *EventReactor) React() error {
	activeEvents := make([]Event, 1024)
	callbacks := make([]EventCallback, 1024)
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
