package event

// EventElement is an element of a linked list.
type EventElement struct {
	next, prev *EventElement
	list       *EventList
	Value      *Event
}

// Next returns the next list element or nil.
func (e *EventElement) Next() *EventElement {
	if p := e.next; e.list != nil && p != &e.list.root {
		return p
	}
	return nil
}

// Prev returns the previous list element or nil.
func (e *EventElement) Prev() *EventElement {
	if p := e.prev; e.list != nil && p != &e.list.root {
		return p
	}
	return nil
}

// EventList represents a doubly linked list.
type EventList struct {
	root EventElement
	len  int
}

// NewEventList creates a new list.
func NewEventList() *EventList {
	l := new(EventList)
	l.root.next = &l.root
	l.root.prev = &l.root
	l.len = 0
	return l
}

// Len returns the number of elements of list l.
func (l *EventList) Len() int {
	return l.len
}

// Front returns the first element of list l or nil.
func (l *EventList) Front() *EventElement {
	if l.len == 0 {
		return nil
	}
	return l.root.next
}

// Back returns the last element of list l or nil.
func (l *EventList) Back() *EventElement {
	if l.len == 0 {
		return nil
	}
	return l.root.prev
}

// PushFront inserts a new element e with value v at the front of list l and returns e.
func (l *EventList) PushBack(ev *Event) *EventElement {
	e := &EventElement{list: l, Value: ev}
	n := &l.root
	p := n.prev
	e.next = n
	e.prev = p
	p.next = e
	n.prev = e
	l.len++
	return e
}

// PushBack inserts a new element e with value v at the back of list l and returns e.
func (l *EventList) Remove(e *EventElement) *Event {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil
	e.prev = nil
	e.list = nil
	l.len--
	return e.Value
}
