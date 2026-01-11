package cache

import (
	"container/list"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// MemoryCache implements Cache using an in-memory LRU cache.
type MemoryCache struct {
	mu         sync.RWMutex
	items      map[string]*list.Element
	lru        *list.List
	config     Config
	currentMem int64
	hits       int64
	misses     int64
	stopCh     chan struct{}
	stopped    bool
}

type memoryEntry struct {
	key       string
	value     []byte
	expiresAt time.Time
	size      int64
}

// NewMemoryCache creates a new in-memory cache.
func NewMemoryCache(cfg Config) *MemoryCache {
	if cfg.DefaultTTL == 0 {
		cfg.DefaultTTL = 5 * time.Minute
	}

	c := &MemoryCache{
		items:  make(map[string]*list.Element),
		lru:    list.New(),
		config: cfg,
		stopCh: make(chan struct{}),
	}

	// Start cleanup goroutine
	go c.cleanupLoop()

	return c
}

func (c *MemoryCache) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCh:
			return
		}
	}
}

func (c *MemoryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	var toDelete []string

	for key, elem := range c.items {
		entry := elem.Value.(*memoryEntry)
		if now.After(entry.expiresAt) {
			toDelete = append(toDelete, key)
		}
	}

	for _, key := range toDelete {
		c.deleteInternal(key)
	}
}

// Cleanup triggers an immediate cleanup of expired items.
// Exposed for testing purposes.
func (c *MemoryCache) Cleanup() {
	c.cleanup()
}

func (c *MemoryCache) deleteInternal(key string) {
	if elem, ok := c.items[key]; ok {
		entry := elem.Value.(*memoryEntry)
		c.currentMem -= entry.size
		c.lru.Remove(elem)
		delete(c.items, key)
	}
}

func (c *MemoryCache) evict(needed int64) {
	// Evict oldest entries until we have enough space
	for c.lru.Len() > 0 && c.config.MaxMemory > 0 && c.currentMem+needed > c.config.MaxMemory {
		oldest := c.lru.Back()
		if oldest == nil {
			break
		}
		entry := oldest.Value.(*memoryEntry)
		c.deleteInternal(entry.key)
	}

	// Also check MaxItems
	for c.lru.Len() > 0 && c.config.MaxItems > 0 && c.lru.Len() >= c.config.MaxItems {
		oldest := c.lru.Back()
		if oldest == nil {
			break
		}
		entry := oldest.Value.(*memoryEntry)
		c.deleteInternal(entry.key)
	}
}

// Get retrieves a value from the cache.
func (c *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		atomic.AddInt64(&c.misses, 1)
		return nil, ErrCacheMiss
	}

	entry := elem.Value.(*memoryEntry)
	if time.Now().After(entry.expiresAt) {
		c.deleteInternal(key)
		atomic.AddInt64(&c.misses, 1)
		return nil, ErrCacheMiss
	}

	// Move to front (most recently used)
	c.lru.MoveToFront(elem)
	atomic.AddInt64(&c.hits, 1)

	// Return a copy to avoid data races
	result := make([]byte, len(entry.value))
	copy(result, entry.value)
	return result, nil
}

// Set stores a value in the cache with the given TTL.
func (c *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl == 0 {
		ttl = c.config.DefaultTTL
	}

	// Make a copy of the value
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	size := int64(len(valueCopy))

	c.mu.Lock()
	defer c.mu.Unlock()

	// If key exists, remove old entry first
	if elem, ok := c.items[key]; ok {
		oldEntry := elem.Value.(*memoryEntry)
		c.currentMem -= oldEntry.size
		c.lru.Remove(elem)
		delete(c.items, key)
	}

	// Evict if necessary
	c.evict(size)

	entry := &memoryEntry{
		key:       key,
		value:     valueCopy,
		expiresAt: time.Now().Add(ttl),
		size:      size,
	}

	elem := c.lru.PushFront(entry)
	c.items[key] = elem
	c.currentMem += size

	return nil
}

// Delete removes a key from the cache.
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.deleteInternal(key)
	return nil
}

// Exists checks if a key exists in the cache.
func (c *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	elem, ok := c.items[key]
	if !ok {
		return false, nil
	}

	entry := elem.Value.(*memoryEntry)
	return time.Now().Before(entry.expiresAt), nil
}

// GetJSON retrieves and unmarshals a JSON value from the cache.
func (c *MemoryCache) GetJSON(ctx context.Context, key string, dest any) error {
	data, err := c.Get(ctx, key)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("json unmarshal: %w", err)
	}
	return nil
}

// SetJSON marshals and stores a value as JSON in the cache.
func (c *MemoryCache) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	return c.Set(ctx, key, data, ttl)
}

