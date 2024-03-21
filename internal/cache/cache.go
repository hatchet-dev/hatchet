package cache

import (
	"sync"
	"time"
)

// item represents a cache item with a value and an expiration time.
type item[V any] struct {
	value  V
	expiry time.Time
}

// isExpired checks if the cache item has expired.
func (i item[V]) isExpired() bool {
	return time.Now().After(i.expiry)
}

// TTLCache is a generic cache implementation with support for time-to-live
// (TTL) expiration.
type TTLCache[K comparable, V any] struct {
	items map[K]item[V]    // The map storing cache items.
	mu    sync.Mutex       // Mutex for controlling concurrent access to the cache.
	stop  chan interface{} // Channel to stop the goroutine that removes expired items.
}

// NewTTL creates a new TTLCache instance and starts a goroutine to periodically
// remove expired items every 5 seconds.
func NewTTL[K comparable, V any]() *TTLCache[K, V] {
	c := &TTLCache[K, V]{
		items: make(map[K]item[V]),
		stop:  make(chan interface{}),
	}

	go func() {
		// Create a new ticker to remove expired items every 5 seconds.
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-c.stop:
				return
			case <-ticker.C:
				c.mu.Lock()

				// Iterate over the cache items and delete expired ones.
				for key, item := range c.items {
					if item.isExpired() {
						delete(c.items, key)
					}
				}

				c.mu.Unlock()
			}
		}
	}()

	return c
}

func (c *TTLCache[K, V]) Stop() {
	close(c.stop)
}

// Set adds a new item to the cache with the specified key, value, and
// time-to-live (TTL).
func (c *TTLCache[K, V]) Set(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = item[V]{
		value:  value,
		expiry: time.Now().Add(ttl),
	}
}

// Get retrieves the value associated with the given key from the cache.
func (c *TTLCache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, found := c.items[key]
	if !found {
		// If the key is not found, return the zero value for V and false.
		return item.value, false
	}

	if item.isExpired() {
		// If the item has expired, remove it from the cache and return the
		// value and false.
		delete(c.items, key)
		return item.value, false
	}

	// Otherwise return the value and true.
	return item.value, true
}

// Remove removes the item with the specified key from the cache.
func (c *TTLCache[K, V]) Remove(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Delete the item with the given key from the cache.
	delete(c.items, key)
}

// Pop removes and returns the item with the specified key from the cache.
func (c *TTLCache[K, V]) Pop(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, found := c.items[key]
	if !found {
		// If the key is not found, return the zero value for V and false.
		return item.value, false
	}

	// If the key is found, delete the item from the cache.
	delete(c.items, key)

	if item.isExpired() {
		// If the item has expired, return the value and false.
		return item.value, false
	}

	// Otherwise return the value and true.
	return item.value, true
}
