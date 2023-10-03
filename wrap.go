// Copyright (c) 2023 cheng-zhongliang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

// NewTimer creates a new timer event.
func NewTimer(base *EventBase, callback func(fd int, events uint32, arg interface{}), arg interface{}) *Event {
	return New(base, -1, EvTimeout, callback, arg)
}

// NewTicker creates a new ticker event.
func NewTicker(base *EventBase, callback func(fd int, events uint32, arg interface{}), arg interface{}) *Event {
	return New(base, -1, EvTimeout|EvPersist, callback, arg)
}
