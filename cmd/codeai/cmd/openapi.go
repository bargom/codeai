package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bargom/codeai/internal/openapi"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	openapiOutput   string
	openapiFormat   string
	openapiConfig   string
	openapiTitle    string
	openapiVersion  string
	openapiValidate bool
	openapiStrict   bool
)

func newOpenapiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "openapi",
		Short: "OpenAPI specification generation",
		Long:  `Generate OpenAPI 3.0 specifications from CodeAI source files.`,
	}

	cmd.AddCommand(newOpenapiGenerateCmd())
	cmd.AddCommand(newOpenapiValidateCmd())

	return cmd
}

func newOpenapiGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate <file>",
		Short: "Generate OpenAPI specification from CodeAI source",
		Long: `Generate an OpenAPI 3.0 specification from a CodeAI source file.

The generated specification includes:
- Paths and operations extracted from function declarations
- Schemas generated from type definitions
- Parameter definitions with validation constraints
- Response definitions

Example:
  codeai openapi generate api.cai -o api.yaml
  codeai openapi generate api.cai --format json -o api.json
  codeai openapi generate api.cai --config openapi.yaml`,
		Args: cobra.ExactArgs(1),
		RunE: runOpenapiGenerate,
	}

	cmd.Flags().StringVarP(&openapiOutput, "output", "o", "", "output file path (default: stdout)")
	cmd.Flags().StringVarP(&openapiFormat, "format", "f", "yaml", "output format (json|yaml)")
	cmd.Flags().StringVarP(&openapiConfig, "config", "c", "", "configuration file path")
	cmd.Flags().StringVar(&openapiTitle, "title", "", "API title (overrides config)")
	cmd.Flags().StringVar(&openapiVersion, "version", "", "API version (overrides config)")
	cmd.Flags().BoolVar(&openapiValidate, "validate", true, "validate generated spec")

	return cmd
}

func newOpenapiValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate <file>",
		Short: "Validate an OpenAPI specification",
		Long: `Validate an OpenAPI 3.0 specification file.

The validator checks for:
- Required fields (openapi version, info, paths)
- Valid path and operation definitions
- Parameter and schema correctness
- Security scheme definitions

Example:
  codeai openapi validate api.yaml
  codeai openapi validate api.json --strict`,
		Args: cobra.ExactArgs(1),
		RunE: runOpenapiValidate,
	}

	cmd.Flags().BoolVar(&openapiStrict, "strict", false, "enable strict validation mode")

	return cmd
}

func runOpenapiGenerate(cmd *cobra.Command, args []string) error {
	inputFile := args[0]

	// Load configuration
	var config *openapi.Config
	if openapiConfig != "" {
		var err error
		config, err = openapi.LoadConfig(openapiConfig)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	} else {
		config = openapi.DefaultConfig()
	}

	// Override config with flags
	if openapiTitle != "" {
		config.Title = openapiTitle
	}
	if openapiVersion != "" {
		config.Version = openapiVersion
	}
	if openapiFormat != "" {
		config.OutputFormat = openapiFormat
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	printVerbose(cmd, "Parsing %s...\n", inputFile)

	// Create generator
	generator := openapi.NewGenerator(config)

	// Generate from file
	spec, err := generator.GenerateFromFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to generate OpenAPI spec: %w", err)
	}

	// Validate if requested
	if openapiValidate {
		printVerbose(cmd, "Validating generated specification...\n")
		result := openapi.ValidateSpec(spec)
		if !result.Valid {
			fmt.Fprintln(cmd.ErrOrStderr(), "Validation errors:")
			for _, e := range result.Errors {
				fmt.Fprintf(cmd.ErrOrStderr(), "  - %s\n", e.Error())
			}
			return fmt.Errorf("generated specification has validation errors")
		}
		if len(result.Warnings) > 0 && isVerbose() {
			fmt.Fprintln(cmd.ErrOrStderr(), "Validation warnings:")
			for _, warn := range result.Warnings {
				fmt.Fprintf(cmd.ErrOrStderr(), "  - %s\n", warn.Error())
			}
		}
	}

	// Determine output format
	format := strings.ToLower(openapiFormat)
	if openapiOutput != "" && format == "" {
		ext := strings.ToLower(filepath.Ext(openapiOutput))
		if ext == ".json" {
			format = "json"
		} else {
			format = "yaml"
		}
	}
	if format == "" {
		format = "yaml"
	}

	// Output
	if openapiOutput != "" {
		printVerbose(cmd, "Writing to %s...\n", openapiOutput)
		if err := generator.WriteToFile(openapiOutput, spec); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "OpenAPI specification written to %s\n", openapiOutput)
	} else {
		// Write to stdout
		switch format {
		case "json":
			if err := generator.WriteJSON(cmd.OutOrStdout(), spec); err != nil {
				return fmt.Errorf("failed to write JSON: %w", err)
			}
		default:
			if err := generator.WriteYAML(cmd.OutOrStdout(), spec); err != nil {
				return fmt.Errorf("failed to write YAML: %w", err)
			}
		}
	}

	return nil
}

func runOpenapiValidate(cmd *cobra.Command, args []string) error {
	inputFile := args[0]

	printVerbose(cmd, "Reading %s...\n", inputFile)

	// Read the file
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse based on extension
	var spec openapi.OpenAPI
	ext := strings.ToLower(filepath.Ext(inputFile))

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &spec); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &spec); err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}
	default:
		// Try YAML first, then JSON
		if err := yaml.Unmarshal(data, &spec); err != nil {
			if err := json.Unmarshal(data, &spec); err != nil {
				return fmt.Errorf("failed to parse file (tried YAML and JSON)")
			}
		}
	}

	printVerbose(cmd, "Validating specification...\n")

	// Validate
	validator := openapi.NewValidator()
	validator.StrictMode = openapiStrict

	result := validator.Validate(&spec)

	// Output results
	if result.Valid {
		fmt.Fprintln(cmd.OutOrStdout(), "Specification is valid.")
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "Specification has validation errors:")
		for _, e := range result.Errors {
			fmt.Fprintf(cmd.OutOrStdout(), "  ERROR: %s\n", e.Error())
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "\nWarnings:")
		for _, warn := range result.Warnings {
			fmt.Fprintf(cmd.OutOrStdout(), "  WARNING: %s\n", warn.Error())
		}
	}

	if !result.Valid {
		return fmt.Errorf("validation failed with %d errors", len(result.Errors))
	}

	return nil
}

func init() {
	rootCmd.AddCommand(newOpenapiCmd())
}
