package cache

import (
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

type Cacheable interface {
	// Set sets a value in the cache with the given key
	Set(key string, value interface{})

	// Get gets a value from the cache with the given key
	Get(key string) (interface{}, bool)
}

type Cache struct {
	cache *expirable.LRU[string, interface{}]
}

func (c *Cache) Set(key string, value interface{}) {
	c.cache.Add(key, value)
}

func (c *Cache) Get(key string) (interface{}, bool) {
	return c.cache.Get(key)
}

func New(duration time.Duration) *Cache {
	if duration == 0 {
		// consider a duration of 0 a very short expiry instead of no expiry
		duration = 1 * time.Millisecond
	}
	return &Cache{
		cache: expirable.NewLRU[string, interface{}](512, nil, duration),
	}
}
