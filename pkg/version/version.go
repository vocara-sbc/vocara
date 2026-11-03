// Package version provides version information injected at build time.
package version

// The following variables are injected at build time with -ldflags "-X" and use development default values when not injected.
var (
	// Version is the application's semantic version.
	Version = "dev"
	// BuildTime is the build time (UTC, RFC3339).
	BuildTime = "unknown"
	// GitCommit is the commit hash for the build.
	GitCommit = "unknown"
)
