package commands

import "github.com/spf13/cobra"

func schemaCmd() *cobra.Command {
	return &cobra.Command{Use: "schema", Short: "Introspect database schema"}
}