// GetOrSet implements the cache-aside pattern.
func (c *MemoryCache) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (any, error)) (any, error) {
	var cached any
	err := c.GetJSON(ctx, key, &cached)
	if err == nil {
		return cached, nil
	}
	if err != ErrCacheMiss {
		// Unexpected error, still try to fetch fresh data
	}

	value, err := fn()
	if err != nil {
		return nil, err
	}

	// Store in cache (ignore errors)
	_ = c.SetJSON(ctx, key, value, ttl)

	return value, nil
}

// MGet retrieves multiple values from the cache.
func (c *MemoryCache) MGet(ctx context.Context, keys ...string) ([][]byte, error) {
	results := make([][]byte, len(keys))
	for i, key := range keys {
		data, err := c.Get(ctx, key)
		if err == nil {
			results[i] = data
		}
		// On miss, results[i] remains nil
	}
	return results, nil
}

// MSet stores multiple values in the cache.
func (c *MemoryCache) MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
	for key, value := range items {
		if err := c.Set(ctx, key, value, ttl); err != nil {
			return err
		}
	}
	return nil
}

// DeletePattern deletes all keys matching the glob pattern.
func (c *MemoryCache) DeletePattern(ctx context.Context, pattern string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var toDelete []string
	for key := range c.items {
		matched, err := filepath.Match(pattern, key)
		if err != nil {
			return fmt.Errorf("invalid pattern: %w", err)
		}
		if matched {
			toDelete = append(toDelete, key)
		}
	}

	for _, key := range toDelete {
		c.deleteInternal(key)
	}

	return nil
}

// Keys returns all keys matching the glob pattern.
func (c *MemoryCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var keys []string
	now := time.Now()

	for key, elem := range c.items {
		entry := elem.Value.(*memoryEntry)
		if now.After(entry.expiresAt) {
			continue
		}

		matched, err := filepath.Match(pattern, key)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern: %w", err)
		}
		if matched {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

// Incr increments a key's value.
func (c *MemoryCache) Incr(ctx context.Context, key string) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var currentVal int64

	if elem, ok := c.items[key]; ok {
		entry := elem.Value.(*memoryEntry)
		if time.Now().Before(entry.expiresAt) {
			if err := json.Unmarshal(entry.value, &currentVal); err != nil {
				return 0, fmt.Errorf("value is not a number: %w", err)
			}
		}
	}

	currentVal++
	data, _ := json.Marshal(currentVal)

	// Use default TTL for new counter
	ttl := c.config.DefaultTTL

	entry := &memoryEntry{
		key:       key,
		value:     data,
		expiresAt: time.Now().Add(ttl),
		size:      int64(len(data)),
	}

	if elem, ok := c.items[key]; ok {
		oldEntry := elem.Value.(*memoryEntry)
		c.currentMem -= oldEntry.size
		c.lru.Remove(elem)
	}

	elem := c.lru.PushFront(entry)
	c.items[key] = elem
	c.currentMem += entry.size

	return currentVal, nil
}

// Decr decrements a key's value.
func (c *MemoryCache) Decr(ctx context.Context, key string) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var currentVal int64

	if elem, ok := c.items[key]; ok {
		entry := elem.Value.(*memoryEntry)
		if time.Now().Before(entry.expiresAt) {
			if err := json.Unmarshal(entry.value, &currentVal); err != nil {
				return 0, fmt.Errorf("value is not a number: %w", err)
			}
		}
	}

	currentVal--
	data, _ := json.Marshal(currentVal)

	ttl := c.config.DefaultTTL

	entry := &memoryEntry{
		key:       key,
		value:     data,
		expiresAt: time.Now().Add(ttl),
		size:      int64(len(data)),
	}

	if elem, ok := c.items[key]; ok {
		oldEntry := elem.Value.(*memoryEntry)
		c.currentMem -= oldEntry.size
		c.lru.Remove(elem)
	}

	elem := c.lru.PushFront(entry)
	c.items[key] = elem
	c.currentMem += entry.size

	return currentVal, nil
}

// Close stops the cleanup goroutine and clears the cache.
func (c *MemoryCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.stopped {
		close(c.stopCh)
		c.stopped = true
	}

	c.items = make(map[string]*list.Element)
	c.lru = list.New()
	c.currentMem = 0

	return nil
}

// Health always returns nil for memory cache.
func (c *MemoryCache) Health(ctx context.Context) error {
	return nil
}

// Stats returns cache statistics.
func (c *MemoryCache) Stats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return Stats{
		Hits:       atomic.LoadInt64(&c.hits),
		Misses:     atomic.LoadInt64(&c.misses),
		Keys:       int64(len(c.items)),
		MemoryUsed: c.currentMem,
	}
}

// Clear removes all items from the cache.
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.lru = list.New()
	c.currentMem = 0
}
