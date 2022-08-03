package strava

import (
	"embed"
	"io/fs"
)

// Static site resources.
//go:embed static
var staticFiles embed.FS

// StaticFiles from static/ embeded in this binary.
func StaticFiles() (fs.FS, error) {
	return fs.Sub(fs.FS(staticFiles), "static")
}
