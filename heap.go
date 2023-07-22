package event

import "container/heap"

// EventHeap is the min heap of timeout events.
type EventHeap []*Event

// NewEventHeap creates a new min event heap.
func NewEventHeap() *EventHeap {
	eh := &EventHeap{}
	heap.Init(eh)
	return eh
}

// Len returns the length of the event heap.
func (h EventHeap) Len() int { return len(h) }

// Less returns true if the event at index i is less than the event at index j.
func (h EventHeap) Less(i, j int) bool {
	return h[i].Deadline < h[j].Deadline
}

// Swap swaps the events at index i and j.
func (h EventHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].Index = i
	h[j].Index = j
}

// Push pushes the event x onto the heap.
func (h *EventHeap) Push(x interface{}) {
	ev := x.(*Event)
	*h = append(*h, x.(*Event))
	ev.Index = len(*h) - 1
}

// Pop removes and returns the event at the root of the heap.
func (h *EventHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// PopEvent removes and returns the event at the root of the heap.
func (h *EventHeap) PopEvent() { heap.Pop(h) }

// PushEvent pushes the event x onto the heap.
func (h *EventHeap) PushEvent(ev *Event) { heap.Push(h, ev) }

// PeekEvent returns the event at the root of the heap.
func (h *EventHeap) PeekEvent() *Event { return (*h)[0] }

// Empty returns true if the heap is empty.
func (h *EventHeap) Empty() bool { return len(*h) == 0 }

// RemoveEvent remove the event at index.
func (h *EventHeap) RemoveEvent(index int) {
	heap.Remove(h, index)
}
