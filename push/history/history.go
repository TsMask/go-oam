// Package history provides generic history storage with ring buffer implementation.
//
// History supports two implementations:
//   - History[T]: Standard implementation, lazy buffer creation
//   - ShardedHistory[T]: Sharded implementation, 16 shards for high concurrency
//
// Usage Examples:
//
//	// Standard History
//	h := history.New[MyType](1024)
//	h.Push(data, "key")
//	records := h.List("key", 10)
//
//	// ShardedHistory for high concurrency
//	sh := history.NewSharded[MyType](1024)
//	sh.Push(data, "key")
//	records := sh.List("key", 10)
package history

import (
	"sync"
	"sync/atomic"
)

// History provides generic type-based history storage with lazy buffer initialization.
//
// Design Decisions:
//   - Generic type T: Store any data type
//   - Lazy initialization: Buffer created on first access, saves memory
//   - Type isolation: Different keys use independent buffers
//
// Thread Safety:
//   - sync.Map ensures thread-safe storage
//   - Each buffer operation is thread-safe
type History[T any] struct {
	// buffers stores ring buffers for each key
	// Key: user-provided key string
	// Value: *RingBuffer[T]
	buffers sync.Map

	// maxSize is the maximum capacity for each ring buffer
	// Uses atomic.Int32 for lock-free read/write
	maxSize atomic.Int32
}

// New creates a History instance with specified buffer capacity.
//
// Parameters:
//   - maxSize: Maximum capacity of each ring buffer
//     If <= 0, uses default value 1024
//
// Returns: Initialized History instance
//
// Example:
//
//	h := history.New[MyData](2048)
func New[T any](maxSize int) *History[T] {
	h := &History[T]{}
	if maxSize > 0 {
		h.maxSize.Store(int32(maxSize))
	}
	return h
}

// getBuffer gets or creates the ring buffer for a specific key.
//
// Logic:
//  1. Try to load existing buffer from sync.Map
//  2. If not exists, create new buffer based on maxSize
//  3. Use LoadOrStore to ensure atomic creation
//
// Thread Safety: sync.Map ensures atomicity
func (h *History[T]) getBuffer(key string) *RingBuffer[T] {
	if h == nil {
		return nil
	}

	if v, ok := h.buffers.Load(key); ok {
		return v.(*RingBuffer[T])
	}

	size := int(h.maxSize.Load())
	if size <= 0 {
		size = 1024
	}
	newBuf := NewRingBuffer[T](size)
	actual, _ := h.buffers.LoadOrStore(key, newBuf)
	return actual.(*RingBuffer[T])
}

// List retrieves elements of a specific key.
//
// Parameters:
//   - key: Storage key identifier
//   - n: Number of elements to retrieve
//     n < 0: Return nil
//     n == 0: Return all elements of this key
//     n > 0: Return last n elements
//
// Returns: Elements in insertion order (oldest first)
//
// Thread Safety: Read operation, data won't be modified
//
// Example:
//
//	items := h.List("alarm", 10) // Get last 10 items with key "alarm"
func (h *History[T]) List(key string, n int) []T {
	if h == nil || n < 0 {
		return nil
	}

	buf := h.getBuffer(key)
	if buf == nil {
		return nil
	}
	if n == 0 {
		return buf.GetAll()
	}
	return buf.GetLast(n)
}

// SetSize adjusts the maximum capacity of all buffers.
//
// Parameters:
//   - newSize: New maximum capacity, must be > 0
//
// Effects:
//   - Updates maxSize config for new buffer creation
//   - Iterates all existing buffers and adjusts size
//   - If newSize < current capacity, data may be truncated (keeps newest data)
//
// Thread Safety: Data stable during Range traversal
//
// Example:
//
//	h.SetSize(4096) // Resize all buffers to 4096
func (h *History[T]) SetSize(newSize int) {
	if h == nil || newSize <= 0 {
		return
	}
	h.maxSize.Store(int32(newSize))
	h.buffers.Range(func(_, v any) bool {
		buf := v.(*RingBuffer[T])
		buf.Resize(newSize)
		return true
	})
}

// SetSizeByKey adjusts buffer size for a specific key.
//
// Parameters:
//   - key: Target buffer key
//   - newSize: New maximum capacity, must be > 0
//
// Comparison with SetSize:
//   - SetSize: Affects all buffers
//   - SetSizeByKey: Only affects specified key
//
// Example:
//
//	h.SetSizeByKey("alarm", 4096) // Only resize buffer with key "alarm"
func (h *History[T]) SetSizeByKey(key string, newSize int) {
	if h == nil || key == "" || newSize <= 0 {
		return
	}
	buf := h.getBuffer(key)
	if buf != nil {
		buf.Resize(newSize)
	}
}

// Clear removes all elements of a specific key.
//
// Parameters:
//   - key: Target buffer key
//
// Effects:
//   - Clears the ring buffer of this key
//   - Releases stored element memory
//
// Example:
//
//	h.Clear("alarm") // Clear all elements with key "alarm"
func (h *History[T]) Clear(key string) {
	if h == nil || key == "" {
		return
	}
	buf := h.getBuffer(key)
	if buf != nil {
		buf.Clear()
	}
}

// Keys returns all existing buffer keys.
//
// Returns: Array containing all created buffer keys
//
// Usage Scenarios:
//   - Debugging and monitoring
//   - Enumerating all used keys
//   - Key distribution statistics
//
// Thread Safety: Data stable during Range traversal
//
// Example:
//
//	keys := h.Keys() // Get all keys
func (h *History[T]) Keys() []string {
	if h == nil {
		return nil
	}
	keys := make([]string, 0)
	h.buffers.Range(func(k, v any) bool {
		keys = append(keys, k.(string))
		return true
	})
	return keys
}

// Push adds an element to buffer of the specified key.
//
// Parameters:
//   - r: Element to add
//   - key: Buffer key identifier
//
// Flow:
//  1. Parameter validation
//  2. Get or create buffer for this key
//  3. Push element into ring buffer
//
// Features:
//   - If new key, automatically create corresponding buffer
//   - When exceeding capacity, automatically overwrite oldest element
//
// Thread Safety: sync.Map + RingBuffer operations are all atomic
//
// Example:
//
//	h.Push(data, "metrics") // Store data with key "metrics"
func (h *History[T]) Push(key string, r T) {
	if h == nil {
		return
	}
	buf := h.getBuffer(key)
	if buf != nil {
		buf.Push(r)
	}
}
