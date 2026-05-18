package analytics

import "testing"

// Verify compile-time interface implementations.
func TestPropertyImplementations(t *testing.T) {
	var _ Property = (Properties)(nil)
	var _ Property = Increment(0)
}

// Verify mixed Property types can coexist and be handled correctly.
func TestMixedPropertyTypes(t *testing.T) {
	props := []Property{
		Properties{
			"resource": "workflow",
			"action":   "create",
		},
		Increment(5),
	}

	var (
		foundProperties bool
		totalIncrement  int
	)

	for _, prop := range props {
		switch p := prop.(type) {
		case Properties:
			foundProperties = true

			if p["resource"] != "workflow" {
				t.Fatalf("expected resource=workflow, got %v", p["resource"])
			}

		case Increment:
			totalIncrement += int(p)
		}
	}

	if !foundProperties {
		t.Fatal("expected Properties in Property slice")
	}

	if totalIncrement != 5 {
		t.Fatalf("expected increment=5, got %d", totalIncrement)
	}
}

// Verify Increment supports the intended counting use-case.
func TestIncrementPropertyUsage(t *testing.T) {
	props := []Property{
		Properties{
			"span_type": "workflow",
		},
		Increment(10),
	}

	var increment int

	for _, prop := range props {
		if inc, ok := prop.(Increment); ok {
			increment = int(inc)
		}
	}

	if increment != 10 {
		t.Fatalf("expected increment=10, got %d", increment)
	}
}