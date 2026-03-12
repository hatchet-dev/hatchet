//go:build ignore

// Third-party integration - requires: go get github.com/otiai10/gosseract/v2
// and Tesseract C library. Build tag excludes from default build (no native deps in CI).
// See: /guides/document-processing

package integrations

import "github.com/otiai10/gosseract/v2"

// > Tesseract usage
func ParseDocument(content []byte) (string, error) {
	client := gosseract.NewClient()
	defer client.Close()
	return client.SetImageFromBytes(content).GetText()
}

