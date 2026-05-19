package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	lockTimeout     = 5 * time.Second
	lockRetryDelay  = 50 * time.Millisecond
	maxLockAttempts = 100
)

// acquireLock attempts to acquire a lock file for config operations
func acquireLock() (func(), error) {
	if HomeDir == "" {
		return nil, fmt.Errorf("home directory not set")
	}

	lockFile := filepath.Join(HomeDir, ".hatchet", "config.lock")
	hatchetDir := filepath.Join(HomeDir, ".hatchet")

	// Ensure .hatchet directory exists
	if err := os.MkdirAll(hatchetDir, 0755); err != nil {
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
