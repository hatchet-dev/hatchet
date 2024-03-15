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

func New() *Cache {
	return &Cache{
		cache: expirable.NewLRU[string, interface{}](512, nil, time.Minute*1),
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
