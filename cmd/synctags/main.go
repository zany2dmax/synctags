// Command synctags syncs tags across security tools, with Qualys as the
// source of truth.
//
//	synctags <vendor> <action> [flags]
//
// Vendors: qualys, crowdstrike. Actions: get, create, sync.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/zany2dmax/synctags/internal/crowdstrike"
	"github.com/zany2dmax/synctags/internal/qualys"
	"github.com/zany2dmax/synctags/internal/tags"
	"github.com/zany2dmax/synctags/internal/version"
)

var cfgFile string

func main() {
	cobra.OnInitialize(initConfig)

	rootCmd := &cobra.Command{
		Use:   "synctags",
		Short: "Sync tags across multiple integrations",
	}
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.synctags.yaml)")

	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the build version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.String())
		},
	})

	// --------------------- Qualys Commands ---------------------
	qualysCmd := &cobra.Command{
		Use:   "qualys",
		Short: "Operate on Qualys tags",
	}

	qualysGet := &cobra.Command{
		Use:   "get",
		Short: "Fetch all Qualys tags and write to YAML",
		Run: func(cmd *cobra.Command, args []string) {
			qc := newQualysClient()
			qtags, err := qc.ListTags()
			if err != nil {
				fail("Error listing Qualys tags: %v", err)
			}
			norm := normalizeQualysTags(qtags)
			out := viper.GetString("qualys.output")
			if out == "" {
				out = "qualys_tags.yml"
			}
			if err := tags.WriteYAML(norm, out); err != nil {
				fail("Error writing YAML: %v", err)
			}
			fmt.Printf("Wrote %d tags to %s\n", len(norm), out)
		},
	}
	qualysGet.Flags().StringP("output", "o", "", "YAML output file for Qualys tags")
	bindFlag("qualys.output", qualysGet.Flags().Lookup("output"))

	qualysCreate := &cobra.Command{
		Use:   "create",
		Short: "Read YAML and create/update tags in Qualys",
		Run: func(cmd *cobra.Command, args []string) {
			in := viper.GetString("qualys.input")
			if in == "" {
				in = "qualys_tags.yml"
			}
			list, err := tags.ReadYAML(in)
			if err != nil {
				fail("Error reading YAML: %v", err)
			}
			qc := newQualysClient()
			for _, t := range list {
				if err := qc.UpsertTag(t.Name); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to upsert Qualys tag %s: %v\n", t.Name, err)
				}
			}
			fmt.Printf("Processed %d tags in Qualys\n", len(list))
		},
	}
	qualysCreate.Flags().StringP("input", "i", "", "YAML input file of Qualys tags")
	bindFlag("qualys.input", qualysCreate.Flags().Lookup("input"))

	qualysSync := &cobra.Command{
		Use:   "sync",
		Short: "Reconcile Qualys tags with YAML source",
		Run: func(cmd *cobra.Command, args []string) {
			qualysGet.Run(cmd, args)
			qualysCreate.Run(cmd, args)
		},
	}

	qualysCmd.AddCommand(qualysGet, qualysCreate, qualysSync)
	rootCmd.AddCommand(qualysCmd)

	// ----------------- CrowdStrike Commands --------------------
	csCmd := &cobra.Command{
		Use:   "crowdstrike",
		Short: "Operate on CrowdStrike tags",
	}

	csGet := &cobra.Command{
		Use:   "get",
		Short: "Fetch all CrowdStrike tags and write to YAML",
		Run: func(cmd *cobra.Command, args []string) {
			cs := newCrowdstrikeClient()
			list, err := cs.ListTags()
			if err != nil {
				fail("Error listing CrowdStrike tags: %v", err)
			}
			out := viper.GetString("crowdstrike.output")
			if out == "" {
				out = "crowdstrike_tags.yml"
			}
			if err := tags.WriteYAML(list, out); err != nil {
				fail("Error writing YAML: %v", err)
			}
			fmt.Printf("Wrote %d tags to %s\n", len(list), out)
		},
	}
	csGet.Flags().StringP("output", "o", "", "YAML output file for CrowdStrike tags")
	bindFlag("crowdstrike.output", csGet.Flags().Lookup("output"))

	csCreate := &cobra.Command{
		Use:   "create",
		Short: "Read YAML and create/update tags in CrowdStrike",
		Run: func(cmd *cobra.Command, args []string) {
			in := viper.GetString("crowdstrike.input")
			if in == "" {
				in = "crowdstrike_tags.yml"
			}
			list, err := tags.ReadYAML(in)
			if err != nil {
				fail("Error reading YAML: %v", err)
			}
			cs := newCrowdstrikeClient()
			for _, t := range list {
				if err := cs.UpsertTag(t.Name); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to upsert CrowdStrike tag %s: %v\n", t.Name, err)
				}
			}
			fmt.Printf("Processed %d tags in CrowdStrike\n", len(list))
		},
	}
	csCreate.Flags().StringP("input", "i", "", "YAML input file of CrowdStrike tags")
	bindFlag("crowdstrike.input", csCreate.Flags().Lookup("input"))

	csSync := &cobra.Command{
		Use:   "sync",
		Short: "Reconcile CrowdStrike tags with YAML source",
		Run: func(cmd *cobra.Command, args []string) {
			csGet.Run(cmd, args)
			csCreate.Run(cmd, args)
		},
	}

	csCmd.AddCommand(csGet, csCreate, csSync)
	rootCmd.AddCommand(csCmd)

	if err := rootCmd.Execute(); err != nil {
		fail("%v", err)
	}
}

// fail prints to stderr and exits 1.
func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}

// bindFlag binds a flag to a viper key; a failure is a programming error.
func bindFlag(key string, f *pflag.Flag) {
	if err := viper.BindPFlag(key, f); err != nil {
		fail("internal error binding --%s: %v", key, err)
	}
}

// initConfig reads the config file and environment variables.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fail("%v", err)
		}
		viper.AddConfigPath(home)
		viper.SetConfigName(".synctags")
	}
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func newQualysClient() *qualys.Client {
	base := viper.GetString("qualys.base_url")
	user := viper.GetString("qualys.username")
	pass := viper.GetString("qualys.password")
	if base == "" || user == "" || pass == "" {
		fail("Qualys credentials not set in config or env")
	}
	return qualys.NewClient(base, user, pass)
}

func newCrowdstrikeClient() *crowdstrike.Client {
	base := viper.GetString("crowdstrike.base_url")
	id := viper.GetString("crowdstrike.client_id")
	secret := viper.GetString("crowdstrike.client_secret")
	if base == "" || id == "" || secret == "" {
		fail("CrowdStrike credentials not set in config or env")
	}
	cs, err := crowdstrike.NewClient(base, id, secret)
	if err != nil {
		fail("Error initializing CrowdStrike client: %v", err)
	}
	return cs
}

// normalizeQualysTags maps Qualys tags to the canonical Tag form.
func normalizeQualysTags(qts []qualys.Tag) []tags.Tag {
	out := make([]tags.Tag, 0, len(qts))
	for _, qt := range qts {
		out = append(out, tags.Tag{ID: qt.ID, Name: tags.Normalize(qt.Name)})
	}
	return out
}
