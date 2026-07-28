// Package web embeds the built SPA so the Go binary serves a self-contained
// UI. The assets live under web/dist and are produced by `npm run build`; a
// placeholder index.html is committed so the embed always compiles.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var assets embed.FS

// FS returns the embedded SPA build rooted at dist.
func FS() fs.FS {
	sub, err := fs.Sub(assets, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}