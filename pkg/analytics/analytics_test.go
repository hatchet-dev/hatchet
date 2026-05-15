package analytics

import "testing"

func TestPropertyImplementations(t *testing.T) {
	var _ Property = (Properties)(nil)
	var _ Property = Increment(0)
}
