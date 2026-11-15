// Command synctags syncs tags across security tools, with Qualys as the
// source of truth.
//
//	synctags <vendor> <action> [flags]
//
// Vendors: qualys, crowdstrike. Actions: get, create, sync. create and sync
// write to the tool and need exactly one of --dry-run or --for-real.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/zany2dmax/synctags/internal/cli"
	"github.com/zany2dmax/synctags/internal/crowdstrike"
	"github.com/zany2dmax/synctags/internal/qualys"
	"github.com/zany2dmax/synctags/internal/tags"
	"github.com/zany2dmax/synctags/internal/version"
)

var cfgFile string

func main() {
	cobra.OnInitialize(initConfig)
	root := newRootCmd()
	cmd, err := root.ExecuteC()
	if err == nil {
		return
	}
	var ue *cli.UsageError
	// cobra's "unknown command" at the root is a usage error too.
	if errors.As(err, &ue) || strings.HasPrefix(err.Error(), "unknown command") {
		fmt.Fprintln(os.Stderr, "Error:", err)
		fmt.Fprint(os.Stderr, cmd.UsageString())
		os.Exit(2)
	}
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(1)
}

// vendor describes one tool: how to read its tags and how to connect for
// writes. connect is only called for --for-real, so a dry run makes no
// network call and needs no write credentials.
type vendor struct {
	name       string // CLI word and viper prefix
	title      string // for messages
	defaultYML string
	fetch      func() ([]tags.Tag, error)
	connect    func() (upsert func(name string) error, err error)
}

// newRootCmd builds the command tree. Tests pass fake vendors so no test can
// ever reach a real client; with none given it uses the real ones.
func newRootCmd(vendors ...vendor) *cobra.Command {
	if len(vendors) == 0 {
		vendors = realVendors()
	}
	root := &cobra.Command{
		Use:           "synctags",
		Short:         "Sync tags across multiple integrations",
		SilenceUsage:  true, // main prints usage, and only for usage errors
		SilenceErrors: true,
	}
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return &cli.UsageError{Msg: err.Error()}
	})
	root.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.synctags.yaml)")

	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the build version",
		Run: func(cmd *cobra.Command, _ []string) {
			say(cmd.OutOrStdout(), "%s\n", version.String())
		},
	})

	for _, v := range vendors {
		root.AddCommand(vendorCmd(v))
	}
	return root
}

// realVendors are the tools synctags talks to.
func realVendors() []vendor {
	return []vendor{{
		name:       "qualys",
		title:      "Qualys",
		defaultYML: "qualys_tags.yml",
		fetch: func() ([]tags.Tag, error) {
			qc, err := newQualysClient()
			if err != nil {
				return nil, err
			}
			qtags, err := qc.ListTags()
			if err != nil {
				return nil, fmt.Errorf("listing Qualys tags: %w", err)
			}
			return normalizeQualysTags(qtags), nil
		},
		connect: func() (func(string) error, error) {
			qc, err := newQualysClient()
			if err != nil {
				return nil, err
			}
			return qc.UpsertTag, nil
		},
	}, {
		name:       "crowdstrike",
		title:      "CrowdStrike",
		defaultYML: "crowdstrike_tags.yml",
		fetch: func() ([]tags.Tag, error) {
			cs, err := newCrowdstrikeClient()
			if err != nil {
				return nil, err
			}
			list, err := cs.ListTags()
			if err != nil {
				return nil, fmt.Errorf("listing CrowdStrike tags: %w", err)
			}
			return list, nil
		},
		connect: func() (func(string) error, error) {
			cs, err := newCrowdstrikeClient()
			if err != nil {
				return nil, err
			}
			return cs.UpsertTag, nil
		},
	}}
}

