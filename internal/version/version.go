// Package version reports the version of the running binary.
package version

import "runtime/debug"

// Version is set at build time:
//
//	go build -ldflags "-X github.com/adagiumm/adag-lens/internal/version.Version=v0.1.0"
//
// The linker can only set package-level string variables, which is why this
// is a variable at all. Everything else should call String.
var Version = ""

// String returns the build-time version if set; otherwise the module version
// recorded by "go install ...@vX.Y.Z"; otherwise "dev".
func String() string {
	if Version != "" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return "dev"
}
