package unicorn

import "sync"

type EventReactor struct {
	Demultiplexer EventDemultiplexer
	Hanlder       EventHandler
	sync.RWMutex
}

func NewEventReactor() *EventReactor {
	return &EventReactor{
		Hanlder: make(EventHandler, 0xFFFF),
	}
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
	evs := make([]Event, 0xFFFF)
	for {
		n, err := r.Demultiplexer.WaitEvents(evs)
		if err != nil {
			return err
		}

		r.RLock()
		for i := 0; i < n; i++ {
			if handleFunc, ok := r.Hanlder[evs[i].fd]; ok {
				handleFunc(evs[i])
			}
		}
		r.RUnlock()
	}
}
