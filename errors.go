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

import (
	"errors"
	"syscall"
)

var (
	ErrBadFileDescriptor = syscall.EBADF
	ErrEventExists       = errors.New("event exists")
	ErrEventNotExists    = errors.New("event does not exist")
	ErrEventInvalid      = errors.New("event invalid")
)

func temporaryErr(err error) bool {
	errno, ok := err.(syscall.Errno)
	if !ok {
		return false
	}
	return errno.Temporary()
}
