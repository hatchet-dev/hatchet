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

	"github.com/hatchet-dev/hatchet/pkg/scheduling/v0/randomticker"
)

func TestRandomTicker(t *testing.T) {
	t.Parallel()

	minDuration := time.Duration(10)
	maxDuration := time.Duration(20)

	// tick can take a little longer since we're not adjusting it to account for
	// processing.
	precision := time.Duration(4)

	rt := randomticker.NewRandomTicker(minDuration*time.Millisecond, maxDuration*time.Millisecond)
	for i := 0; i < 5; i++ {
		t0 := time.Now()
		t1 := <-rt.C
		td := t1.Sub(t0)
		if td < minDuration*time.Millisecond {
			t.Fatalf("tick was shorter than expected: %s", td)
		} else if td > (maxDuration+precision)*time.Millisecond {
			t.Fatalf("tick was longer than expected: %s", td)
		}
	}
	rt.Stop()
	time.Sleep((maxDuration + precision) * time.Millisecond)
	select {
	case v, ok := <-rt.C:
		if ok || !v.IsZero() {
			t.Fatal("ticker did not shut down")
		}
	default:
		t.Fatal("expected to receive close channel signal")
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
