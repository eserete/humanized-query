// cmd/hq/commands/schema.go
package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/eduardoserete/humanized-query/internal/config"
	"github.com/eduardoserete/humanized-query/internal/executor"
	"github.com/eduardoserete/humanized-query/internal/schema"
	"github.com/spf13/cobra"
)

func schemaCmd() *cobra.Command {
	var dbName, tableFilter string

	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Introspect database schema and return JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, err := configPath()
			if err != nil {
				return writeError("config_error", err.Error())
			}
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return writeError("config_error", err.Error())
			}

			dbCfg, err := cfg.DB(dbName)
			if err != nil {
				return writeError("db_not_found", fmt.Sprintf("no database named %s in ~/.hq/config.yaml", dbName))
			}

			adapter, err := executor.New(dbCfg.Dialect)
			if err != nil {
				return writeError("unsupported_dialect", err.Error())
			}

			db, err := adapter.Open(dbCfg.DSN)
			if err != nil {
				return writeError("connection_error", err.Error())
			}
			defer db.Close()

			introspector, err := schema.New(dbCfg.Dialect)
			if err != nil {
				return writeError("unsupported_dialect", err.Error())
			}

			s, err := introspector.Introspect(db, dbName, tableFilter)
			if err != nil {
				return writeError("introspection_error", err.Error())
			}

			if s == nil {
				return writeError("introspection_error", "introspector returned nil schema")
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(s)
		},
	}

	cmd.Flags().StringVar(&dbName, "db", "", "Database name (required)")
	cmd.Flags().StringVar(&tableFilter, "table", "", "Filter to a specific table")
	cmd.MarkFlagRequired("db")

	return cmd
}
