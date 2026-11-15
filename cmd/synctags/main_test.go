package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/zany2dmax/synctags/internal/cli"
	"github.com/zany2dmax/synctags/internal/tags"
)

// fake is a vendor that records what it is asked to do. Tests only ever use
// fakes, so no test can reach a real tool whatever credentials are around.
type fake struct {
	fetched   int
	connected int
	upserted  []string
}

func (f *fake) vendor(name string) vendor {
	return vendor{
		name:       name,
		title:      name,
		defaultYML: name + "_tags.yml",
		fetch: func() ([]tags.Tag, error) {
			f.fetched++
			return []tags.Tag{{ID: 1, Name: "fetched-tag"}}, nil
		},
		connect: func() (func(string) error, error) {
			f.connected++
			return func(n string) error {
				f.upserted = append(f.upserted, n)
				return nil
			}, nil
		},
	}
}

func run(t *testing.T, f *fake, args ...string) (string, error) {
	t.Helper()
	root := newRootCmd(f.vendor("qualys"), f.vendor("crowdstrike"))
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	_, err := root.ExecuteC()
	return out.String(), err
}

func isUsage(err error) bool {
	var ue *cli.UsageError
	return errors.As(err, &ue)
}

func (f *fake) touched() bool { return f.fetched+f.connected+len(f.upserted) > 0 }

func TestWriteCommandsRefuseWithoutAMode(t *testing.T) {
	for _, args := range [][]string{
		{"qualys", "create", "-i", "testdata/tags.yml"},
		{"qualys", "sync"},
		{"crowdstrike", "create", "-i", "testdata/tags.yml"},
		{"crowdstrike", "sync"},
	} {
		f := &fake{}
		if _, err := run(t, f, args...); !isUsage(err) {
			t.Errorf("%v: want a usage error, got %v", args, err)
		}
		if f.touched() {
			t.Errorf("%v: refused command still touched the tool: %+v", args, f)
		}
	}
}

func TestWriteCommandsRefuseBothModes(t *testing.T) {
	f := &fake{}
	_, err := run(t, f, "qualys", "create", "--dry-run", "--for-real", "-i", "testdata/tags.yml")
	if !isUsage(err) || f.touched() {
		t.Fatalf("want a usage error and nothing touched, got %v, %+v", err, f)
	}
}

func TestDryRunCreateListsTagsAndNeverConnects(t *testing.T) {
	f := &fake{}
	out, err := run(t, f, "qualys", "create", "--dry-run", "-i", "testdata/tags.yml")
	if err != nil {
		t.Fatal(err)
	}
	if f.touched() {
		t.Fatalf("dry run touched the tool: %+v", f)
	}
	for _, want := range []string{"DRY RUN", "+ example-site-a", "+ example-business-unit"} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
}

func TestForRealCreateUpsertsEveryTag(t *testing.T) {
	f := &fake{}
	if _, err := run(t, f, "qualys", "create", "--for-real", "-i", "testdata/tags.yml"); err != nil {
		t.Fatal(err)
	}
	want := []string{"example-site-a", "example-business-unit"}
	if f.connected != 1 || strings.Join(f.upserted, ",") != strings.Join(want, ",") {
		t.Fatalf("want one connect and upserts %v, got %+v", want, f)
	}
}

func TestDryRunSyncReadsButWritesNothing(t *testing.T) {
	f := &fake{}
	out := filepath.Join(t.TempDir(), "out.yml")
	stdout, err := run(t, f, "qualys", "sync", "--dry-run", "-o", out, "-i", out)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(out); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry-run sync wrote %s (stat err %v)", out, err)
	}
	if f.fetched != 1 || f.connected != 0 || len(f.upserted) != 0 {
		t.Fatalf("dry-run sync should read once and write nothing: %+v", f)
	}
	if !strings.Contains(stdout, "DRY RUN") || !strings.Contains(stdout, "+ fetched-tag") {
		t.Fatalf("output:\n%s", stdout)
	}
}

func TestForRealSyncWritesFileAndUpserts(t *testing.T) {
	f := &fake{}
	out := filepath.Join(t.TempDir(), "out.yml")
	if _, err := run(t, f, "qualys", "sync", "--for-real", "-o", out, "-i", out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("for-real sync did not write %s: %v", out, err)
	}
	if f.connected != 1 || strings.Join(f.upserted, ",") != "fetched-tag" {
		t.Fatalf("got %+v", f)
	}
}

func TestBadInvocationsAreUsageErrors(t *testing.T) {
	for _, args := range [][]string{
		{"qualys", "create", "--for-reel"}, // unknown flag
		{"qualys", "sycn"},                 // mistyped action
		{"qualys"},                         // no action
		{"qualys", "create", "--dry-run", "testdata/tags.yml"},     // stray positional
		{"crowdstrike", "sync", "--for-real", "testdata/tags.yml"}, // stray positional
	} {
		f := &fake{}
		if _, err := run(t, f, args...); !isUsage(err) {
			t.Errorf("%v: want a usage error, got %v", args, err)
		}
		if f.touched() {
			t.Errorf("%v: touched the tool: %+v", args, f)
		}
	}
}

func TestPathForPrefersTheFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "x"}
	cmd.Flags().String("output", "", "")
	if got := pathFor(cmd, "output", "test.unset.key", "default.yml"); got != "default.yml" {
		t.Fatalf("no flag, no config: got %q", got)
	}
	if err := cmd.Flags().Set("output", "mine.yml"); err != nil {
		t.Fatal(err)
	}
	if got := pathFor(cmd, "output", "test.unset.key", "default.yml"); got != "mine.yml" {
		t.Fatalf("flag given: got %q", got)
	}
}
