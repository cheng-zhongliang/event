package event

// eventListEle is an element of a linked list.
type eventListEle struct {
	next, prev *eventListEle
	list       *eventList
	Value      *Event
}

// next returns the next list element or nil.
func (e *eventListEle) nextEle() *eventListEle {
	if p := e.next; e.list != nil && p != &e.list.root {
		return p
	}
	return nil
}

// eventList represents a doubly linked list.
type eventList struct {
	root eventListEle
	len  int
}

// newEventList creates a new list.
func newEventList() *eventList {
	l := new(eventList)
	l.root.next = &l.root
	l.root.prev = &l.root
	l.len = 0
	return l
}

// front returns the first element of list l or nil.
func (l *eventList) front() *eventListEle {
	if l.len == 0 {
		return nil
	}
	return l.root.next
}

// pushBack inserts a new element e with value v at the back of list l and returns e.
func (l *eventList) pushBack(ev *Event) *eventListEle {
	e := &eventListEle{list: l, Value: ev}
	n := &l.root
	p := n.prev
	e.next = n
	e.prev = p
	p.next = e
	n.prev = e
	l.len++
	return e
}

// remove removes e from its list, decrements l.len, and returns e.
func (l *eventList) remove(e *eventListEle) *Event {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil
	e.prev = nil
	e.list = nil
	l.len--
	return e.Value
}
