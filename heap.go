// Copyright (c) 2023 cheng-zhongliang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

type eventHeap []*Event

func (eh eventHeap) less(i, j int) bool {
	return eh[i].deadline.Before(eh[j].deadline)
}

func (eh eventHeap) swap(i, j int) {
	eh[i], eh[j] = eh[j], eh[i]
	eh[i].index = i
	eh[j].index = j
}

func (eh eventHeap) up(j int) {
	for {
		i := (j - 1) / 4
		if i == j || !eh.less(j, i) {
			break
		}
		eh.swap(i, j)
		j = i
	}
}

func (eh eventHeap) down(i0, n int) bool {
	i := i0
	for {
		j1 := 4*i + 1
		if j1 >= n {
			break
		}
		j := j1
		if j2 := j1 + 1; j2 < n && eh.less(j2, j1) {
			j = j2
		}
		if j3 := j1 + 2; j3 < n && eh.less(j3, j) {
			j = j3
		}
		if j4 := j1 + 3; j4 < n && eh.less(j4, j) {
			j = j4
		}
		if !eh.less(j, i) {
			break
		}
		eh.swap(i, j)
		i = j
	}
	return i > i0
}

func (eh *eventHeap) pushEvent(ev *Event) int {
	*eh = append(*eh, ev)
	ev.index = len(*eh) - 1
	eh.up(ev.index)
	return ev.index
}

func (eh *eventHeap) removeEvent(index int) {
	n := len(*eh) - 1
	if n != index {
		eh.swap(index, n)
		if !eh.down(index, n) {
			eh.up(index)
		}
	}
	*eh = (*eh)[:n]
}

func (eh *eventHeap) peekEvent() *Event {
	return (*eh)[0]
}

func (eh *eventHeap) empty() bool {
	return len(*eh) == 0
}
