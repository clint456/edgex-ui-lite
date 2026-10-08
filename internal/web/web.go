// Package web embeds the static console frontend.
package web

import (
	"embed"
	"io/fs"
)

//go:embed static
var embedded embed.FS

// Static returns the embedded frontend rooted at the static directory.
func Static() (fs.FS, error) {
	return fs.Sub(embedded, "static")
}
