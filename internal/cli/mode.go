// Package cli holds the run-mode rule shared by every command that writes to
// a tool: it must be told exactly one of --dry-run or --for-real. There is no
// default in either direction, so a forgotten flag never writes and a
// scheduled job never silently does nothing.
package cli

import (
	"github.com/spf13/pflag"
)

// Mode says whether a command may write to a tool.
type Mode int

// The zero Mode is deliberately not a valid mode.
const (
	DryRun Mode = iota + 1
	ForReal
)

func (m Mode) String() string {
	switch m {
	case DryRun:
		return "dry-run"
	case ForReal:
		return "for-real"
	}
	return "unset"
}

// UsageError is a mistake in how the command was invoked. main prints the
// command's usage and exits 2 on it, before anything is touched.
type UsageError struct{ Msg string }

func (e *UsageError) Error() string { return e.Msg }

// Resolve turns the two flags into a Mode. Neither or both is a UsageError.
func Resolve(dryRun, forReal bool) (Mode, error) {
	switch {
	case dryRun && forReal:
		return 0, &UsageError{Msg: "--dry-run and --for-real cannot be used together; give exactly one"}
	case dryRun:
		return DryRun, nil
	case forReal:
		return ForReal, nil
	}
	return 0, &UsageError{Msg: "this command writes to a tool: give exactly one of --dry-run or --for-real"}
}

// AddModeFlags registers --dry-run and --for-real on a command's flags.
func AddModeFlags(fs *pflag.FlagSet) {
	fs.Bool("dry-run", false, "show what would be written, write nothing")
	fs.Bool("for-real", false, "write the changes to the tool")
}

// FromFlags reads the flags registered by AddModeFlags and resolves them.
func FromFlags(fs *pflag.FlagSet) (Mode, error) {
	dry, err := fs.GetBool("dry-run")
	if err != nil {
		return 0, &UsageError{Msg: "internal error: --dry-run not registered: " + err.Error()}
	}
	forReal, err := fs.GetBool("for-real")
	if err != nil {
		return 0, &UsageError{Msg: "internal error: --for-real not registered: " + err.Error()}
	}
	return Resolve(dry, forReal)
}