// vendorCmd builds `synctags <vendor> get|create|sync`.
func vendorCmd(v vendor) *cobra.Command {
	parent := &cobra.Command{
		Use:   v.name,
		Short: "Operate on " + v.title + " tags",
		// Without a RunE, a mistyped action ("sycn") prints help and exits 0:
		// a scheduled job would silently do nothing.
		RunE: func(_ *cobra.Command, args []string) error {
			return &cli.UsageError{Msg: fmt.Sprintf("unknown or missing action %q for %s: use get, create or sync", strings.Join(args, " "), v.name)}
		},
	}

	// Paths come from the command's own flag if given, else the config key
	// (<vendor>.output / <vendor>.input), else the default file. Flags are
	// not bound into viper, so each command reads only its own flags.
	outPath := func(cmd *cobra.Command) string { return pathFor(cmd, "output", v.name+".output", v.defaultYML) }
	inPath := func(cmd *cobra.Command) string { return pathFor(cmd, "input", v.name+".input", v.defaultYML) }

	get := &cobra.Command{
		Use:   "get",
		Args:  noArgs,
		Short: "Fetch all " + v.title + " tags and write to YAML (reads the tool, writes a local file)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			list, err := v.fetch()
			if err != nil {
				return err
			}
			out := outPath(cmd)
			if err := tags.WriteYAML(list, out); err != nil {
				return fmt.Errorf("writing YAML: %w", err)
			}
			say(cmd.OutOrStdout(), "Wrote %d tags to %s\n", len(list), out)
			return nil
		},
	}
	get.Flags().StringP("output", "o", "", "YAML output file for "+v.title+" tags")

	create := &cobra.Command{
		Use:   "create",
		Args:  noArgs,
		Short: "Read YAML and create/update tags in " + v.title + " (needs --dry-run or --for-real)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			mode, err := cli.FromFlags(cmd.Flags()) // first: nothing happens without a mode
			if err != nil {
				return err
			}
			in := inPath(cmd)
			list, err := tags.ReadYAML(in)
			if err != nil {
				return fmt.Errorf("reading YAML: %w", err)
			}
			return upsertAll(cmd.OutOrStdout(), v, list, mode)
		},
	}
	create.Flags().StringP("input", "i", "", "YAML input file of "+v.title+" tags")
	cli.AddModeFlags(create.Flags())

	syncCmd := &cobra.Command{
		Use:   "sync",
		Args:  noArgs,
		Short: "get, then create (needs --dry-run or --for-real)",
		Long: "Runs get, then create. With --dry-run the tool is read but nothing is\n" +
			"written: not the YAML file, not the tool.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			mode, err := cli.FromFlags(cmd.Flags()) // first: nothing happens without a mode
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			fetched, err := v.fetch()
			if err != nil {
				return err
			}
			out, in := outPath(cmd), inPath(cmd)
			var list []tags.Tag
			if mode == cli.DryRun {
				say(w, "DRY RUN: would write %d tags to %s\n", len(fetched), out)
				// create would read back the file get just wrote.
				list = fetched
				if in != out {
					if list, err = tags.ReadYAML(in); err != nil {
						return fmt.Errorf("reading YAML: %w", err)
					}
				}
			} else {
				if err := tags.WriteYAML(fetched, out); err != nil {
					return fmt.Errorf("writing YAML: %w", err)
				}
				say(w, "Wrote %d tags to %s\n", len(fetched), out)
				if list, err = tags.ReadYAML(in); err != nil {
					return fmt.Errorf("reading YAML: %w", err)
				}
			}
			return upsertAll(w, v, list, mode)
		},
	}
	syncCmd.Flags().StringP("output", "o", "", "YAML output file for "+v.title+" tags")
	syncCmd.Flags().StringP("input", "i", "", "YAML input file of "+v.title+" tags")
	cli.AddModeFlags(syncCmd.Flags())

	parent.AddCommand(get, create, syncCmd)
	return parent
}

// upsertAll writes each tag to the tool for --for-real, or lists what it
// would write for --dry-run. Any other mode writes nothing and is an error.
func upsertAll(w io.Writer, v vendor, list []tags.Tag, mode cli.Mode) error {
	switch mode {
	case cli.DryRun:
		say(w, "DRY RUN: would create/update %d tags in %s:\n", len(list), v.title)
		for _, t := range list {
			say(w, "  + %s\n", t.Name)
		}
		return nil
	case cli.ForReal:
	default:
		return fmt.Errorf("internal error: refusing to write with mode %v", mode)
	}

	upsert, err := v.connect()
	if err != nil {
		return err
	}
	failed := 0
	for _, t := range list {
		if err := upsert(t.Name); err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "Failed to upsert %s tag %s: %v\n", v.title, t.Name, err)
		}
	}
	say(w, "Processed %d tags in %s (%d failed)\n", len(list), v.title, failed)
	if failed > 0 {
		return fmt.Errorf("%d of %d tags failed", failed, len(list))
	}
	return nil
}

// noArgs refuses positional arguments as a usage error: a stray
// "create --dry-run foo.yml" must not quietly dry-run a different file.
func noArgs(cmd *cobra.Command, args []string) error {
	if err := cobra.NoArgs(cmd, args); err != nil {
		return &cli.UsageError{Msg: err.Error() + " (use -i / -o for files)"}
	}
	return nil
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// pathFor returns the named flag if it was given, else the config value,
// else def.
func pathFor(cmd *cobra.Command, flag, key, def string) string {
	if f := cmd.Flags().Lookup(flag); f != nil && f.Changed {
		return f.Value.String()
	}
	return orDefault(viper.GetString(key), def)
}

// say writes to the command's output. A failed write to stdout is not
// actionable, so its error is dropped on purpose.
func say(w io.Writer, format string, a ...any) {
	_, _ = fmt.Fprintf(w, format, a...)
}

// initConfig reads the config file and environment variables.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		viper.AddConfigPath(home)
		viper.SetConfigName(".synctags")
	}
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func newQualysClient() (*qualys.Client, error) {
	base := viper.GetString("qualys.base_url")
	user := viper.GetString("qualys.username")
	pass := viper.GetString("qualys.password")
	if base == "" || user == "" || pass == "" {
		return nil, errors.New("qualys credentials not set in config or env")
	}
	return qualys.NewClient(base, user, pass), nil
}

func newCrowdstrikeClient() (*crowdstrike.Client, error) {
	base := viper.GetString("crowdstrike.base_url")
	id := viper.GetString("crowdstrike.client_id")
	secret := viper.GetString("crowdstrike.client_secret")
	if base == "" || id == "" || secret == "" {
		return nil, errors.New("crowdstrike credentials not set in config or env")
	}
	cs, err := crowdstrike.NewClient(base, id, secret)
	if err != nil {
		return nil, fmt.Errorf("initializing CrowdStrike client: %w", err)
	}
	return cs, nil
}

// normalizeQualysTags maps Qualys tags to the canonical Tag form.
func normalizeQualysTags(qts []qualys.Tag) []tags.Tag {
	out := make([]tags.Tag, 0, len(qts))
	for _, qt := range qts {
		out = append(out, tags.Tag{ID: qt.ID, Name: tags.Normalize(qt.Name)})
	}
	return out
}
