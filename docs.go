package apub

import "embed"

//go:embed doc/*

// DocFS contains the apas overview documentation
// to be served by [http.FileServer].
var DocFS embed.FS
