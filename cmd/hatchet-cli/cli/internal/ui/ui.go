package ui

import (
	"embed"
	"io/fs"
)

//go:embed all:assets
var embedded embed.FS

func Assets() (fs.FS, error) {
	return fs.Sub(embedded, "assets")
}

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
