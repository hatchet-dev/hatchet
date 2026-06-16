package main

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestWaitForRegistrationSuccess(t *testing.T) {
	registered := make(chan error, 1)
	registered <- nil

	if err := waitForRegistration(registered, time.Second); err != nil {
		t.Fatalf("waitForRegistration() error = %v, want nil", err)
	}
}

func TestWaitForRegistrationPropagatesError(t *testing.T) {
	wantErr := errors.New("registration failed")
	registered := make(chan error, 1)
	registered <- wantErr

	if err := waitForRegistration(registered, time.Second); !errors.Is(err, wantErr) {
		t.Fatalf("waitForRegistration() error = %v, want %v", err, wantErr)
	}
}

func TestWaitForRegistrationTimeout(t *testing.T) {
	registered := make(chan error, 1)

	err := waitForRegistration(registered, time.Millisecond)
	if err == nil {
		t.Fatal("waitForRegistration() error = nil, want timeout")
	}

	if !strings.Contains(err.Error(), "timed out after 1ms waiting for workflow registration") {
		t.Fatalf("waitForRegistration() error = %q, want timeout message", err)
	}
}

func TestLoadTestTimingUsesDefaultRegistrationTimeout(t *testing.T) {
	timing := calculateLoadTestTiming(LoadTestConfig{
		Duration: 240 * time.Second,
		Wait:     60 * time.Second,
	})

	if timing.registrationTimeout != time.Minute {
		t.Fatalf("registrationTimeout = %s, want 1m0s", timing.registrationTimeout)
	}

	if timing.activeWindow != 310*time.Second {
		t.Fatalf("activeWindow = %s, want 5m10s", timing.activeWindow)
	}

	if timing.safetyTimeout != 400*time.Second {
		t.Fatalf("safetyTimeout = %s, want 6m40s", timing.safetyTimeout)
	}
}

func TestLoadTestTimingAccountsForWorkerDelayInSafetyOnly(t *testing.T) {
	timing := calculateLoadTestTiming(LoadTestConfig{
		Duration:    240 * time.Second,
		Wait:        120 * time.Second,
		WorkerDelay: 120 * time.Second,
	})

	if timing.activeWindow != 370*time.Second {
		t.Fatalf("activeWindow = %s, want 6m10s", timing.activeWindow)
	}

	if timing.safetyTimeout != 580*time.Second {
		t.Fatalf("safetyTimeout = %s, want 9m40s", timing.safetyTimeout)
	}
}

func TestLoadTestTimingUsesExplicitRegistrationTimeout(t *testing.T) {
	timing := calculateLoadTestTiming(LoadTestConfig{
		RegistrationTimeout: 20 * time.Second,
		WorkerDelay:         5 * time.Second,
		Duration:            30 * time.Second,
		Wait:                40 * time.Second,
	})

	if timing.registrationTimeout != 20*time.Second {
		t.Fatalf("registrationTimeout = %s, want 20s", timing.registrationTimeout)
	}

	if timing.activeWindow != 80*time.Second {
		t.Fatalf("activeWindow = %s, want 1m20s", timing.activeWindow)
	}

	if timing.safetyTimeout != 135*time.Second {
		t.Fatalf("safetyTimeout = %s, want 2m15s", timing.safetyTimeout)
	}
}
