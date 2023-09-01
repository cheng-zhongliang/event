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

type element struct {
	next, prev *element
	list       *list
	value      *Event
}

func (e *element) nextEle() *element {
	if p := e.next; e.list != nil && p != &e.list.root {
		return p
	}
	return nil
}

type list struct {
	root element
	len  int
}

func newList() *list {
	l := new(list)
	l.root.next = &l.root
	l.root.prev = &l.root
	l.len = 0
	return l
}

func (l *list) front() *element {
	if l.len == 0 {
		return nil
	}
	return l.root.next
}

func (l *list) pushBack(v *Event, e *element) {
	e.list = l
	e.value = v
	n := &l.root
	p := n.prev
	e.next = n
	e.prev = p
	p.next = e
	n.prev = e
	l.len++
}

func (l *list) remove(e *element) {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil
	e.prev = nil
	e.list = nil
	e.value = nil
	l.len--
}
