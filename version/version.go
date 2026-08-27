// Package version carries the build identity of the running binary.
//
// The values below are placeholders overwritten at LINK time by the release
// build (see .github/workflows/release.yml and the Makefile's LDFLAGS):
//
//	go build -ldflags "-X olympsis-server/version.Version=v0.9.5 ..."
//
// They are deliberately NOT read from the environment. An env var can be
// changed without rebuilding — which would let the server report a version it
// is not actually running — whereas a linker-set value is welded to the binary
// and cannot drift from the image tag it was published under.
package version

import "fmt"

var (
	// Version is the git tag this binary was cut from, e.g. "v0.9.5".
	// Stays "dev" for local `go build` / `go run`.
	Version = "dev"

	// Commit is the short git SHA the tag pointed at, e.g. "6351f14".
	Commit = "none"

	// BuildTime is the UTC RFC3339 timestamp of the build, e.g.
	// "2026-08-26T22:40:00Z".
	BuildTime = "unknown"
)

// Info is the JSON shape embedded in the /v1/health response.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"buildTime"`
}

// Get returns the build identity as a serialisable struct.
func Get() Info {
	return Info{Version: Version, Commit: Commit, BuildTime: BuildTime}
}

// String renders a one-line summary for the startup log, e.g.
// "v0.9.5 (6351f14, built 2026-08-26T22:40:00Z)".
func String() string {
	return fmt.Sprintf("%s (%s, built %s)", Version, Commit, BuildTime)
}
