// Package site embeds the optimized browser client in the server binary.
package site

import (
	"embed"
	"io/fs"
)

//go:embed build/*
var files embed.FS

// Content returns the embedded browser client with build as its root.
func Content() fs.FS {
	content, err := fs.Sub(files, "build")
	if err != nil {
		panic(err)
	}
	return content
}
