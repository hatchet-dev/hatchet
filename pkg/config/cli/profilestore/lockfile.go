package profilestore

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	lockFileName    = "config.lock"
	lockTimeout     = 5 * time.Second
	lockRetryDelay  = 50 * time.Millisecond
	maxLockAttempts = 100
)

// acquireLock attempts to acquire the profile store lock file (config.lock in
// dir) for config operations. The protocol is shared with every other writer
// of the profile file: exclusive create, retry every 50ms up to 100 attempts,
// and locks older than 5s are considered stale.
func acquireLock(dir string) (func(), error) {
	if dir == "" {
		return nil, fmt.Errorf("home directory not set")
	}

	lockFile := filepath.Join(dir, lockFileName)

	// Ensure .hatchet directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create hatchet directory: %w", err)
	}

	attempts := 0
	for attempts < maxLockAttempts {
		// Try to create the lock file exclusively
		f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err == nil {
			// Successfully acquired lock
			// Write timestamp to lock file
			timestamp := time.Now().Format(time.RFC3339)
			_, err := f.WriteString(timestamp)
			f.Close()

			if err != nil {
				os.Remove(lockFile)
				return nil, fmt.Errorf("failed to write to lock file: %w", err)
			}

			// Return cleanup function
			return func() {
				os.Remove(lockFile)
			}, nil
		}

		// Lock file exists, check if it's stale
		if os.IsExist(err) {
			stat, statErr := os.Stat(lockFile)
			if statErr == nil {
				// Check if lock is expired
				if time.Since(stat.ModTime()) > lockTimeout {
					// Lock is stale, remove it and retry
					os.Remove(lockFile)
					continue
				}
			}
		}

		// Wait and retry
		time.Sleep(lockRetryDelay)
		attempts++
	}

	return nil, fmt.Errorf("failed to acquire lock after %d attempts", maxLockAttempts)
}
