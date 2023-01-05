package unicorn

import "sync"

type EventReactor struct {
	Events        map[int]Event
	Demultiplexer EventDemultiplexer
	mx            sync.Mutex
}

func NewEventReactor() *EventReactor {
	return &EventReactor{
		Events: make(map[int]Event),
	}
}

func (r *EventReactor) Register(ev Event) error {
	r.mx.Lock()
	defer r.mx.Unlock()

	if _, ok := r.Events[ev.fd]; ok {
		return ErrEventExists
	}
	r.Events[ev.fd] = ev

	return r.Demultiplexer.Add(ev)
}

func (r *EventReactor) Unregister(ev Event) error {
	r.mx.Lock()
	defer r.mx.Unlock()

	if _, ok := r.Events[ev.fd]; !ok {
		return ErrEventNotExists
	}
	delete(r.Events, ev.fd)

	return r.Demultiplexer.Del(ev)
}

func (r *EventReactor) Handle() error {
	activeEvs := make([]Event, len(r.Events))
	n, err := r.Demultiplexer.Wait(activeEvs)
	if err != nil {
		return err
	}

	r.mx.Lock()
	defer r.mx.Unlock()

	for i := 0; i < n; i++ {
		if ev, ok := r.Events[activeEvs[i].fd]; ok && ev.flags&activeEvs[i].flags != 0 {
			ev.cb(ev.fd, ev.flags, ev.arg)
		}
	}

	return nil
}
