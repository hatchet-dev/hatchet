package cache

import (
	"time"

	"github.com/patrickmn/go-cache"
)

type Cacheable interface {
	// Set sets a value in the cache with the given key
	Set(key string, value interface{})

	// Get gets a value from the cache with the given key
	Get(key string) (interface{}, bool)
}

type Cache struct {
	cache      *cache.Cache
	expiration time.Duration
}

func (c *Cache) Set(key string, value interface{}) {
	c.cache.Set(key, value, c.expiration)
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
		expiration: duration,
		cache:      cache.New(duration, 2*time.Minute),
	}
}

func MakeCacheable[T any](cache Cacheable, id string, f func() (*T, error)) (*T, error) {
	if v, ok := cache.Get(id); ok {
		return v.(*T), nil
	}

	v, err := f()
	if err != nil {
		return nil, err
	}

	cache.Set(id, v)

	return v, nil
}
