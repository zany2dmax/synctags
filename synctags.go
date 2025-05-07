/*
Directory structure:

├── synctags.go                // CLI entrypoint (package main)
├── tags.go                    // Tag type & YAML helpers
├── qualys_client.go           // Qualys API client
├── crowdstrike_client.go      // CrowdStrike API client
*/

// synctags.go
package main

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var cfgFile string

func main() {
    cobra.OnInitialize(initConfig)

    rootCmd := &cobra.Command{
        Use:   "synctags",
        Short: "Sync tags across multiple integrations",
    }

    // Global config flag
    rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.synctags.yaml)")

    // --------------------- Qualys Commands ---------------------
    qualysCmd := &cobra.Command{
        Use:   "qualys",
        Short: "Operate on Qualys tags",
    }

    // qualys get
    qualysGet := &cobra.Command{
        Use:   "get",
        Short: "Fetch all Qualys tags and write to YAML",
        Run: func(cmd *cobra.Command, args []string) {
            qc := newQualysClient()
            qtags, err := qc.ListTags()
            if err != nil {
                fmt.Fprintf(os.Stderr, "Error listing Qualys tags: %v\n", err)
                os.Exit(1)
            }
            // Normalize
            norm := normalizeQualysTags(qtags)
            out := viper.GetString("qualys.output")
            if out == "" {
                out = "qualys_tags.yml"
            }
            if err := WriteTagsToYAML(norm, out); err != nil {
                fmt.Fprintf(os.Stderr, "Error writing YAML: %v\n", err)
                os.Exit(1)
            }
            fmt.Printf("Wrote %d tags to %s\n", len(norm), out)
        },
    }
    qualysGet.Flags().StringP("output", "o", "", "YAML output file for Qualys tags")
    viper.BindPFlag("qualys.output", qualysGet.Flags().Lookup("output"))

    // qualys create
    qualysCreate := &cobra.Command{
        Use:   "create",
        Short: "Read YAML and create/update tags in Qualys",
        Run: func(cmd *cobra.Command, args []string) {
            in := viper.GetString("qualys.input")
            if in == "" {
                in = "qualys_tags.yml"
            }
            tags, err := ReadTagsFromYAML(in)
            if err != nil {
                fmt.Fprintf(os.Stderr, "Error reading YAML: %v\n", err)
                os.Exit(1)
            }
            qc := newQualysClient()
            for _, t := range tags {
                if err := qc.UpsertTag(t); err != nil {
                    fmt.Fprintf(os.Stderr, "Failed to upsert Qualys tag %s: %v\n", t.Name, err)
                }
            }
            fmt.Printf("Processed %d tags in Qualys\n", len(tags))
        },
    }
    qualysCreate.Flags().StringP("input", "i", "", "YAML input file of Qualys tags")
    viper.BindPFlag("qualys.input", qualysCreate.Flags().Lookup("input"))

    // qualys sync (get + create)
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

    // crowdstrike get
    csGet := &cobra.Command{
        Use:   "get",
        Short: "Fetch all CrowdStrike tags and write to YAML",
        Run: func(cmd *cobra.Command, args []string) {
            cs := newCrowdstrikeClient()
            tags, err := cs.ListTags()
            if err != nil {
                fmt.Fprintf(os.Stderr, "Error listing CrowdStrike tags: %v\n", err)
                os.Exit(1)
            }
            out := viper.GetString("crowdstrike.output")
            if out == "" {
                out = "crowdstrike_tags.yml"
            }
            if err := WriteTagsToYAML(tags, out); err != nil {
                fmt.Fprintf(os.Stderr, "Error writing YAML: %v\n", err)
                os.Exit(1)
            }
            fmt.Printf("Wrote %d tags to %s\n", len(tags), out)
        },
    }
    csGet.Flags().StringP("output", "o", "", "YAML output file for CrowdStrike tags")
    viper.BindPFlag("crowdstrike.output", csGet.Flags().Lookup("output"))

    // crowdstrike create
    csCreate := &cobra.Command{
        Use:   "create",
        Short: "Read YAML and create/update tags in CrowdStrike",
        Run: func(cmd *cobra.Command, args []string) {
            in := viper.GetString("crowdstrike.input")
            if in == "" {
                in = "crowdstrike_tags.yml"
            }
            tags, err := ReadTagsFromYAML(in)
            if err != nil {
                fmt.Fprintf(os.Stderr, "Error reading YAML: %v\n", err)
                os.Exit(1)
            }
            cs := newCrowdstrikeClient()
            for _, t := range tags {
                if err := cs.UpsertTag(t); err != nil {
                    fmt.Fprintf(os.Stderr, "Failed to upsert CrowdStrike tag %s: %v\n", t.Name, err)
                }
            }
            fmt.Printf("Processed %d tags in CrowdStrike\n", len(tags))
        },
    }
    csCreate.Flags().StringP("input", "i", "", "YAML input file of CrowdStrike tags")
    viper.BindPFlag("crowdstrike.input", csCreate.Flags().Lookup("input"))

    // crowdstrike sync
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

    // Execute
    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}

// initConfig reads config file and ENV variables
func initConfig() {
    if cfgFile != "" {
        viper.SetConfigFile(cfgFile)
    } else {
        home, err := os.UserHomeDir()
        if err != nil {
            fmt.Fprintln(os.Stderr, err)
            os.Exit(1)
        }
        viper.AddConfigPath(home)
        viper.SetConfigName(".synctags")
    }
    viper.AutomaticEnv()
    if err := viper.ReadInConfig(); err == nil {
        fmt.Println("Using config file:", viper.ConfigFileUsed())
    }
}

// newQualysClient initializes Qualys client
func newQualysClient() *Client {
    base := viper.GetString("qualys.base_url")
    user := viper.GetString("qualys.username")
    pass := viper.GetString("qualys.password")
    if base == "" || user == "" || pass == "" {
        fmt.Fprintln(os.Stderr, "Qualys credentials not set in config or env")
        os.Exit(1)
    }
    return NewQualysClient(base, user, pass)
}

// newCrowdstrikeClient initializes CrowdStrike client
func newCrowdstrikeClient() *CrowdstrikeClient {
    base := viper.GetString("crowdstrike.base_url")
    id := viper.GetString("crowdstrike.client_id")
    secret := viper.GetString("crowdstrike.client_secret")
    if base == "" || id == "" || secret == "" {
        fmt.Fprintln(os.Stderr, "CrowdStrike credentials not set in config or env")
        os.Exit(1)
    }
    cs, err := NewCrowdstrikeClient(base, id, secret)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error initializing CrowdStrike client: %v\n", err)
        os.Exit(1)
    }
    return cs
}

// Helper to normalize QualysTag list (maps to Tag slice)
func normalizeQualysTags(qts []QualysTag) []Tag {
    var out []Tag
    for _, qt := range qts {
        out = append(out, NormalizeQualysTag(qt))
    }
    return out
}

