package commands

import "github.com/spf13/cobra"

func queryCmd() *cobra.Command {
	return &cobra.Command{Use: "query", Short: "Execute a read-only SQL query"}
}
