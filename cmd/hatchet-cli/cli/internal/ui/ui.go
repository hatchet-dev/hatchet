// Package ui holds the Hatchet dashboard web bundle that is embedded into the
// CLI binary. During a release build the compiled frontend is copied into the
// assets directory (see hack/build/embed-ui.sh); local `go build` invocations
// ship only a placeholder, in which case Bundled reports false.
package ui

import (
	"embed"
	"io/fs"
)

//go:embed all:assets
var embedded embed.FS

// Assets returns the embedded UI bundle rooted at the assets directory.
func Assets() (fs.FS, error) {
	return fs.Sub(embedded, "assets")
}

// Bundled reports whether a real UI build was embedded into this binary. It is
// false for plain `go build` binaries that only carry the placeholder.
func Bundled() bool {
	sub, err := Assets()
	if err != nil {
		return false
	}

	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return false
	}

	return true
}
