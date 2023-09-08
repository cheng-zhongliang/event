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

// NewTimer creates a new timer event.
func NewTimer(base *EventBase, callback func(fd int, events uint32, arg interface{}), arg interface{}) *Event {
	return New(base, -1, EvTimeout, callback, arg)
}

// NewTicker creates a new ticker event.
func NewTicker(base *EventBase, callback func(fd int, events uint32, arg interface{}), arg interface{}) *Event {
	return New(base, -1, EvTimeout|EvPersist, callback, arg)
}
