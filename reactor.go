package unicorn

type EventReactorConfig struct {
	Capacity          int
	DemultiplexerType EventDemultiplexerType
}

type EventReactor struct {
	Demultiplexer EventDemultiplexer
	Hanlder       EventHandler
	ActiveEvs     []Event
}

func NewEventReactor(capacity int, demultiplexerType EventDemultiplexerType) (*EventReactor, error) {
	demultiplexer, err := NewEventDemultiplexer(demultiplexerType, capacity)
	if err != nil {
		return nil, err
	}
	return &EventReactor{
		Demultiplexer: demultiplexer,
		Hanlder:       make(EventHandler),
		ActiveEvs:     make([]Event, capacity),
	}, nil
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
	for {
		n, err := r.Demultiplexer.WaitActiveEvents(r.ActiveEvs)
		if err != nil {
			return err
		}
		for i := 0; i < n; i++ {
			r.Hanlder[r.ActiveEvs[i].Fd](r.ActiveEvs[i])
		}
	}
}

func (r *EventReactor) CoolOff() error {
	return r.Demultiplexer.Close()
}
