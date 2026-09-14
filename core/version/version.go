// Package version holds the single source of truth for the build version.
//
// The value is injected at build time via:
//
//	go build -ldflags "-X github.com/teexue/nexa/core/version.Version=v1.2.3"
//
// Local/dev builds without the linker flag fall back to "dev". Releases are
// tagged with a version and the Makefile / CI injects the tag into this var,
// so version numbers never have to be hard-coded in source.
package version

// Version is the release version. Overridden by -ldflags at build time.
var Version = "dev"
