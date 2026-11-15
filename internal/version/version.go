// Package version carries build information stamped in by `task build`
// with -ldflags -X. A plain `go build` leaves the defaults.
package version

// Set at build time.
var (
	Tag    = "unknown"
	Commit = "unknown"
	Built  = "unknown"
)

// String is the one-line form printed by `synctags version`.
func String() string {
	return "synctags " + Tag + " (commit " + Commit + ", built " + Built + ")"
}
