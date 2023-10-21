// Copyright (c) 2023 cheng-zhongliang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

type element struct {
	next, prev *element
	list       *list
	value      interface{}
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

func (l *list) pushBack(v interface{}, e *element) *element {
	e.list = l
	e.value = v
	n := &l.root
	p := n.prev
	e.next = n
	e.prev = p
	p.next = e
	n.prev = e
	l.len++
	return e
}

func (l *list) remove(e *element) {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil
	e.prev = nil
	e.list = nil
	l.len--
}
