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
