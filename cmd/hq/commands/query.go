// cmd/hq/commands/query.go
package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/eduardoserete/humanized-query/internal/audit"
	"github.com/eduardoserete/humanized-query/internal/cache"
	"github.com/eduardoserete/humanized-query/internal/config"
	"github.com/eduardoserete/humanized-query/internal/executor"
	"github.com/eduardoserete/humanized-query/internal/policy"
	"github.com/spf13/cobra"
)

func queryCmd() *cobra.Command {
	var dbName, sql string
	var includeHeader bool
	var offset int

	cmd := &cobra.Command{
		Use:   "query",
		Short: "Execute a read-only SQL query and stream CSV results",
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

			// Layer 2: lexical policy check
			if err := policy.Check(sql, cfg.Execution.AllowedSchemas); err != nil {
				logAudit(cfg, dbName, "rejected", err.Error(), sql, 0, 0)
				return writeError("forbidden_statement", err.Error())
			}

			adapter, err := executor.New(dbCfg.Dialect)
			if err != nil {
				return writeError("unsupported_dialect", err.Error())
			}

			// Pagination / limit logic
			finalSQL, pagination, err := executor.BuildQuery(adapter, sql, offset, cfg.Execution.MaxRows)
			if err != nil {
				var le *executor.LimitExceededError
				if isLimitExceeded(err, &le) {
					logAudit(cfg, dbName, "rejected", "limit_exceeded", sql, 0, 0)
					return writeLimitExceeded(le)
				}
				return writeError("query_error", err.Error())
			}

			// Layer 1: read-only connection
			db, err := adapter.Open(dbCfg.DSN)
			if err != nil {
				return writeError("connection_error", err.Error())
			}
			defer db.Close()

			ctx, cancel := context.WithTimeout(context.Background(),
				time.Duration(cfg.Execution.TimeoutSeconds)*time.Second)
			defer cancel()

			result, err := executor.StreamCSV(ctx, db, finalSQL, includeHeader, os.Stdout)
			if err != nil {
				if ctx.Err() != nil {
					logAudit(cfg, dbName, "rejected", "timeout", sql, 0, 0)
					return writeError("timeout", fmt.Sprintf("query exceeded %ds limit", cfg.Execution.TimeoutSeconds))
				}
				return writeError("query_error", err.Error())
			}

			durationMs := result.Duration.Milliseconds()
			logAudit(cfg, dbName, "ok", "", sql, result.RowCount, durationMs)
			updateCache(cfg, dbName, sql)

			// Pagination metadata to stderr
			if pagination != nil {
				fmt.Fprintf(os.Stderr, "# rows=%d page=%d has_more=%v next=--offset %d\n",
					result.RowCount, pagination.Page,
					result.RowCount == cfg.Execution.MaxRows,
					pagination.NextOffset())
			} else {
				fmt.Fprintf(os.Stderr, "# rows=%d duration_ms=%d\n", result.RowCount, durationMs)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&dbName, "db", "", "Database name (required)")
	cmd.Flags().StringVar(&sql, "sql", "", "SQL query to execute (required)")
	cmd.Flags().BoolVar(&includeHeader, "header", false, "Include column names as first CSV line")
	cmd.Flags().IntVar(&offset, "offset", 0, "Row offset for pagination")
	cmd.MarkFlagRequired("db")
	cmd.MarkFlagRequired("sql")

	return cmd
}

func isLimitExceeded(err error, out **executor.LimitExceededError) bool {
	var le *executor.LimitExceededError
	if ok := errors.As(err, &le); ok {
		*out = le
		return true
	}
	return false
}

func logAudit(cfg *config.Config, dbName, status, errCode, sql string, rows int, durationMs int64) {
	dir, err := hqDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "# audit write failed: %v\n", err)
		return
	}
	path := filepath.Join(dir, "audit.log")
	logger := audit.New(path)
	if err := logger.Log(audit.Entry{
		DB: dbName, Status: status, Error: errCode,
		SQL: sql, RowCount: rows, DurationMs: durationMs,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "# audit write failed: %v\n", err)
	}
}

// tableRe extracts table names from simple FROM/JOIN clauses, handling schema-qualified names.
var tableRe = regexp.MustCompile(`(?i)(?:FROM|JOIN)\s+(?:[a-z_][a-z0-9_]*\.)?([a-z_][a-z0-9_]*)`)

func updateCache(cfg *config.Config, dbName, sql string) {
	matches := tableRe.FindAllStringSubmatch(sql, -1)
	tables := make([]string, 0, len(matches))
	for _, m := range matches {
		tables = append(tables, strings.ToLower(m[1]))
	}
	if len(tables) == 0 {
		return
	}
	dir, err := hqDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "# cache write failed: %v\n", err)
		return
	}
	path := filepath.Join(dir, "cache", "table_usage.json")
	c := cache.New(path)
	if err := c.Increment(dbName, tables); err != nil {
		fmt.Fprintf(os.Stderr, "# cache write failed: %v\n", err)
	}
}
