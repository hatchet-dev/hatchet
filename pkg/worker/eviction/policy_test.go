package eviction

import (
	"testing"
	"time"
)

func TestDefaultPolicy(t *testing.T) {
	p := DefaultPolicy()
	if p == nil {
		t.Fatal("expected non-nil policy")
	}
	if p.TTL == nil {
		t.Fatal("expected non-nil TTL")
	}
	if *p.TTL != 15*time.Minute {
		t.Fatalf("expected 15m TTL, got %v", *p.TTL)
	}
	if !p.AllowCapacityEviction {
		t.Fatal("expected AllowCapacityEviction to be true")
	}
	if p.Priority != 0 {
		t.Fatalf("expected priority 0, got %d", p.Priority)
	}
}

func TestDefaultManagerConfig(t *testing.T) {
	c := DefaultManagerConfig()
	if c.CheckInterval != 1*time.Second {
		t.Fatalf("expected 1s check interval, got %v", c.CheckInterval)
	}
	if c.ReserveSlots != 0 {
		t.Fatalf("expected 0 reserve slots, got %d", c.ReserveSlots)
	}
	if c.MinWaitForCapacityEviction != 10*time.Second {
		t.Fatalf("expected 10s min wait, got %v", c.MinWaitForCapacityEviction)
	}
}

func TestErrEvicted(t *testing.T) {
	if ErrEvicted == nil {
		t.Fatal("expected non-nil ErrEvicted")
	}
	if ErrEvicted.Error() != "durable run evicted" {
		t.Fatalf("unexpected error message: %s", ErrEvicted.Error())
	}
}
