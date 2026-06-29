package tft

// Build metadata, injected at link time via -ldflags "-X". See the Makefile's
// LDFLAGS. Defaults keep local `go run` builds readable when no flags are set.
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)
