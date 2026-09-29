//go:build !e2e && !load && !rampup && !integration

// Copyright (c) 2020 Filip Wojciechowski

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package randomticker_test

import (
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/randomticker"
)

func TestRandomTicker(t *testing.T) {
	t.Parallel()

	const (
		minDuration = 10 * time.Millisecond
		maxDuration = 20 * time.Millisecond
		ticks       = 50
		// The ticker sends without blocking, so a tick that fires while this
		// goroutine is not parked on the channel is dropped and the next one
		// lands up to maxDuration later. Ticks average well under maxDuration,
		// so the cumulative bound leaves room for roughly twenty dropped ticks
		// plus timer lateness on a loaded runner, while a ticker that ignores
		// maxDuration (for example a fixed 30ms cadence, 1.5s in total) still
		// fails.
		slack = 10 * maxDuration
	)

	// Every timer is armed after start and after the previous tick was
	// delivered, so the k-th tick cannot arrive before k*minDuration. That
	// bound is strict; the upper bound is checked once, cumulatively.
	start := time.Now()
	rt := randomticker.NewRandomTicker(minDuration, maxDuration)

	for i := 1; i <= ticks; i++ {
		select {
		case <-rt.C:
		case <-time.After(5 * time.Second):
			t.Fatalf("no tick received for tick %d", i)
		}

		if elapsed := time.Since(start); elapsed < time.Duration(i)*minDuration {
			t.Fatalf("tick %d arrived after %s, sooner than %d ticks of at least %s allow", i, elapsed, i, minDuration)
		}
	}

	if elapsed := time.Since(start); elapsed > ticks*maxDuration+slack {
		t.Fatalf("%d ticks took %s, longer than %d ticks of at most %s plus %s slack", ticks, elapsed, ticks, maxDuration, slack)
	}

	rt.Stop()

	// The channel is closed by the ticker goroutine after Stop has been
	// acknowledged, so it may close a moment after Stop returns.
	deadline := time.Now().Add(5 * time.Second)
	for {
		select {
		case v, ok := <-rt.C:
			if ok || !v.IsZero() {
				t.Fatal("ticker did not shut down")
			}
			return
		default:
		}

		if time.Now().After(deadline) {
			t.Fatal("expected the tick channel to be closed after Stop")
		}

		time.Sleep(time.Millisecond)
	}
}

// TestRandomTickerUnblockingIssue is a regression test for a bug in the original implementation
// where the ticker would stop generating new events if no one was reading from the channel.
func TestRandomTickerUnblockingIssue(t *testing.T) {
	minDuration := 50 * time.Millisecond
	maxDuration := 100 * time.Millisecond

	// Create the random ticker
	rt := randomticker.NewRandomTicker(minDuration, maxDuration)
	defer rt.Stop()

	// Get the first tick to make sure it's working
	select {
	case <-rt.C:
		// Good, we got a tick
	case <-time.After(maxDuration * 2):
		t.Fatal("didn't receive initial tick in the expected timeframe")
	}

	// Now simulate a scenario where the consumer isn't reading from the channel
	// by just waiting without reading from rt.C
	time.Sleep(maxDuration * 2)

	// After ignoring the channel for a while, now try to read from it again
	// With the bug, this would hang because no new ticks are generated
	// With the fix, we should get a new tick within 2*maxDuration

	tickCount := 0
	timeout := time.After(maxDuration * 5) // Give it plenty of time to tick

	for tickCount < 3 { // Try to get 3 more ticks
		select {
		case <-rt.C:
			tickCount++
		case <-timeout:
			// With the original implementation, we'll hit this timeout
			t.Fatalf("only received %d ticks after ignoring the channel; ticker appears stuck", tickCount)
			return
		}
	}

	// If we get here, the ticker continued to generate events even when
	// we weren't reading from the channel, which means the fix is working
}
