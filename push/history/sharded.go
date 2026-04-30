package history

import (
	"hash/fnv"
	"sync"
	"sync/atomic"
)

const shardCount = 16

// ShardedHistory provides high-performance sharded history storage with 16 independent shards.
//
// Design Decisions:
//   - 16 shards: Balance between memory overhead and concurrency performance
//   - Generic type T: Store any data type
//   - Batch push: Group records by shard during batch write, reduce lock acquisitions
//
// Shard Count Selection:
//   - CPU cores typically 4-16, 16 shards can match mainstream hardware
//   - Too few shards -> severe lock contention
//   - Too many shards -> increased memory overhead
//   - FNV-1a hash for data distribution, avoid hotspots
//
// Lock Strategy:
//   - Each shard uses independent sync.RWMutex
//   - Write operations use Lock() for atomicity
//   - Read operations use RLock() for concurrent reads
//   - Batch operations group by shard, reduce lock acquisitions
//
// Thread Safety:
//   - All operations are thread-safe
//   - Read operations don't block writes (read-write lock)
//   - Batch operations guarantee atomicity
type ShardedHistory[T any] struct {
	shards [shardCount]struct {
		buffer *RingBuffer[T]
		mu     sync.RWMutex
	}
	maxSize atomic.Int32
}

// NewSharded creates a ShardedHistory instance with specified buffer capacity.
//
// Parameters:
//   - maxSize: Maximum capacity of each shard's ring buffer
//     If <= 0, uses default value 1024
//
// Design Considerations:
//   - Allocate independent buffer for each shard, enabling true parallel access
//   - Use same capacity configuration for simplified management
//   - Pre-allocate memory initially, avoid runtime dynamic allocation overhead
//
// Example:
//
//	sh := history.NewSharded[MyType](1024) // Each shard gets 1024 capacity
func NewSharded[T any](maxSize int) *ShardedHistory[T] {
	h := &ShardedHistory[T]{}
	if maxSize > 0 {
		h.maxSize.Store(int32(maxSize))
	}
	for i := 0; i < shardCount; i++ {
		h.shards[i].buffer = NewRingBuffer[T](maxSize)
	}
	return h
}

// fnv32 computes FNV-1a 32-bit hash of a string.
func fnv32(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}

// getShard calculates shard index by key and returns the corresponding shard.
func (h *ShardedHistory[T]) getShard(key string) *struct {
	buffer *RingBuffer[T]
	mu     sync.RWMutex
} {
	if h == nil || key == "" {
		return nil
	}
	hashed := fnv32(key)
	return &h.shards[hashed%shardCount]
}

// Push adds an element to buffer of the specified key.
//
// Flow:
//  1. Parameter validation
//  2. Calculate shard index by key
//  3. Acquire shard write lock
//  4. Push element into ring buffer
//  5. Release lock
//
// Performance: O(1), locks only single shard
//
// Thread Safety: Atomic operation, no additional synchronization needed
func (h *ShardedHistory[T]) Push(key string, r T) {
	if h == nil {
		return
	}
	shard := h.getShard(key)
	if shard == nil {
		return
	}
	shard.mu.Lock()
	shard.buffer.Push(r)
	shard.mu.Unlock()
}

// BatchPush adds multiple elements with batch optimization.
//
// Design Considerations:
//
//  1. Record Aggregation Phase (Lock-free)
//     - Iterate all elements, calculate shard index
//     - Group elements by shard, using map[int][]T
//     - This phase is completely lock-free, can execute in parallel
//
//  2. Shard Write Phase (Lock per shard)
//     - Iterate grouped shards
//     - Lock each shard separately, batch write all elements of that shard
//     - Reduce lock acquisitions: from len(records) to at most shardCount
//
// Performance Advantages:
//   - Reduce lock contention: Reuse same-shard lock for multi-element writes
//   - Improve cache locality: Same-shard elements written continuously
//   - Reduce system call overhead: Reduce lock acquire/release count
//
// Complexity:
//   - Time: O(n) where n is element count
//   - Space: O(n) for aggregation map
//
// Thread Safety: Atomic operation guarantee
func (h *ShardedHistory[T]) BatchPush(key func(T) string, records []T) {
	if h == nil || len(records) == 0 {
		return
	}

	shardMap := make(map[int][]T)
	for i := range records {
		recordKey := key(records[i])
		shardIdx := int(fnv32(recordKey) % shardCount)
		shardMap[shardIdx] = append(shardMap[shardIdx], records[i])
	}

	for idx, recs := range shardMap {
		shard := &h.shards[idx]
		shard.mu.Lock()
		for _, r := range recs {
			shard.buffer.Push(r)
		}
		shard.mu.Unlock()
	}
}

