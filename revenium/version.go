package revenium

import (
	"runtime/debug"
	"sync"
)

const (
	// ModuleName is the Go module name for this middleware
	ModuleName = "github.com/revenium/revenium-middleware-openai-go"
	// DefaultVersion is used when version cannot be determined
	DefaultVersion = "0.0.0-dev"
)

var (
	// Cached version info
	middlewareSourceOnce sync.Once
	middlewareSourceVal  string
)

// GetMiddlewareSource returns the middleware source string with version
// Format: revenium-middleware-openai-go@{version}
// Uses runtime/debug.ReadBuildInfo() for zero-maintenance version detection
func GetMiddlewareSource() string {
	middlewareSourceOnce.Do(func() {
		version := DefaultVersion

		if info, ok := debug.ReadBuildInfo(); ok {
			// Check if this is the main module
			if info.Main.Path == ModuleName && info.Main.Version != "" && info.Main.Version != "(devel)" {
				version = info.Main.Version
			} else {
				// Check dependencies for our module (when used as a library)
				for _, dep := range info.Deps {
					if dep.Path == ModuleName {
						version = dep.Version
						break
					}
				}
			}
		}

		middlewareSourceVal = "revenium-middleware-openai-go@" + version
	})

	return middlewareSourceVal
}

// GetVersion returns just the version string
func GetVersion() string {
	version := DefaultVersion

	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Path == ModuleName && info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
		for _, dep := range info.Deps {
			if dep.Path == ModuleName {
				return dep.Version
			}
		}
	}

	return version
}
