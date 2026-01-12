package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/bargom/codeai/internal/parser"
	"github.com/spf13/cobra"
)

// newParseCmd creates the parse command.
func newParseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "parse <file>",
		Short: "Parse DSL file and display AST",
		Long: `Parse a CodeAI DSL file and display the resulting Abstract Syntax Tree (AST).

This command reads a .cai file, parses it using the CodeAI grammar,
and outputs the AST structure. Useful for debugging and understanding
the structure of your DSL code.`,
		Args:    cobra.ExactArgs(1),
		Example: `  codeai parse myfile.cai
  codeai parse --output json myfile.cai
  codeai parse --verbose myfile.cai`,
		RunE: runParse,
	}

	return cmd
}

func runParse(cmd *cobra.Command, args []string) error {
	filename := args[0]

	// Verbose output
	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Parsing file: %s\n", filename)
	}

	// Parse the file
	program, err := parser.ParseFile(filename)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	// Output based on format
	switch outputFormat {
	case "json":
		return outputJSON(cmd, program)
	case "table":
		return outputAST(cmd, program) // Table format same as plain for AST
	default:
		return outputAST(cmd, program)
	}
}

// outputJSON outputs the AST as JSON.
func outputJSON(cmd *cobra.Command, data interface{}) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// outputAST outputs the AST in human-readable format.
func outputAST(cmd *cobra.Command, program interface{}) error {
	// Use the AST's String() method or custom formatting
	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", formatAST(program))
	return nil
}

// formatAST formats the AST for display.
func formatAST(v interface{}) string {
	// Use JSON for now as a readable format, but could be custom tree format
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%+v", v)
	}
	return string(data)
}
