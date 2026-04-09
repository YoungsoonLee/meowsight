package web

import "embed"

// StaticFS holds the embedded static web assets.
//
//go:embed static/*
var StaticFS embed.FS
