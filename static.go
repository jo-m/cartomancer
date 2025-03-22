package main

import (
	"embed"
	"fmt"
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

	// TODO: remove
	fmt.Println(staticEmbed.Open("static/test.html"))
	fmt.Println(sub.Open("test.html"))

	return sub, nil
}
