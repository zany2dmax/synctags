package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
)

func main() {
	cobra.OnInitialize(initConfig)

	rootCmd := &cobra.Command{
		Use:   "synctags",
		Short: "Sync tags across multiple solutions",
	}

	// persistent flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.synctags.yaml)")

	// Qualys subcommand
	qualysCmd := &cobra.Command{
		Use:   "qualys",
		Short: "Operate on Qualys tags",
	}

	// Qualys get
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Fetch all Qualys tags and write to YAML",
		Run: func(cmd *cobra.Command, args []string) {
			client := newQualysClient()
			tags, err := client.ListTags()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error listing Qualys tags: %v\n", err)
				os.Exit(1)
			}
			var norm []Tag
			for _, qt := range tags {
				norm = append(norm, NormalizeQualysTag(qt))
			}
			out := viper.GetString("output")
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
	getCmd.Flags().StringP("output", "o", "", "YAML output file")
	viper.BindPFlag("output", getCmd.Flags().Lookup("output"))

	// Qualys create
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Read YAML and create/update tags in Qualys",
		Run: func(cmd *cobra.Command, args []string) {
			yml := viper.GetString("input")
			if yml == "" {
				yml = "qualys_tags.yml"
			}
			tags, err := ReadTagsFromYAML(yml)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading YAML: %v\n", err)
				os.Exit(1)
			}
			client := newQualysClient()
			for _, t := range tags {
				if err := client.UpsertTag(t); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to upsert tag %s: %v\n", t.Name, err)
				}
			}
			fmt.Printf("Processed %d tags\n", len(tags))
		},
	}
	createCmd.Flags().StringP("input", "i", "", "YAML input file of tags")
	viper.BindPFlag("input", createCmd.Flags().Lookup("input"))

	// Qualys sync: get + create
	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Fetch from Qualys and reconcile with YAML source",
		Run: func(cmd *cobra.Command, args []string) {
			// for now, simple: run get then create
			getCmd.Run(cmd, args)
			createCmd.Run(cmd, args)
		},
	}

	qualysCmd.AddCommand(getCmd, createCmd, syncCmd)
	rootCmd.AddCommand(qualysCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// initConfig reads in config file and ENV variables
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

// newQualysClient initializes client from env or config
func newQualysClient() *Client {
	baseURL := viper.GetString("qualys.base_url")
	user := viper.GetString("qualys.username")
	pass := viper.GetString("qualys.password")
	if baseURL == "" || user == "" || pass == "" {
		fmt.Fprintln(os.Stderr, "Qualys credentials not set in config or env")
		os.Exit(1)
	}
	return NewClient(baseURL, user, pass)
}
