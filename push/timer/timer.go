package timer

import (
	"sync"
	"sync/atomic"
	"time"
)

// Timer provides periodic callback execution.
//
// Design Purpose:
//   - Provides reliable periodic task scheduling for push services
//   - Supports repeated callback execution at specified intervals
//   - Ensures graceful resource release on stop
//
// Lifecycle:
//   1. New() -> Create timer instance
//   2. Start() -> Begin periodic callbacks
//   3. IsRunning() -> Check running state (optional)
//   4. Stop() -> Gracefully stop timer
//
// Thread Safety:
//   - Start/Stop are safe for concurrent calls
//   - Only one callback goroutine runs at a time
//   - Uses sync.Once to ensure Stop cleanup happens once
//   - Uses atomic.Bool for lock-free state checking
//
// Usage Example:
//
//	timer := timer.New()
//	timer.Start(100*time.Millisecond, func(t time.Time) {
//	    log.Printf("Timer triggered at %v", t)
//	})
//	time.Sleep(5 * time.Second)
//	timer.Stop()
type Timer struct {
	// ticker is the underlying time ticker that sends periodic signals.
	// Created in Start() and stopped in Stop().
	ticker *time.Ticker

	// stopCh is used to signal the callback goroutine to exit.
	// Created in Start() and closed in Stop().
	stopCh chan struct{}

	// stopOnce ensures the cleanup logic in Stop() executes only once.
	// Prevents double-close panic if Stop is called multiple times.
	stopOnce sync.Once

	// mu protects shared resources during Start/Stop transitions.
	// Guards ticker and stopCh assignment.
	mu sync.Mutex

	// running indicates whether the timer is currently active.
	// Uses atomic.Bool for lock-free read/write.
	running atomic.Bool
}

// New creates a new Timer instance.
//
// The timer starts in stopped state. Call Start() to begin periodic callbacks.
//
// Returns:
//   - *Timer: New timer instance in stopped state
func New() *Timer {
	return &Timer{}
}

// Start begins periodic callback execution at the specified interval.
//
// Preconditions:
//   - Timer must be stopped
//   - interval must be > 0
//   - callback must not be nil
//
// Postconditions:
//   - Timer enters running state
//   - Callback will be invoked at the specified interval
//   - IsRunning() returns true
//
// Thread Safety:
//   - Uses CompareAndSwap for atomic state transition
//   - If timer is already running, returns immediately without action
//
// Parameters:
//   - interval: Time between callbacks, must be > 0
//   - callback: Function to call on each tick, receives current time
func (t *Timer) Start(interval time.Duration, callback func(time.Time)) {
	if !t.running.CompareAndSwap(false, true) {
		return
	}

	t.mu.Lock()
	t.stopCh = make(chan struct{})
	t.ticker = time.NewTicker(interval)
	callbackFn := callback
	t.mu.Unlock()

	go t.run(callbackFn)
}

// run is the internal loop that executes callbacks in a goroutine.
//
// It listens on two channels using select:
//   - ticker.C: Trigger callback
//   - stopCh: Exit the loop
//
// Thread Safety:
//   - Re-fetchs ticker and stopCh references each iteration
//   - Ensures correct state capture during Stop()
func (t *Timer) run(callback func(time.Time)) {
	for {
		t.mu.Lock()
		ticker := t.ticker
		stopCh := t.stopCh
		t.mu.Unlock()

		if ticker == nil || stopCh == nil {
			return
		}

		select {
		case now := <-ticker.C:
			if callback != nil {
				callback(now)
			}
		case <-stopCh:
			return
		}
	}
}

// Stop gracefully stops the timer.
//
// Preconditions:
//   - Timer must be running
//
// Postconditions:
//   - Timer enters stopped state
//   - Callback goroutine will exit after current operation
//   - All resources (ticker, channel) are released
//   - IsRunning() returns false
//
// Thread Safety:
//   - Uses CompareAndSwap to ensure only one goroutine cleans up
//   - Uses sync.Once to prevent double cleanup
//   - Safe to call even if timer is already stopped
//
// Note:
//   - Does not wait for current callback to finish
//   - Only sends stop signal, callback goroutine exits naturally
func (t *Timer) Stop() {
	if !t.running.CompareAndSwap(true, false) {
		return
	}

	t.stopOnce.Do(func() {
		t.mu.Lock()
		if t.ticker != nil {
			t.ticker.Stop()
		}
		stopCh := t.stopCh
		t.mu.Unlock()

		if stopCh != nil {
			close(stopCh)
		}
	})
}

// IsRunning returns whether the timer is currently active.
//
// Returns:
//   - true: Timer is running, callbacks are being invoked
//   - false: Timer is stopped, no callbacks will be triggered
//
// Thread Safety:
//   - Uses atomic.Bool.Load() for lock-free read
//   - Read-only operation, does not modify any state
func (t *Timer) IsRunning() bool {
	return t.running.Load()
}