// Package cmd provides the CLI commands for CodeAI.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// cfgFile holds the path to the config file
	cfgFile string
	// verbose enables verbose output
	verbose bool
	// outputFormat specifies the output format (json, table, plain)
	outputFormat string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "codeai",
	Short: "CodeAI DSL runtime and deployment tool",
	Long: `CodeAI is a command-line tool for parsing, validating, and deploying
AI agent specifications written in a domain-specific language.

It provides a structured approach to defining and running AI agents
using a Markdown-based DSL.`,
	SilenceUsage: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

// GetRootCmd returns the root command for testing purposes.
func GetRootCmd() *cobra.Command {
	return rootCmd
}

// NewRootCmd creates a new root command for testing.
// This allows tests to create fresh command trees.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "codeai",
		Short:        "CodeAI DSL runtime and deployment tool",
		Long:         rootCmd.Long,
		SilenceUsage: true,
	}

	// Add persistent flags
	cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cai.yaml)")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	cmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "plain", "output format (json|table|plain)")

	// Add all subcommands
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newParseCmd())
	cmd.AddCommand(newValidateCmd())
	cmd.AddCommand(newDeployCmd())
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newServerCmd())
	cmd.AddCommand(newCompletionCmd())

	return cmd
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cai.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "plain", "output format (json|table|plain)")

	// Add all subcommands
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newParseCmd())
	rootCmd.AddCommand(newValidateCmd())
	rootCmd.AddCommand(newDeployCmd())
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newServerCmd())
	rootCmd.AddCommand(newCompletionCmd())
}

// isVerbose returns true if verbose mode is enabled.
func isVerbose() bool {
	return verbose
}

// getOutputFormat returns the current output format.
func getOutputFormat() string {
	return outputFormat
}

// printVerbose prints message only if verbose mode is enabled.
func printVerbose(cmd *cobra.Command, format string, args ...interface{}) {
	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), format, args...)
	}
}

// printError prints an error message to stderr.
func printError(cmd *cobra.Command, format string, args ...interface{}) {
	fmt.Fprintf(cmd.ErrOrStderr(), "Error: "+format+"\n", args...)
}

// exitWithError prints an error and exits with code 1.
func exitWithError(cmd *cobra.Command, err error) {
	printError(cmd, "%v", err)
	os.Exit(1)
}
