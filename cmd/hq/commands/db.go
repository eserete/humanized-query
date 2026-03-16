// cmd/hq/commands/db.go
package commands

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/eduardoserete/humanized-query/internal/config"
	"github.com/spf13/cobra"
)

func dbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Inspect configured database connections",
	}
	cmd.AddCommand(dbListCmd())
	return cmd
}

func dbListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured databases",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, err := configPath()
			if err != nil {
				return writeError("config_not_found",
					fmt.Sprintf("~/.hq/config.yaml does not exist or is invalid: %v", err))
			}
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return writeError("config_not_found",
					fmt.Sprintf("~/.hq/config.yaml does not exist or is invalid: %v", err))
			}

			type dbEntry struct {
				Name    string `json:"name"`
				Dialect string `json:"dialect"`
				DSN     string `json:"dsn"`
			}

			entries := make([]dbEntry, 0, len(cfg.Databases))
			for name, db := range cfg.Databases {
				entries = append(entries, dbEntry{
					Name:    name,
					Dialect: db.Dialect,
					DSN:     maskDSN(db.DSN),
				})
			}

			result := map[string]interface{}{"databases": entries}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		},
	}
}

// maskDSN replaces the password in a DSN string with "***".
func maskDSN(dsn string) string {
	// Handle non-URL DSNs like MySQL driver format: user:pass@tcp(host)/db
	// These lack a "://" scheme separator, so handle them via string splitting.
	if !strings.Contains(dsn, "://") {
		parts := strings.SplitN(dsn, "@", 2)
		if len(parts) == 2 {
			creds := strings.SplitN(parts[0], ":", 2)
			if len(creds) == 2 {
				return creds[0] + ":***@" + parts[1]
			}
		}
		return dsn
	}

	// URL-format DSN (e.g. postgres://user:pass@host/db).
	// Reconstruct manually to avoid percent-encoding the masked password.
	u, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}
	if u.User != nil {
		if _, hasPass := u.User.Password(); hasPass {
			// Rebuild the URL string with the literal "***" password to avoid
			// url.String() percent-encoding the asterisks as %2A.
			username := u.User.Username()
			host := u.Host
			if u.Port() == "" {
				// host already has no port
			}
			path := u.Path
			if u.RawQuery != "" {
				path += "?" + u.RawQuery
			}
			if u.Fragment != "" {
				path += "#" + u.Fragment
			}
			return fmt.Sprintf("%s://%s:***@%s%s", u.Scheme, username, host, path)
		}
	}
	return u.String()
}
