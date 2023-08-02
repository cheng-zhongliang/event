package event

type eventListEle struct {
	next, prev *eventListEle
	list       *eventList
	value      *Event
}

func (e *eventListEle) nextEle() *eventListEle {
	if p := e.next; e.list != nil && p != &e.list.root {
		return p
	}
	return nil
}

type eventList struct {
	root eventListEle
	len  int
}

func newEventList() *eventList {
	l := new(eventList)
	l.root.next = &l.root
	l.root.prev = &l.root
	l.len = 0
	return l
}

func (l *eventList) front() *eventListEle {
	if l.len == 0 {
		return nil
	}
	return l.root.next
}

func (l *eventList) pushBack(ev *Event) *eventListEle {
	e := &eventListEle{list: l, value: ev}
	n := &l.root
	p := n.prev
	e.next = n
	e.prev = p
	p.next = e
	n.prev = e
	l.len++
	return e
}

func (l *eventList) remove(e *eventListEle) *Event {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil
	e.prev = nil
	e.list = nil
	l.len--
	return e.value
}
