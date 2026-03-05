// Package cli provides the command-line interface for gosearch.
//
// It uses Cobra for command parsing and Viper for configuration management.
package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Version info set by main
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// SetVersion sets the version info from main (injected by goreleaser)
func SetVersion(v, c, d string) {
	version = v
	commit = c
	date = d
}

var rootCmd = &cobra.Command{
	Use:   "gosearch",
	Short: "A lightweight web search engine",
	Long: `gosearch is a lightweight web search engine built from scratch in Go.

It crawls web pages, stores cleaned markdown content, builds a custom
inverted index, and provides fast search capabilities with ranking,
boolean queries, fuzzy matching, and query caching.

Storage schema:
  - Current schema: 2-markdown-only
  - First crawl after upgrade automatically resets legacy corpus data
    and initializes the new schema marker.`,
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
	_ = viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	_ = viper.BindPFlag("data-dir", rootCmd.PersistentFlags().Lookup("data-dir"))
	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	_ = viper.BindPFlag("log-format", rootCmd.PersistentFlags().Lookup("log-format"))

	// Set default Redis configuration
	viper.SetDefault("redis.host", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.ttl", "24h")

	rootCmd.SetHelpCommand(newComprehensiveHelpCommand())
}

func newComprehensiveHelpCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "help [command]",
		Short: "Help about any command",
		Long:  "Help provides a comprehensive overview of all commands and supports command-specific help.",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			if len(args) == 0 {
				renderComprehensiveRootHelp(root.OutOrStdout(), root)
				return nil
			}

			target, _, err := root.Find(args)
			if err != nil {
				return err
			}
			if target == nil {
				return fmt.Errorf("unknown help topic: %s", strings.Join(args, " "))
			}
			if target == cmd {
				// Avoid infinite recursion on "help help".
				return cmd.Help()
			}
			return target.Help()
		},
	}
}

func renderComprehensiveRootHelp(w io.Writer, root *cobra.Command) {
	_, _ = fmt.Fprintln(w, root.Long)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "Usage:\n  %s [command]\n\n", root.CommandPath())

	_, _ = fmt.Fprintln(w, "Command Tree:")
	printCommandTree(w, root, "")
	_, _ = fmt.Fprintln(w)

	_, _ = fmt.Fprintln(w, "All Commands:")
	all := collectVisibleCommands(root)
	for _, cmd := range all {
		_, _ = fmt.Fprintf(w, "  %-28s %s\n", cmd.CommandPath(), cmd.Short)
	}
	_, _ = fmt.Fprintln(w)

	_, _ = fmt.Fprintln(w, "Global Flags:")
	printFlags(w, root.PersistentFlags(), "  ")

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Command Flags:")
	for _, cmd := range all {
		if !cmd.LocalFlags().HasAvailableFlags() {
			continue
		}
		_, _ = fmt.Fprintf(w, "  %s\n", cmd.CommandPath())
		printFlags(w, cmd.LocalFlags(), "    ")
	}

	_, _ = fmt.Fprintf(w, "\nUse \"%s <command> --help\" for command-specific details.\n", root.CommandPath())
}

func printCommandTree(w io.Writer, cmd *cobra.Command, indent string) {
	children := visibleSubcommands(cmd)
	if cmd.Parent() == nil {
		_, _ = fmt.Fprintf(w, "  %s\n", cmd.CommandPath())
	} else {
		_, _ = fmt.Fprintf(w, "%s- %s: %s\n", indent, cmd.Name(), cmd.Short)
	}
	for _, child := range children {
		nextIndent := indent + "  "
		if cmd.Parent() == nil {
			nextIndent = "    "
		}
		printCommandTree(w, child, nextIndent)
	}
}

func collectVisibleCommands(root *cobra.Command) []*cobra.Command {
	var out []*cobra.Command
	var walk func(cmd *cobra.Command)
	walk = func(cmd *cobra.Command) {
		for _, child := range visibleSubcommands(cmd) {
			out = append(out, child)
			walk(child)
		}
	}
	walk(root)
	return out
}

func visibleSubcommands(cmd *cobra.Command) []*cobra.Command {
	children := make([]*cobra.Command, 0, len(cmd.Commands()))
	for _, child := range cmd.Commands() {
		if child.Hidden {
			continue
		}
		children = append(children, child)
	}
	return children
}

func printFlags(w io.Writer, flags *pflag.FlagSet, indent string) {
	if !flags.HasAvailableFlags() {
		_, _ = fmt.Fprintf(w, "%s(none)\n", indent)
		return
	}
	flags.VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		name := "--" + f.Name
		if f.Shorthand != "" {
			name = "-" + f.Shorthand + ", " + name
		}
		if f.DefValue != "" && f.DefValue != "false" {
			_, _ = fmt.Fprintf(w, "%s%-24s %s (default %q)\n", indent, name, f.Usage, f.DefValue)
			return
		}
		_, _ = fmt.Fprintf(w, "%s%-24s %s\n", indent, name, f.Usage)
	})
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
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	// If a config file is found, read it in.
	_ = viper.ReadInConfig() // Config file is optional
}
