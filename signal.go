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
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type signaler struct {
	feedback func(signal int)
	signalCh chan os.Signal
	signals  []os.Signal
	exitCh   chan struct{}
	wg       *sync.WaitGroup
}

func newSignaler(cb func(signal int)) *signaler {
	sp := &signaler{
		feedback: cb,
		signalCh: make(chan os.Signal, 1),
		signals:  make([]os.Signal, 0),
		exitCh:   make(chan struct{}),
		wg:       &sync.WaitGroup{},
	}

	sp.wg.Add(1)
	go sp.poll()

	return sp
}

func (sp *signaler) poll() {
	defer sp.wg.Done()
	for {
		select {
		case sig := <-sp.signalCh:
			sp.feedback(int(sig.(syscall.Signal)))
		case <-sp.exitCh:
			return
		}
	}
}

func (sp *signaler) close() {
	close(sp.exitCh)
	sp.wg.Wait()
}

func (sp *signaler) add(sig int) error {
	for _, s := range sp.signals {
		if s == syscall.Signal(sig) {
			return ErrEventExists
		}
	}
	sp.signals = append(sp.signals, syscall.Signal(sig))
	signal.Notify(sp.signalCh, sp.signals...)
	return nil
}

func (sp *signaler) del(sig int) error {
	notExists := true
	for i, s := range sp.signals {
		if s == syscall.Signal(sig) {
			notExists = false
			sp.signals = append(sp.signals[:i], sp.signals[i+1:]...)
			break
		}
	}
	if notExists {
		return ErrEventNotExists
	}
	signal.Stop(sp.signalCh)
	signal.Notify(sp.signalCh, sp.signals...)
	return nil
}
