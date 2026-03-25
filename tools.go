//go:build tools

// Package main (tools) pins tool and future-phase dependencies so they appear
// in go.sum and are available without re-fetching during phase integration.
package main

import (
	// semver is used in Phase 5 (internal/hugo) to compare Hugo binary versions.
	_ "github.com/Masterminds/semver/v3"
)
