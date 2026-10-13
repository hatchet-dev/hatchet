package main

import (
	"slices"
	"testing"

	"github.com/hatchet-dev/hatchet/pkg/loadtest/eventkeys"
)

func TestWithConcurrencyWorkload(t *testing.T) {
	selected := []eventkeys.EventKey{eventkeys.EventKeyDefault, eventkeys.EventKeyDag}

	got := withConcurrencyWorkload(slices.Clone(selected), nil)
	want := []eventkeys.EventKey{eventkeys.EventKeyDefault, eventkeys.EventKeyDag, eventkeys.EventKeyConcurrency}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	if got := withConcurrencyWorkload(slices.Clone(want), nil); !slices.Equal(got, want) {
		t.Fatalf("already selected: got %v, want %v", got, want)
	}

	if got := withConcurrencyWorkload(slices.Clone(selected), []string{" concurrency "}); !slices.Equal(got, selected) {
		t.Fatalf("excluded: got %v, want %v", got, selected)
	}
}
