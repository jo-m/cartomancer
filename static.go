package main

import (
	"embed"
	"io/fs"
)

const staticDir = "static"

//go:embed static
var staticEmbed embed.FS

func getStaticFS() (fs.FS, error) {
	sub, err := fs.Sub(staticEmbed, staticDir)
	if err != nil {
		return nil, err
	}
	return sub, nil
}
