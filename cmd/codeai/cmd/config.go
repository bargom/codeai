package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/bargom/codeai/internal/parser"
	"github.com/bargom/codeai/internal/validator"
	"github.com/spf13/cobra"
)

var (
	// configName is the name for the config
	configName string
	// configDryRun enables dry run mode for config commands
	configDryRun bool
)

// newConfigCmd creates the config command with subcommands.
func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configurations",
		Long: `Manage CodeAI configurations.

Configurations store parsed DSL code and validation results.
Use subcommands to create, list, get, or validate configurations.`,
	}

	// Add subcommands
	cmd.AddCommand(newConfigCreateCmd())
	cmd.AddCommand(newConfigListCmd())
	cmd.AddCommand(newConfigGetCmd())
	cmd.AddCommand(newConfigValidateCmd())

	return cmd
}

// newConfigCreateCmd creates the config create subcommand.
func newConfigCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <file>",
		Short: "Create config from DSL file",
		Long: `Create a new configuration from a CodeAI DSL file.

The file will be parsed, validated, and stored as a configuration.
Use --dry-run to validate without actually creating the config.`,
		Args:    cobra.ExactArgs(1),
		Example: `  codeai config create myfile.cai
  codeai config create --name my-config myfile.cai
  codeai config create --dry-run myfile.cai`,
		RunE: runConfigCreate,
	}

	cmd.Flags().StringVarP(&configName, "name", "n", "", "config name (defaults to filename)")
	cmd.Flags().BoolVar(&configDryRun, "dry-run", false, "validate without creating")
	cmd.Flags().StringVar(&apiURL, "api-url", "http://localhost:8080", "API server URL")

	return cmd
}

func runConfigCreate(cmd *cobra.Command, args []string) error {
	filename := args[0]

	// Parse the file
	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Parsing file: %s\n", filename)
	}

	program, err := parser.ParseFile(filename)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	// Validate
	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Validating...\n")
	}

	v := validator.New()
	if err := v.Validate(program); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	// Set name if not provided
	if configName == "" {
		configName = filename
	}

	if configDryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "Dry run: file is valid\n")
		fmt.Fprintf(cmd.OutOrStdout(), "Config name: %s\n", configName)
		return nil
	}

	// TODO: Call API to create config
	if apiURL == "" {
		return fmt.Errorf("API URL not configured")
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Creating config: %s\n", configName)
	fmt.Fprintf(cmd.OutOrStdout(), "Note: API integration not yet implemented\n")

	return nil
}

// newConfigListCmd creates the config list subcommand.
func newConfigListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configurations",
		Long:  `List all configurations with their names and status.`,
		Example: `  codeai config list
  codeai config list --limit 10
  codeai config list --output json`,
		RunE: runConfigList,
	}

	cmd.Flags().IntVar(&listLimit, "limit", 50, "maximum number of configs to list")
	cmd.Flags().StringVar(&apiURL, "api-url", "http://localhost:8080", "API server URL")

	return cmd
}

func runConfigList(cmd *cobra.Command, args []string) error {
	if apiURL == "" {
		return fmt.Errorf("API URL not configured")
	}

	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Listing configs from %s\n", apiURL)
	}

	// TODO: Call API to list configs
	fmt.Fprintf(cmd.OutOrStdout(), "No configs found (API integration not yet implemented)\n")

	return nil
}

// newConfigGetCmd creates the config get subcommand.
func newConfigGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get config details",
		Long:  `Get detailed information about a specific configuration.`,
		Args:  cobra.ExactArgs(1),
		Example: `  codeai config get abc123
  codeai config get --output json abc123`,
		RunE: runConfigGet,
	}

	cmd.Flags().StringVar(&apiURL, "api-url", "http://localhost:8080", "API server URL")

	return cmd
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	configID := args[0]

	if apiURL == "" {
		return fmt.Errorf("API URL not configured")
	}

	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Getting config %s from %s\n", configID, apiURL)
	}

	// TODO: Call API to get config
	fmt.Fprintf(cmd.OutOrStdout(), "Config %s not found (API integration not yet implemented)\n", configID)

	return nil
}

// newConfigValidateCmd creates the config validate subcommand.
func newConfigValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate <file>",
		Short: "Validate config file",
		Long: `Validate a CodeAI DSL file for syntax and semantic errors.

This is a convenience command that wraps the top-level validate command.`,
		Args:    cobra.ExactArgs(1),
		Example: `  codeai config validate myfile.cai
  codeai config validate --verbose myfile.cai`,
		RunE: runConfigValidate,
	}

	return cmd
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	filename := args[0]

	// Parse the file
	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Validating file: %s\n", filename)
	}

	program, err := parser.ParseFile(filename)
	if err != nil {
		return fmt.Errorf("syntax error: %w", err)
	}

	// Validate semantics
	v := validator.New()
	if err := v.Validate(program); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	// Output result
	result := map[string]interface{}{
		"file":  filename,
		"valid": true,
	}

	switch outputFormat {
	case "json":
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "File is valid: %s\n", filename)
	}

	return nil
}
