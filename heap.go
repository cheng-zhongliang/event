/*
 * Copyright (c) 2023 cheng-zhongliang. All rights reserved.
 *
 * This source code is licensed under the 3-Clause BSD License, which can be
 * found in the accompanying "LICENSE" file, or at:
 * https://opensource.org/licenses/BSD-3-Clause
 *
 * By accessing, using, copying, modifying, or distributing this software,
 * you agree to the terms and conditions of the BSD 3-Clause License.
 */

package event

type eventHeap []*Event

func newEventHeap() *eventHeap {
	eh := &eventHeap{}
	return eh.init()
}

func (eh eventHeap) less(i, j int) bool {
	return eh[i].deadline < eh[j].deadline
}

func (eh eventHeap) swap(i, j int) {
	eh[i], eh[j] = eh[j], eh[i]
	eh[i].index = i
	eh[j].index = j
}

func (eh eventHeap) len() int {
	return len(eh)
}

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

func (eh *eventHeap) pushEvent(ev *Event) int {
	*eh = append(*eh, ev)
	ev.index = eh.len() - 1
	eh.up(ev.index)
	return ev.index
}

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

func (eh *eventHeap) peekEvent() *Event {
	return (*eh)[0]
}

func (eh *eventHeap) empty() bool {
	return eh.len() == 0
}

func (eh *eventHeap) init() *eventHeap {
	n := eh.len()
	for i := n/2 - 1; i >= 0; i-- {
		eh.down(i, n)
	}
	return eh
}
