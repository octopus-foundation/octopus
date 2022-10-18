package testdata

import "embed"

//go:embed "*"
var TestData embed.FS
