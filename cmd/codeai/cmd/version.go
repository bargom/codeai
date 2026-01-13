package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// Version information (set at build time via ldflags)
var (
	Version   = "0.3.0"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

// VersionInfo holds version information for JSON output.
type VersionInfo struct {
	Version   string `json:"version"`
	BuildDate string `json:"buildDate"`
	GitCommit string `json:"gitCommit"`
}

// newVersionCmd creates the version command.
func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long: `Print the version, build date, and git commit of the CodeAI CLI.

This information is useful for debugging and ensuring you have the correct version.`,
		Args:    cobra.NoArgs,
		Example: `  codeai version
  codeai version --output json`,
		RunE: runVersion,
	}

	return cmd
}

func runVersion(cmd *cobra.Command, args []string) error {
	info := VersionInfo{
		Version:   Version,
		BuildDate: BuildDate,
		GitCommit: GitCommit,
	}

	switch outputFormat {
	case "json":
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(info)
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "CodeAI v%s\n", Version)
		fmt.Fprintf(cmd.OutOrStdout(), "Build Date: %s\n", BuildDate)
		fmt.Fprintf(cmd.OutOrStdout(), "Git Commit: %s\n", GitCommit)
		return nil
	}
}
