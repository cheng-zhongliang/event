package unicorn

type EventReactorConfig struct {
	Capacity          int
	DemultiplexerType EventDemultiplexerType
}

type EventReactor struct {
	C             EventReactorConfig
	Demultiplexer EventDemultiplexer
	Hanlder       EventHandler
}

func NewEventReactor(c EventReactorConfig) (evReactor *EventReactor, err error) {
	var demultiplexer EventDemultiplexer
	switch c.DemultiplexerType {
	case Epoll:
		demultiplexer, err = NewEpoller(c.Capacity)
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
	if _, ok := r.Hanlder[ev.Fd]; ok {
		return ErrEventExists
	}
	r.Hanlder[ev.Fd] = fn
	return r.Demultiplexer.AddEvent(ev)
}

func (r *EventReactor) UnregisterEvent(ev Event) error {
	if _, ok := r.Hanlder[ev.Fd]; !ok {
		return ErrEventNotExists
	}

	delete(r.Hanlder, ev.Fd)
	return r.Demultiplexer.DelEvent(ev)
}

func (r *EventReactor) ModifyEvent(ev Event, fn EventCallback) error {
	if _, ok := r.Hanlder[ev.Fd]; !ok {
		return ErrEventNotExists
	}
	r.Hanlder[ev.Fd] = fn
	return r.Demultiplexer.ModEvent(ev)
}

func (r *EventReactor) React() error {
	activeEvents := make([]Event, r.C.Capacity)
	for {
		n, err := r.Demultiplexer.WaitActiveEvents(activeEvents)
		if err != nil {
			return err
		}
		for i := 0; i < n; i++ {
			r.Hanlder[activeEvents[i].Fd](activeEvents[i])
		}
	}
}

func (r *EventReactor) CoolOff() error {
	return r.Demultiplexer.Close()
}
