package unicorn

import "sync"

type EventReactor struct {
	Demultiplexer EventDemultiplexer
	Hanlder       EventHandler
	sync.Mutex
}

func NewEventReactor() *EventReactor {
	return &EventReactor{}
}

func (r *EventReactor) RegisterEvent(ev Event, handler HandleFunc) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.Hanlder[ev]; ok {
		return ErrEventExists
	}
	r.Hanlder[ev] = handler

	return r.Demultiplexer.AddEvent(ev)
}

func (r *EventReactor) UnregisterEvent(ev Event) error {
	r.Lock()
	defer r.Unlock()

	delete(r.Hanlder, ev)

	return r.Demultiplexer.DelEvent(ev)
}

func (r *EventReactor) React() error {
	evs, err := r.Demultiplexer.WaitEvents()
	if err != nil {
		return err
	}

	for _, ev := range evs {
		if handleFunc, ok := r.Hanlder[ev]; ok {
			handleFunc(ev)
		}
	}

	return nil
}
