package event

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// SignalPoller is the signal poller.
type SignalPoller struct {
	Feedback func(Signal int)
	SignalCh chan os.Signal
	Signals  []os.Signal
	ExitCh   chan struct{}
	Wg       *sync.WaitGroup
}

// NewSignalPoller creates a new signal poller.
func NewSignalPoller(cb func(signal int)) *SignalPoller {
	sp := &SignalPoller{
		Feedback: cb,
		SignalCh: make(chan os.Signal, 1),
		Signals:  make([]os.Signal, 0),
		ExitCh:   make(chan struct{}),
		Wg:       &sync.WaitGroup{},
	}

	sp.Wg.Add(1)
	go sp.PollSignal()

	return sp
}

// PollSignal polls the signal.
func (sp *SignalPoller) PollSignal() {
	defer sp.Wg.Done()
	for {
		select {
		case sig := <-sp.SignalCh:
			sp.Feedback(int(sig.(syscall.Signal)))
		case <-sp.ExitCh:
			return
		}
	}
}

// Close stops the signal poller.
func (sp *SignalPoller) Close() {
	close(sp.ExitCh)
	sp.Wg.Wait()
}

// SubscribeSignal subscribes the signal.
// Thread-safe.
func (sp *SignalPoller) SubscribeSignal(sig int) {
	for _, s := range sp.Signals {
		if s == syscall.Signal(sig) {
			return
		}
	}
	sp.Signals = append(sp.Signals, syscall.Signal(sig))

	signal.Notify(sp.SignalCh, sp.Signals...)
}

// UnsubscribeSignal unsubscribes the signal.
// Thread-safe.
func (sp *SignalPoller) UnsubscribeSignal(sig int) {
	for i, s := range sp.Signals {
		if s == syscall.Signal(sig) {
			sp.Signals = append(sp.Signals[:i], sp.Signals[i+1:]...)
			break
		}
	}

	signal.Stop(sp.SignalCh)
	signal.Notify(sp.SignalCh, sp.Signals...)
}
