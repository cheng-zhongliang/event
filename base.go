package event

import (
	"container/list"
	"fmt"
)

type EventBase struct {
	//  Poller is the event poller to watch events.
	Poller *Epoll
	// EvList is the list of all events.
	EvList *list.List
	// ActiveEvList is the list of active events.
	ActiveEvList *list.List
}

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

func (base *EventBase) AddEvent(ev *Event) error {
	if ev.Events&(EvRead|EvWrite) == 0 {
		return fmt.Errorf("invalid events: %d", ev.Events)
	}

	if ev.Flags&(EvListInserted|EvListActive) != 0 {
		return fmt.Errorf("event already added")
	}

	ev.Ele = base.EvList.PushBack(ev)

	return base.Poller.Add(ev)
}

func (base *EventBase) DelEvent(ev *Event) error {
	if ev.Flags&(EvListInserted|EvListActive) == 0 {
		return fmt.Errorf("event not added")
	}

	if ev.Flags&EvListActive != 0 {
		base.ActiveEvList.Remove(ev.ActiveEle)
		ev.Flags &^= EvListActive
	}

	if ev.Flags&EvListInserted != 0 {
		base.EvList.Remove(ev.Ele)
		ev.Flags &^= EvListInserted
	}

	return base.Poller.Del(ev)
}

func (base *EventBase) OnEvent(ev *Event, res uint32) {
	if ev.Flags&EvListActive != 0 {
		ev.Res |= res
		return
	}

	ev.Res = res
	ev.Flags |= EvListActive
	ev.ActiveEle = base.ActiveEvList.PushBack(ev)
}

func (base *EventBase) Dispatch() error {
	for {
		if err := base.Poller.Polling(base.OnEvent); err != nil {
			return err
		}

		for e := base.ActiveEvList.Front(); e != nil; e = e.Next() {
			base.ActiveEvList.Remove(e)
			ev := e.Value.(*Event)
			ev.Flags &^= EvListActive
			ev.Cb(ev.Fd, ev.Res, ev.Arg)
		}
	}
}
