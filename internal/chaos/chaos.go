package chaos

import (
	"math/rand"
)

var (
	chaosProbability = 0.01  // Default probability (1 in 100)
	enabled          = false // Chaos Monkey is disabled by default
)

// Configure sets the probability for chaos to occur.
// probability should be a value between 0 and 1.
// Set probability to 0 to effectively disable chaos.
func Configure(probability float64) {
	if probability < 0 || probability > 1 {
		panic("probability must be between 0 and 1")
	}
	chaosProbability = probability
}

// Enable activates the Chaos Monkey.
func Enable() {
	enabled = true
}

// Disable deactivates the Chaos Monkey, ensuring it never triggers.
func Disable() {
	enabled = false
}

// ShouldChaos returns true if the Chaos Monkey should be active in the current context.
// Chaos will only be considered if it is enabled and meets the probability threshold.
func ShouldChaos() bool {
	if !enabled {
		return false
	}
	return rand.Float64() < chaosProbability // #nosec G404
}
