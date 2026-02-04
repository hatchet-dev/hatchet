package syncx

import (
	"sync"
)

// Map is a type-safe concurrent map with comparable keys and any values.
type Map[K comparable, V any] struct {
	m sync.Map
}

// Store sets the value for a key.
func (m *Map[K, V]) Store(key K, value V) {
	m.m.Store(key, value)
}

// Load returns the value for a key if it exists.
func (m *Map[K, V]) Load(key K) (value V, ok bool) {
	v, ok := m.m.Load(key)
	if !ok {
		return value, ok
	}
	// The type assertion is safe because we control the types in Store.
	return v.(V), ok
}

// Delete deletes the value for a key.
func (m *Map[K, V]) Delete(key K) {
	m.m.Delete(key)
}

// LoadOrStore loads or stores the value for a key.
func (m *Map[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	a, loaded := m.m.LoadOrStore(key, value)
	return a.(V), loaded
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, the range stops.
func (m *Map[K, V]) Range(f func(key K, value V) bool) {
	m.m.Range(func(key, value any) bool {
		return f(key.(K), value.(V))
	})
}

// Len returns the number of elements in the map.
func (m *Map[K, V]) Len() int {
	length := 0
	m.m.Range(func(key, value any) bool {
		length++
		return true
	})
	return length
}
