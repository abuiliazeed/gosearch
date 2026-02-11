// Package cli provides the command-line interface for gosearch.
//
// It uses Cobra for command parsing and Viper for configuration management.
package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "gosearch",
	Short: "A lightweight web search engine",
	Long: `gosearch is a lightweight web search engine built from scratch in Go.

It crawls web pages, builds a custom inverted index, and provides fast
search capabilities with page ranking, boolean queries, fuzzy matching,
and query caching.`,
}

// Execute runs the root command and returns any error.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringP("config", "c", "", "config file (default is $HOME/.gosearch.yaml)")
	rootCmd.PersistentFlags().StringP("data-dir", "D", "./data", "data directory for index and pages")
	rootCmd.PersistentFlags().CountP("verbose", "v", "verbose output (can be used multiple times)")
	rootCmd.PersistentFlags().String("log-format", "text", "log format: text or json")

	// Bind flags to viper
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("data-dir", rootCmd.PersistentFlags().Lookup("data-dir"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("log-format", rootCmd.PersistentFlags().Lookup("log-format"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if configFile := viper.GetString("config"); configFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(configFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".gosearch" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".gosearch")
	}

	// Read environment variables
	viper.SetEnvPrefix("GOSEARCH")
	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		// Config file found and successfully parsed
	}
}
