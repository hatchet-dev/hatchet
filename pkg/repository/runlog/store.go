package runlog

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// ErrNotFound is returned for a missing object, and by the readers for a task, run or payload that the
// log does not hold (including a tenant-day whose archive is not complete).
var ErrNotFound = errors.New("not found in run log")

// Store is the object store the run log is written to.
type Store interface {
	Put(ctx context.Context, key string, body []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	// GetRange returns length bytes starting at offset.
	GetRange(ctx context.Context, key string, offset, length int64) ([]byte, error)
	Exists(ctx context.Context, key string) (bool, error)
}

// MemoryStore is a Store for tests.
type MemoryStore struct {
	mu      sync.Mutex
	objects map[string][]byte
	puts    int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{objects: make(map[string][]byte)}
}

func (s *MemoryStore) Put(_ context.Context, key string, body []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.objects[key] = append([]byte(nil), body...)
	s.puts++

	return nil
}

func (s *MemoryStore) Get(_ context.Context, key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	body, ok := s.objects[key]

	if !ok {
		return nil, fmt.Errorf("%s: %w", key, ErrNotFound)
	}

	return append([]byte(nil), body...), nil
}

func (s *MemoryStore) GetRange(_ context.Context, key string, offset, length int64) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	body, ok := s.objects[key]

	if !ok {
		return nil, fmt.Errorf("%s: %w", key, ErrNotFound)
	}

	if offset < 0 || length < 0 || offset+length > int64(len(body)) {
		return nil, fmt.Errorf("range %d+%d outside %s (%d bytes)", offset, length, key, len(body))
	}

	return append([]byte(nil), body[offset:offset+length]...), nil
}

func (s *MemoryStore) Exists(_ context.Context, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.objects[key]

	return ok, nil
}

// Keys lists the stored keys with the given prefix, sorted.
func (s *MemoryStore) Keys(prefix string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	keys := make([]string, 0, len(s.objects))

	for k := range s.objects {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}

	sort.Strings(keys)

	return keys
}

// Puts counts the Put calls so far, including overwrites.
func (s *MemoryStore) Puts() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.puts
}
