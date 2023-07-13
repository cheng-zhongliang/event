package event

import (
	"container/list"
)

type EventBase struct {
	// Poller is the event poller to watch events.
	Poller *Epoll
	// EvList is the list of all events.
	EvList *list.List
	// ActiveEvList is the list of active events.
	ActiveEvList *list.List
}

// NewBase creates a new event base.
func NewBase() (*EventBase, error) {
	poller, err := NewEpoll()
	if err != nil {
		return nil, err
	}

	return &EventBase{
		Poller:       poller,
		EvList:       list.New(),
		ActiveEvList: list.New(),
	}, nil
}

// AddEvent adds an event to the event base.
func (base *EventBase) AddEvent(ev *Event) error {
	if ev.Flags&(EvListInserted|EvListActive) != 0 {
		return ErrEventAlreadyAdded
	}

	ev.Ele = base.EvList.PushBack(ev)

	return base.Poller.Add(ev)
}

// DelEvent deletes an event from the event base.
func (base *EventBase) DelEvent(ev *Event) error {
	if ev.Flags&(EvListInserted|EvListActive) == 0 {
		return ErrEventAlreadyAdded
	}

	if ev.Flags&EvListActive != 0 {
		base.ActiveEvList.Remove(ev.ActiveEle)
		ev.ActiveEle = nil
		ev.Flags &^= EvListActive
	}

	if ev.Flags&EvListInserted != 0 {
		base.EvList.Remove(ev.Ele)
		ev.Ele = nil
		ev.Flags &^= EvListInserted
	}

	return base.Poller.Del(ev)
}

// OnEvent adds an event to the active event list.
func (base *EventBase) OnEvent(ev *Event, res uint32) {
	if ev.Flags&EvListActive != 0 {
		ev.Res |= res
		return
	}

	ev.Res = res
	ev.Flags |= EvListActive
	ev.ActiveEle = base.ActiveEvList.PushBack(ev)
}

// Dispatch waits for events and dispatches them.
func (base *EventBase) Dispatch() error {
	for {
		if err := base.Poller.Polling(base.OnEvent); err != nil {
			return err
		}

		for e := base.ActiveEvList.Front(); e != nil; e = e.Next() {
			base.ActiveEvList.Remove(e)
			ev := e.Value.(*Event)
			ev.Flags &^= EvListActive
			ev.ActiveEle = nil
			ev.Cb(ev.Fd, ev.Res, ev.Arg)
		}
	}
}

// Exit closes the event base.
func (base *EventBase) Exit() error {
	return base.Poller.Close()
}
