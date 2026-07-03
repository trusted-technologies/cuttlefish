package web

import "embed"

// StaticFS contains the built React SPA assets.
//
//go:embed static/*
//go:embed static/assets/*
var StaticFS embed.FS
