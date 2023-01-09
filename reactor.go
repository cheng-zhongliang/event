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

func NewEventReactor(c EventReactorConfig) (*EventReactor, error) {
	switch c.DemultiplexerType {
	case EPOLL:
	case KQUEUE:
	default:
		return nil, ErrInvalidDemultiplexerType
	}
	return &EventReactor{
		C:       c,
		Hanlder: make(EventHandler),
	}, nil
}

func (r *EventReactor) RegisterEvent(ev Event) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.Hanlder[ev.fd]; ok {
		return ErrEventExists
	}
	r.Hanlder[ev.fd] = ev.handleFn

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
	r.Hanlder[ev.fd] = ev.handleFn

	return r.Demultiplexer.ModEvent(ev)
}

func (r *EventReactor) React() error {
	for {
		readyEvs, err := r.Demultiplexer.WaitActiveEvents()
		if err != nil {
			return err
		}

		r.RLock()
		for _, ev := range readyEvs {
			ev.handleFn = r.Hanlder[ev.fd]
		}
		r.RUnlock()

		for _, ev := range readyEvs {
			ev.handleFn(ev)
		}
	}
}
