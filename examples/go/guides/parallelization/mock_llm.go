package main

import "strings"

func MockGenerateContent(message string) string {
	return "Here is a helpful response to: " + message
}

type SafetyResult struct {
	Safe   bool
	Reason string
}

func MockSafetyCheck(message string) SafetyResult {
	if strings.Contains(strings.ToLower(message), "unsafe") {
		return SafetyResult{Safe: false, Reason: "Content flagged as potentially unsafe."}
	}
	return SafetyResult{Safe: true, Reason: "Content is appropriate."}
}

type EvalResult struct {
	Score    float64
	Approved bool
}

func MockEvaluateContent(content string) EvalResult {
	score := 0.3
	if len(content) > 20 {
		score = 0.85
	}
	return EvalResult{Score: score, Approved: score >= 0.7}
}
