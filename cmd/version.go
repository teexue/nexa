package main

import (
	"fmt"

	"github.com/teexue/nexa/core/version"
)

// runVersion prints the build version injected at release time via
// -ldflags -X core/version.Version=... (see Makefile / release workflow).
func runVersion(args []string) {
	fmt.Println(version.Version)
}
