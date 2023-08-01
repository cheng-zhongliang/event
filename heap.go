package event

// eventHeap is the min heap of timeout events.
type eventHeap []*Event

// newEventHeap creates a new min event heap.
func newEventHeap() *eventHeap {
	eh := &eventHeap{}
	return eh.init()
}

// less returns true if the event at index i is less than the event at index j.
func (eh eventHeap) less(i, j int) bool {
	return eh[i].deadline < eh[j].deadline
}

// swap swaps the events at index i and j.
func (eh eventHeap) swap(i, j int) {
	eh[i], eh[j] = eh[j], eh[i]
	eh[i].index = i
	eh[j].index = j
}

// len returns the length of the event heap.
func (eh eventHeap) len() int {
	return len(eh)
}

// up moves the event at index j up the heap.
func (eh eventHeap) up(j int) {
	for {
		i := (j - 1) / 2
		if i == j || !eh.less(j, i) {
			break
		}
		eh.swap(i, j)
		j = i
	}
}

// down moves the event at index i down the heap.
func (eh eventHeap) down(i0, n int) bool {
	i := i0
	for {
		j1 := 2*i + 1
		if j1 >= n {
			break
		}
		j := j1
		if j2 := j1 + 1; j2 < n && eh.less(j2, j1) {
			j = j2
		}
		if !eh.less(j, i) {
			break
		}
		eh.swap(i, j)
		i = j
	}
	return i > i0
}

// pushEvent pushes an event onto the heap.
func (eh *eventHeap) pushEvent(ev *Event) int {
	*eh = append(*eh, ev)
	ev.index = eh.len() - 1
	eh.up(ev.index)
	return ev.index
}

// removeEvent removes an event at index from the heap.
func (eh *eventHeap) removeEvent(index int) *Event {
	n := eh.len() - 1
	if n != index {
		eh.swap(index, n)
		if !eh.down(index, n) {
			eh.up(index)
		}
	}
	ev := (*eh)[n]
	*eh = (*eh)[:n]
	return ev
}

// peekEvent returns the event at the top of the heap.
func (eh *eventHeap) peekEvent() *Event {
	return (*eh)[0]
}

// empty returns true if the heap is empty.
func (eh *eventHeap) empty() bool {
	return eh.len() == 0
}

// init initializes the heap.
func (eh *eventHeap) init() *eventHeap {
	n := eh.len()
	for i := n/2 - 1; i >= 0; i-- {
		eh.down(i, n)
	}
	return eh
}
