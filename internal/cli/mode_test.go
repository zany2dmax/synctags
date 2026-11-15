package cli

import (
	"errors"
	"testing"

	"github.com/spf13/pflag"
)

func TestResolve(t *testing.T) {
	cases := []struct {
		name    string
		dry     bool
		forReal bool
		want    Mode
		wantErr bool
	}{
		{"neither is refused", false, false, 0, true},
		{"both is refused", true, true, 0, true},
		{"dry run", true, false, DryRun, false},
		{"for real", false, true, ForReal, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Resolve(c.dry, c.forReal)
			if c.wantErr {
				var ue *UsageError
				if !errors.As(err, &ue) {
					t.Fatalf("want a UsageError, got %v", err)
				}
				if got != 0 {
					t.Fatalf("a refused mode must be the zero Mode, got %v", got)
				}
				return
			}
			if err != nil || got != c.want {
				t.Fatalf("got %v, %v; want %v", got, err, c.want)
			}
		})
	}
}

func TestFromFlags(t *testing.T) {
	fs := pflag.NewFlagSet("x", pflag.ContinueOnError)
	AddModeFlags(fs)
	if _, err := FromFlags(fs); err == nil {
		t.Fatal("no flags given: want an error")
	}
	if err := fs.Parse([]string{"--dry-run"}); err != nil {
		t.Fatal(err)
	}
	if m, err := FromFlags(fs); err != nil || m != DryRun {
		t.Fatalf("got %v, %v; want dry-run", m, err)
	}
}

func TestFromFlagsWithoutRegisteredFlagsIsRefused(t *testing.T) {
	fs := pflag.NewFlagSet("x", pflag.ContinueOnError)
	if _, err := FromFlags(fs); err == nil {
		t.Fatal("unregistered flags must not resolve to a mode")
	}
}
