// Package pinwrappers implements interfaces that wrap the basic board interface and return types, and expands them with new
// methods and interfaces for the built in board models. Current expands analog reader and digital interrupt.
package pinwrappers

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"go.viam.com/rdk/components/board"
)

// A ReconfigurableDigitalInterrupt is a simple reconfigurable digital interrupt that expects
// reconfiguration within the same type.
type ReconfigurableDigitalInterrupt interface {
	board.DigitalInterrupt
	Reconfigure(cfg board.DigitalInterruptConfig) error
}

// CreateDigitalInterrupt is a factory method for creating a specific DigitalInterrupt based
// on the given config. If no type is specified, a BasicDigitalInterrupt is returned.
func CreateDigitalInterrupt(cfg board.DigitalInterruptConfig) (ReconfigurableDigitalInterrupt, error) {
	i := &BasicDigitalInterrupt{}

	if err := i.Reconfigure(cfg); err != nil {
		return nil, err
	}
	return i, nil
}

// A BasicDigitalInterrupt records how many ticks/interrupts happen and can
// report when they happen to interested callbacks.
type BasicDigitalInterrupt struct {
	count int64

	callbacks []chan board.Tick

	mu  sync.RWMutex
	cfg board.DigitalInterruptConfig
}

// Value returns the amount of ticks that have occurred.
func (i *BasicDigitalInterrupt) Value(ctx context.Context, extra map[string]interface{}) (int64, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	count := atomic.LoadInt64(&i.count)
	return count, nil
}

// Ticks is really just for testing.
func (i *BasicDigitalInterrupt) Ticks(ctx context.Context, num int, now uint64) error {
	for x := 0; x < num; x++ {
		if err := i.Tick(ctx, true, now+uint64(x)); err != nil {
			return err
		}
	}
	return nil
}

// Tick records an interrupt and notifies any interested callbacks. See comment on
// the DigitalInterrupt interface for caveats.
func (i *BasicDigitalInterrupt) Tick(ctx context.Context, high bool, nanoseconds uint64) error {
	if high {
		atomic.AddInt64(&i.count, 1)
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
	for _, c := range i.callbacks {
		select {
		case <-ctx.Done():
			return errors.New("context cancelled")
		case c <- board.Tick{Name: i.cfg.Name, High: high, TimestampNanosec: nanoseconds}:
		}
	}
	return nil
}

// AddCallback adds a listener for interrupts.
func (i *BasicDigitalInterrupt) AddCallback(c chan board.Tick) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.callbacks = append(i.callbacks, c)
}

// RemoveCallback removes a listener for interrupts.
func (i *BasicDigitalInterrupt) RemoveCallback(c chan board.Tick) {
	i.mu.Lock()
	defer i.mu.Unlock()
	for id := range i.callbacks {
		if i.callbacks[id] == c {
			// To remove this item, we replace it with the last item in the list, then truncate the
			// list by 1.
			i.callbacks[id] = i.callbacks[len(i.callbacks)-1]
			i.callbacks = i.callbacks[:len(i.callbacks)-1]
			break
		}
	}
}

func (i *BasicDigitalInterrupt) Name() string {
	return i.cfg.Name
}

// Close does nothing.
func (i *BasicDigitalInterrupt) Close(ctx context.Context) error {
	return nil
}

// Reconfigure reconfigures this digital interrupt with a new formula.
func (i *BasicDigitalInterrupt) Reconfigure(conf board.DigitalInterruptConfig) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.cfg = conf
	return nil
}
