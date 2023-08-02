package event

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type signalPoller struct {
	feedback func(signal int)
	signalCh chan os.Signal
	signals  []os.Signal
	exitCh   chan struct{}
	wg       *sync.WaitGroup
}

func newSignalPoller(cb func(signal int)) *signalPoller {
	sp := &signalPoller{
		feedback: cb,
		signalCh: make(chan os.Signal, 1),
		signals:  make([]os.Signal, 0),
		exitCh:   make(chan struct{}),
		wg:       &sync.WaitGroup{},
	}

	sp.wg.Add(1)
	go sp.pollSignal()

	return sp
}

func (sp *signalPoller) pollSignal() {
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

func (sp *signalPoller) close() {
	close(sp.exitCh)
	sp.wg.Wait()
}

func (sp *signalPoller) subscribeSignal(sig int) {
	for _, s := range sp.signals {
		if s == syscall.Signal(sig) {
			return
		}
	}
	sp.signals = append(sp.signals, syscall.Signal(sig))

	signal.Notify(sp.signalCh, sp.signals...)
}

func (sp *signalPoller) unsubscribeSignal(sig int) {
	for i, s := range sp.signals {
		if s == syscall.Signal(sig) {
			sp.signals = append(sp.signals[:i], sp.signals[i+1:]...)
			break
		}
	}

	signal.Stop(sp.signalCh)
	signal.Notify(sp.signalCh, sp.signals...)
}
