package main

import "testing"

func TestExecutionTallyKeysOnActionAndEvent(t *testing.T) {
	tally := newExecutionTally()

	first := []executionKey{
		{action: "load-test-0:step-0", eventID: 1},
		// The same event on another fanout workflow and on another DAG step
		// are distinct task runs, not duplicates.
		{action: "load-test-1:step-0", eventID: 1},
		{action: "load-test-0:step-1", eventID: 1},
		{action: "load-test-0:step-0", eventID: 2},
	}
	for _, key := range first {
		if dup := tally.record(key); dup {
			t.Fatalf("record(%+v) = duplicate, want first execution", key)
		}
	}

	repeat := executionKey{action: "load-test-0:step-0", eventID: 1}
	if dup := tally.record(repeat); !dup {
		t.Fatalf("record(%+v) = first execution, want duplicate", repeat)
	}

	count, uniques := tally.counts()
	if count != 5 || uniques != 4 {
		t.Fatalf("counts() = (%d, %d), want (5, 4)", count, uniques)
	}
}
