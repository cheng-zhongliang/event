// Copyright (c) 2023 cheng-zhongliang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"errors"
	"syscall"
)

var (
	ErrEventExists    = errors.New("event exists")
	ErrEventNotExists = errors.New("event does not exist")
	ErrEventInvalid   = errors.New("event invalid")
)

func temporaryErr(err error) bool {
	errno, ok := err.(syscall.Errno)
	if !ok {
		return false
	}
	return errno.Temporary()
}
