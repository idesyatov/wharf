package version

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func String() string {
	return Version
}

func Full() string {
	return Version + " (" + Commit + ") " + BuildDate
}
