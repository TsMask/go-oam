package internal

import (
	"sync"
	"sync/atomic"
	"time"
)

type Timer struct {
	ticker   *time.Ticker
	stopCh   chan struct{}
	stopOnce sync.Once
	callback func(time.Time)
	running  atomic.Bool
}

func NewTimer() *Timer {
	return &Timer{
		stopCh: make(chan struct{}),
	}
}

func (t *Timer) Start(interval time.Duration, callback func(time.Time)) {
	if !t.running.CompareAndSwap(false, true) {
		return
	}
	t.callback = callback
	t.stopCh = make(chan struct{})
	t.ticker = time.NewTicker(interval)
	go t.run()
}

func (t *Timer) run() {
	for {
		select {
		case now := <-t.ticker.C:
			if t.callback != nil {
				t.callback(now)
			}
		case <-t.stopCh:
			return
		}
	}
}

func (t *Timer) Stop() {
	t.stopOnce.Do(func() {
		if t.running.CompareAndSwap(true, false) {
			if t.ticker != nil {
				t.ticker.Stop()
				t.ticker = nil
			}
			close(t.stopCh)
		}
	})
}

func (t *Timer) IsRunning() bool {
	return t.running.Load()
}
