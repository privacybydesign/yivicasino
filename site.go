package main

import (
	"embed"
	"io/fs"
)

// Only www/ is embedded, so nothing else in the repository can be served.
// Files starting with "." or "_" are left out by go:embed.
//
//go:embed www
var siteFiles embed.FS

func publicSite() (fs.FS, error) {
	return fs.Sub(siteFiles, "www")
}
