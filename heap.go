package event

// EventHeap is the min heap of timeout events.
type EventHeap []*Event

// NewEventHeap creates a new min event heap.
func NewEventHeap() *EventHeap {
	eh := &EventHeap{}
	return eh.Init()
}

// Less returns true if the event at index i is less than the event at index j.
func (eh EventHeap) Less(i, j int) bool {
	return eh[i].Deadline < eh[j].Deadline
}

// Swap swaps the events at index i and j.
func (eh EventHeap) Swap(i, j int) {
	eh[i], eh[j] = eh[j], eh[i]
	eh[i].Index = i
	eh[j].Index = j
}

// Len returns the length of the event heap.
func (eh EventHeap) Len() int {
	return len(eh)
}

// Up moves the event at index j up the heap.
func (eh EventHeap) Up(j int) {
	for {
		i := (j - 1) / 2
		if i == j || !eh.Less(j, i) {
			break
		}
		eh.Swap(i, j)
		j = i
	}
}

// Down moves the event at index i down the heap.
func (eh EventHeap) Down(i0, n int) bool {
	i := i0
	for {
		j1 := 2*i + 1
		if j1 >= n {
			break
		}
		j := j1
		if j2 := j1 + 1; j2 < n && eh.Less(j2, j1) {
			j = j2
		}
		if !eh.Less(j, i) {
			break
		}
		eh.Swap(i, j)
		i = j
	}
	return i > i0
}

// PushEvent pushes an event onto the heap.
func (eh *EventHeap) PushEvent(ev *Event) int {
	*eh = append(*eh, ev)
	ev.Index = eh.Len() - 1
	eh.Up(ev.Index)
	return ev.Index
}

// PopEvent pops an event off the heap.
func (eh *EventHeap) PopEvent() *Event {
	n := eh.Len() - 1
	eh.Swap(0, n)
	eh.Down(0, n)
	ev := (*eh)[n]
	*eh = (*eh)[:n]
	return ev
}

// RemoveEvent removes an event at index from the heap.
func (eh *EventHeap) RemoveEvent(index int) *Event {
	n := eh.Len() - 1
	if n != index {
		eh.Swap(index, n)
		if !eh.Down(index, n) {
			eh.Up(index)
		}
	}
	ev := (*eh)[n]
	*eh = (*eh)[:n]
	return ev
}

// PeekEvent returns the event at the top of the heap.
func (eh *EventHeap) PeekEvent() *Event {
	return (*eh)[0]
}

// Empty returns true if the heap is empty.
func (eh *EventHeap) Empty() bool {
	return eh.Len() == 0
}

// Init initializes the heap.
func (eh *EventHeap) Init() *EventHeap {
	n := eh.Len()
	for i := n/2 - 1; i >= 0; i-- {
		eh.Down(i, n)
	}
	return eh
}
