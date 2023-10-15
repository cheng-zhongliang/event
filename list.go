// Copyright (c) 2023 cheng-zhongliang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

type element[T any] struct {
	next, prev *element[T]
	list       *list[T]
	value      T
}

func (e *element[T]) nextEle() *element[T] {
	if p := e.next; e.list != nil && p != &e.list.root {
		return p
	}
	return nil
}

type list[T any] struct {
	root element[T]
	len  int
}

func newList[T any]() *list[T] {
	l := new(list[T])
	l.root.next = &l.root
	l.root.prev = &l.root
	l.len = 0
	return l
}

func (l *list[T]) front() *element[T] {
	if l.len == 0 {
		return nil
	}
	return l.root.next
}

func (l *list[T]) pushBack(v T, e *element[T]) *element[T] {
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

func (l *list[T]) remove(e *element[T]) {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil
	e.prev = nil
	e.list = nil
	l.len--
}
