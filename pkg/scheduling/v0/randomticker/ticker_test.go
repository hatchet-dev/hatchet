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
