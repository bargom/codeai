package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/bargom/codeai/internal/parser"
	"github.com/bargom/codeai/internal/validator"
	"github.com/spf13/cobra"
)

var (
	// deployName is the name for the deployment
	deployName string
	// deployDryRun enables dry run mode
	deployDryRun bool
	// deployForce enables force deletion
	deployForce bool
	// deployWait waits for execution to complete
	deployWait bool
	// apiURL is the API server URL
	apiURL string
	// listLimit is the maximum number of items to list
	listLimit int
)

// newDeployCmd creates the deploy command with subcommands.
func newDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Manage deployments",
		Long: `Manage CodeAI deployments.

Deployments are running instances of your CodeAI configurations.
Use subcommands to create, list, get, delete, or execute deployments.`,
	}

	// Add subcommands
	cmd.AddCommand(newDeployCreateCmd())
	cmd.AddCommand(newDeployListCmd())
	cmd.AddCommand(newDeployGetCmd())
	cmd.AddCommand(newDeployDeleteCmd())
	cmd.AddCommand(newDeployExecuteCmd())

	return cmd
}

// newDeployCreateCmd creates the deploy create subcommand.
func newDeployCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <file>",
		Short: "Create deployment from config file",
		Long: `Create a new deployment from a CodeAI DSL configuration file.

The file will be parsed and validated before creating the deployment.
Use --dry-run to validate without actually creating the deployment.`,
		Args:    cobra.ExactArgs(1),
		Example: `  codeai deploy create myconfig.cai
  codeai deploy create --name my-deploy myconfig.cai
  codeai deploy create --dry-run myconfig.cai`,
		RunE: runDeployCreate,
	}

	cmd.Flags().StringVarP(&deployName, "name", "n", "", "deployment name (defaults to filename)")
	cmd.Flags().BoolVar(&deployDryRun, "dry-run", false, "validate without creating")
	cmd.Flags().StringVar(&apiURL, "api-url", "http://localhost:8080", "API server URL")

	return cmd
}

func runDeployCreate(cmd *cobra.Command, args []string) error {
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
	if deployName == "" {
		deployName = filename
	}

	if deployDryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "Dry run: file is valid\n")
		fmt.Fprintf(cmd.OutOrStdout(), "Deployment name: %s\n", deployName)
		return nil
	}

	// TODO: Call API to create deployment
	// For now, just show what would be created
	if apiURL == "" {
		return fmt.Errorf("API URL not configured")
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Creating deployment: %s\n", deployName)
	fmt.Fprintf(cmd.OutOrStdout(), "Note: API integration not yet implemented\n")

	return nil
}

// newDeployListCmd creates the deploy list subcommand.
func newDeployListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List deployments",
		Long:  `List all deployments with their status and basic information.`,
		Example: `  codeai deploy list
  codeai deploy list --limit 10
  codeai deploy list --output json`,
		RunE: runDeployList,
	}

	cmd.Flags().IntVar(&listLimit, "limit", 50, "maximum number of deployments to list")
	cmd.Flags().StringVar(&apiURL, "api-url", "http://localhost:8080", "API server URL")

	return cmd
}

func runDeployList(cmd *cobra.Command, args []string) error {
	if apiURL == "" {
		return fmt.Errorf("API URL not configured")
	}

	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Listing deployments from %s\n", apiURL)
	}

	// TODO: Call API to list deployments
	fmt.Fprintf(cmd.OutOrStdout(), "No deployments found (API integration not yet implemented)\n")

	return nil
}

// newDeployGetCmd creates the deploy get subcommand.
func newDeployGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get deployment details",
		Long:  `Get detailed information about a specific deployment.`,
		Args:  cobra.ExactArgs(1),
		Example: `  codeai deploy get abc123
  codeai deploy get --output json abc123`,
		RunE: runDeployGet,
	}

	cmd.Flags().StringVar(&apiURL, "api-url", "http://localhost:8080", "API server URL")

	return cmd
}

func runDeployGet(cmd *cobra.Command, args []string) error {
	deployID := args[0]

	if apiURL == "" {
		return fmt.Errorf("API URL not configured")
	}

	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Getting deployment %s from %s\n", deployID, apiURL)
	}

	// TODO: Call API to get deployment
	fmt.Fprintf(cmd.OutOrStdout(), "Deployment %s not found (API integration not yet implemented)\n", deployID)

	return nil
}

// newDeployDeleteCmd creates the deploy delete subcommand.
func newDeployDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete deployment",
		Long:  `Delete a deployment by ID. Use --force to skip confirmation.`,
		Args:  cobra.ExactArgs(1),
		Example: `  codeai deploy delete abc123
  codeai deploy delete --force abc123`,
		RunE: runDeployDelete,
	}

	cmd.Flags().BoolVarP(&deployForce, "force", "f", false, "skip confirmation")
	cmd.Flags().StringVar(&apiURL, "api-url", "http://localhost:8080", "API server URL")

	return cmd
}

func runDeployDelete(cmd *cobra.Command, args []string) error {
	deployID := args[0]

	if apiURL == "" {
		return fmt.Errorf("API URL not configured")
	}

	if !deployForce {
		fmt.Fprintf(cmd.OutOrStdout(), "Are you sure you want to delete deployment %s? Use --force to skip this prompt.\n", deployID)
		return nil
	}

	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Deleting deployment %s\n", deployID)
	}

	// TODO: Call API to delete deployment
	fmt.Fprintf(cmd.OutOrStdout(), "Deployment %s deleted (API integration not yet implemented)\n", deployID)

	return nil
}

// newDeployExecuteCmd creates the deploy execute subcommand.
func newDeployExecuteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "execute <id>",
		Short: "Execute deployment",
		Long:  `Execute a deployment by ID. Use --wait to wait for completion.`,
		Args:  cobra.ExactArgs(1),
		Example: `  codeai deploy execute abc123
  codeai deploy execute --wait abc123`,
		RunE: runDeployExecute,
	}

	cmd.Flags().BoolVarP(&deployWait, "wait", "w", false, "wait for execution to complete")
	cmd.Flags().StringVar(&apiURL, "api-url", "http://localhost:8080", "API server URL")

	return cmd
}

func runDeployExecute(cmd *cobra.Command, args []string) error {
	deployID := args[0]

	if apiURL == "" {
		return fmt.Errorf("API URL not configured")
	}

	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Executing deployment %s\n", deployID)
	}

	// TODO: Call API to execute deployment
	result := map[string]string{
		"id":     deployID,
		"status": "pending",
		"note":   "API integration not yet implemented",
	}

	switch outputFormat {
	case "json":
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Execution started for deployment %s (API integration not yet implemented)\n", deployID)
	}

	return nil
}
