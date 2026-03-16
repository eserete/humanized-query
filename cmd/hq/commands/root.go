package commands

import "github.com/spf13/cobra"

func Root() *cobra.Command {
	root := &cobra.Command{
		Use:          "hq",
		Short:        "Safe read-only SQL execution CLI for AI agents",
		Long:         "hq executes SQL queries safely against configured databases and streams results as CSV.",
		SilenceUsage: true,
	}
	root.AddCommand(queryCmd())
	root.AddCommand(schemaCmd())
	root.AddCommand(dbCmd())
	return root
}
