package main

import "fmt"

// parseDocument is a mock - no external OCR dependency.
func parseDocument(content []byte) string {
	return fmt.Sprintf("Parsed text from %d bytes", len(content))
}
