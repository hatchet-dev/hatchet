package serverutils

import (
	"sync"
	"time"

	"github.com/hatchet-dev/hatchet/internal/cache"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

// RateLimitEntry tracks attempts for an IP
type RateLimitEntry struct {
	Count     int
	FirstSeen time.Time
}

// AuthAPIRateLimiter handles IP-based rate limiting for API operations
type AuthAPIRateLimiter struct {
	cache  *cache.TTLCache[string, RateLimitEntry]
	config *server.ServerConfig
	mu     sync.RWMutex
}

// NewAuthAPIRateLimiter creates a new rate limiter for API operations
func NewAuthAPIRateLimiter(config *server.ServerConfig) *AuthAPIRateLimiter {
	return &AuthAPIRateLimiter{
		cache:  cache.NewTTL[string, RateLimitEntry](),
		config: config,
	}
}

// IsAllowed checks if an IP address is allowed to make an API request
func (r *AuthAPIRateLimiter) IsAllowed(scope, ip string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := scope + ":" + ip
	now := time.Now()
	entry, exists := r.cache.Get(key)

	if !exists {
		// First attempt from this IP
		r.cache.Set(key, RateLimitEntry{
			Count:     1,
			FirstSeen: now,
		}, r.config.Runtime.APIRateLimitWindow)
		return true
	}

	// Check if the window has expired
	if now.Sub(entry.FirstSeen) > r.config.Runtime.APIRateLimitWindow {
		// Window expired, reset counter
		r.cache.Set(key, RateLimitEntry{
			Count:     1,
			FirstSeen: now,
		}, r.config.Runtime.APIRateLimitWindow)
		return true
	}

	// Within the window, check if limit exceeded
	if entry.Count >= r.config.Runtime.APIRateLimit {
		return false
	}

	// Increment counter
	entry.Count++
	r.cache.Set(key, entry, r.config.Runtime.APIRateLimitWindow)
	return true
}

// Stop cleans up the rate limiter
func (r *AuthAPIRateLimiter) Stop() {
	r.cache.Stop()
}

func (r *AuthAPIRateLimiter) GetWindow() time.Duration {
	return r.config.Runtime.APIRateLimitWindow
}
