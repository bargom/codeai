package cmd

import (
	"fmt"

	"github.com/bargom/codeai/internal/parser"
	"github.com/bargom/codeai/internal/validator"
	"github.com/spf13/cobra"
)

// newValidateCmd creates the validate command.
func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate <file>",
		Short: "Validate DSL file for syntax and semantic errors",
		Long: `Validate a CodeAI DSL file for both syntax and semantic errors.

This command performs two-phase validation:
1. Syntax validation - Checks that the file follows the CodeAI grammar
2. Semantic validation - Checks for undefined variables, type errors, etc.

Exit code 0 indicates a valid file, non-zero indicates errors.`,
		Args:    cobra.ExactArgs(1),
		Example: `  codeai validate myfile.cai
  codeai validate --verbose myfile.cai`,
		RunE: runValidate,
	}

	return cmd
}

func runValidate(cmd *cobra.Command, args []string) error {
	filename := args[0]

	// Verbose output
	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Validating file: %s\n", filename)
	}

	// Parse the file
	program, err := parser.ParseFile(filename)
	if err != nil {
		return fmt.Errorf("syntax error: %w", err)
	}

	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Syntax: OK\n")
	}

	// Validate semantics
	v := validator.New()
	if err := v.Validate(program); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Semantics: OK\n")
	}

	fmt.Fprintf(cmd.OutOrStdout(), "File is valid: %s\n", filename)
	return nil
}
