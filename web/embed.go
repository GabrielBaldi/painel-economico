// Package web embeds the compiled front-end so the server ships as a single
// binary. Run `npm run build` in this folder before building the Go code.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Dist returns the files produced by `npm run build`, rooted at dist/.
func Dist() (fs.FS, error) {
	return fs.Sub(dist, "dist")
}
