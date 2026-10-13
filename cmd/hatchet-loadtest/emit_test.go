package main

import (
	"slices"
	"testing"

	"github.com/hatchet-dev/hatchet/pkg/loadtest/eventkeys"
)

func TestConcurrencyEventKeyIsOptIn(t *testing.T) {
	available := availableEventKeys(true)

	got, err := resolveEventKeys(nil, nil, available)
	if err != nil {
		t.Fatal(err)
	}
	if want := []eventkeys.EventKey{eventkeys.EventKeyDefault}; !slices.Equal(got, want) {
		t.Fatalf("no flags: got %v, want %v", got, want)
	}

	got, err = resolveEventKeys([]string{"default", "dag"}, nil, available)
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(got, eventkeys.EventKeyConcurrency) {
		t.Fatalf("--select without concurrency published it: %v", got)
	}

	got, err = resolveEventKeys([]string{"default", "concurrency"}, nil, available)
	if err != nil {
		t.Fatal(err)
	}
	if want := []eventkeys.EventKey{eventkeys.EventKeyDefault, eventkeys.EventKeyConcurrency}; !slices.Equal(got, want) {
		t.Fatalf("--select with concurrency: got %v, want %v", got, want)
	}
}
