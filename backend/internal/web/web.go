package web

import "embed"

// Dist contains the Vue production build.
//
//go:embed dist/*
var Dist embed.FS
