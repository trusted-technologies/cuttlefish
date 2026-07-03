package web

import "embed"

// TemplatesFS contains the HTML templates used by the master UI.
//
//go:embed templates/*.html
var TemplatesFS embed.FS

// StaticFS contains static assets (CSS, JS).
//
//go:embed static/*
var StaticFS embed.FS
