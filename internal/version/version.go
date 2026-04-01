// Package version provides build version information set via ldflags.
package version

// Version, Commit, and BuildDate are set at build time via -ldflags.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// String returns the short version string.
func String() string {
	return Version
}

// Full returns the full version string including commit and build date.
func Full() string {
	return Version + " (" + Commit + ") " + BuildDate
}