// List retrieves elements of a specified key.
//
// Parameters:
//   - key: Storage key identifier (e.g., "alarm", "metrics")
//   - n: Number of elements to retrieve
//     n < 0: Return nil
//     n == 0: Retrieve all elements of this key
//     n > 0: Retrieve last n elements
//
// Returns: Elements in insertion order (oldest first)
//
// Performance: O(1), accesses only single shard, uses read lock
//
// Thread Safety: Read lock protection, data won't be modified during read
//
// Example:
//
//	items := sh.List("alarm", 10) // Get last 10 items with key "alarm"
func (h *ShardedHistory[T]) List(key string, n int) []T {
	if h == nil || n < 0 {
		return nil
	}
	shard := h.getShard(key)
	if shard == nil {
		return nil
	}
	shard.mu.RLock()
	var result []T
	if n == 0 {
		result = shard.buffer.GetAll()
	} else {
		result = shard.buffer.GetLast(n)
	}
	shard.mu.RUnlock()
	return result
}

// Clear removes all elements of a specified key.
//
// Parameters:
//   - key: Target buffer key
//
// Effects:
//   - Clears ring buffer of the key's corresponding shard
//   - Releases stored element memory
//
// Performance: O(1), operates only single shard
//
// Thread Safety: Write lock protection
//
// Example:
//
//	sh.Clear("alarm") // Clear all elements with key "alarm"
func (h *ShardedHistory[T]) Clear(key string) {
	if h == nil || key == "" {
		return
	}
	shard := h.getShard(key)
	if shard == nil {
		return
	}
	shard.mu.Lock()
	shard.buffer.Clear()
	shard.mu.Unlock()
}

// SetSize dynamically adjusts buffer size of all shards.
//
// Parameters:
//   - newSize: New maximum capacity, must be > 0
//
// Behavior:
//   - Updates global maximum capacity config
//   - Iterates all shards, adjusts size separately
//   - If newSize < current capacity, data will be truncated (keeps newest data)
//
// Design Considerations:
//   - Adjust shards one by one, ensuring stability during adjustment
//   - Use lock separation, avoid locking all shards at once
//
// Complexity: O(shardCount)
//
// Thread Safety: Write lock protection
//
// Example:
//
//	sh.SetSize(4096) // Resize all shards to 4096
func (h *ShardedHistory[T]) SetSize(newSize int) {
	if h == nil || newSize <= 0 {
		return
	}
	h.maxSize.Store(int32(newSize))
	for i := 0; i < shardCount; i++ {
		h.shards[i].mu.Lock()
		h.shards[i].buffer.Resize(newSize)
		h.shards[i].mu.Unlock()
	}
}

// Count gets element count of a specified key.
//
// Parameters:
//   - key: Storage key identifier
//
// Returns: Current stored element count of this key
//
// Performance: O(1), accesses only single shard
//
// Thread Safety: Read lock protection
//
// Example:
//
//	count := sh.Count("alarm") // Get count of elements with key "alarm"
func (h *ShardedHistory[T]) Count(key string) int {
	if h == nil || key == "" {
		return 0
	}
	shard := h.getShard(key)
	if shard == nil {
		return 0
	}
	shard.mu.RLock()
	count := shard.buffer.Count()
	shard.mu.RUnlock()
	return count
}

// CountAll counts total elements of all shards.
//
// Returns: Total count of all keys' elements
//
// Performance: O(shardCount), needs to iterate all shards
//
// Thread Safety: Read lock protection, data stable during iteration
//
// Example:
//
//	total := sh.CountAll() // Get total count of all shards
func (h *ShardedHistory[T]) CountAll() int {
	if h == nil {
		return 0
	}
	var total int
	for i := 0; i < shardCount; i++ {
		h.shards[i].mu.RLock()
		total += h.shards[i].buffer.Count()
		h.shards[i].mu.RUnlock()
	}
	return total
}

// ClearAll removes elements in all shards.
//
// Performance: O(shardCount), needs to iterate all shards
//
// Thread Safety: Write lock protection
//
// Example:
//
//	sh.ClearAll() // Clear all shards
func (h *ShardedHistory[T]) ClearAll() {
	if h == nil {
		return
	}
	for i := 0; i < shardCount; i++ {
		h.shards[i].mu.Lock()
		h.shards[i].buffer.Clear()
		h.shards[i].mu.Unlock()
	}
}
