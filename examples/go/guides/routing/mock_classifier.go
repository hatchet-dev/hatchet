package main

import "strings"

func MockClassify(message string) string {
	lower := strings.ToLower(message)
	for _, w := range []string{"bug", "error", "help"} {
		if strings.Contains(lower, w) {
			return "support"
		}
	}
	for _, w := range []string{"price", "buy", "plan"} {
		if strings.Contains(lower, w) {
			return "sales"
		}
	}
	return "other"
}

func MockReply(message, role string) string {
	switch role {
	case "support":
		return "[Support] I can help with that technical issue. Let me look into: " + message
	case "sales":
		return "[Sales] Great question about pricing! Here's what I can tell you about: " + message
	default:
		return "[General] Thanks for reaching out. Regarding: " + message
	}
}
