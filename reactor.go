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

func (r *EventReactor) React() error {
	evs, err := r.Demultiplexer.WaitEvents()
	if err != nil {
		return err
	}

	for _, ev := range evs {
		if handleFunc, ok := r.Hanlder[ev.fd]; ok {
			handleFunc(ev)
		}
	}

	return nil
}
