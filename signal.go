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

func (sp *signaler) subscribe(sig int) error {
	for _, s := range sp.signals {
		if s == syscall.Signal(sig) {
			return ErrEventExists
		}
	}
	sp.signals = append(sp.signals, syscall.Signal(sig))

	signal.Notify(sp.signalCh, sp.signals...)

	return nil
}

func (sp *signaler) unsubscribe(sig int) error {
	var (
		i int
		s os.Signal
	)

	for i, s = range sp.signals {
		if s == syscall.Signal(sig) {
			sp.signals = append(sp.signals[:i], sp.signals[i+1:]...)
			break
		}

	}

	if i == len(sp.signals)-1 {
		return ErrEventNotExists
	}

	signal.Stop(sp.signalCh)
	signal.Notify(sp.signalCh, sp.signals...)

	return nil
}
