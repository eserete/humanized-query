package commands

import "github.com/spf13/cobra"

func dbCmd() *cobra.Command {
	return &cobra.Command{Use: "db", Short: "Inspect configured databases"}
}
