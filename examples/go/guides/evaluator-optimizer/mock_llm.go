package main

var generateCount int

func MockGenerate(prompt string) string {
	generateCount++
	if generateCount == 1 {
		return "Check out our product! Buy now!"
	}
	return "Discover how our tool saves teams 10 hours/week. Try it free."
}

type EvalResult struct {
	Score    float64
	Feedback string
}

func MockEvaluate(draft string) EvalResult {
	if len(draft) < 40 {
		return EvalResult{Score: 0.4, Feedback: "Too short and pushy. Add a specific benefit and soften the CTA."}
	}
	return EvalResult{Score: 0.9, Feedback: "Clear value prop, appropriate tone."}
}
