//go:build tools

// Package main (tools) pins build-time dependencies so they appear in go.sum
// and are available without re-fetching.
package main

import (
	// semver is used by internal/hugo to compare Hugo binary versions.
	_ "github.com/Masterminds/semver/v3"
)
