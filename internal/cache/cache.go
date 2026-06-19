package cache

import (
	"sync"
	"time"
)

// Item represents a cache item with expiration.
type Item struct {
	value     interface{}
	expiresAt time.Time
}

// Expired returns true if the item has expired.
func (i Item) Expired() bool {
	if i.expiresAt.IsZero() {
		return false
	}
	return time.Now().After(i.expiresAt)
}

// Cache is a simple thread-safe in-memory cache.
type Cache struct {
	mu       sync.RWMutex
	items    map[string]*Item
	ttl      time.Duration
	stopCh   chan struct{}
	stopped  bool
}

// New creates a new cache with the given TTL and cleanup interval.
// If cleanupInterval <= 0, no background cleanup is performed.
func New(ttl, cleanupInterval time.Duration) *Cache {
	c := &Cache{
		items:  make(map[string]*Item),
		ttl:    ttl,
		stopCh: make(chan struct{}),
	}

	if cleanupInterval > 0 {
		go c.cleanupLoop(cleanupInterval)
	}

	return c
}

// Get returns the value for the given key, or nil if not found or expired.
func (c *Cache) Get(key string) interface{} {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()

	if !ok {
		return nil
	}

	if item.Expired() {
		c.mu.Lock()
		// Double-check after acquiring write lock
		if item, ok := c.items[key]; ok && item.Expired() {
			delete(c.items, key)
		}
		c.mu.Unlock()
		return nil
	}

	return item.value
}

// Set stores a value with the default TTL.
func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &Item{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// SetWithTTL stores a value with a specific TTL.
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &Item{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
}

// Delete removes a key from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear removes all items from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*Item)
}

// Len returns the number of items in the cache.
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Stop stops the background cleanup goroutine.
func (c *Cache) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stopped {
		return
	}

	c.stopped = true
	close(c.stopCh)
}

// cleanupLoop periodically removes expired items.
func (c *Cache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.deleteExpired()
		case <-c.stopCh:
			return
		}
	}
}

func (c *Cache) deleteExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
			delete(c.items, key)
		}
	}
}
