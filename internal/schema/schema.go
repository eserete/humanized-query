package schema

import (
	"database/sql"
	"fmt"
)

// Column describes a single column in a table.
type Column struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

// ForeignKey describes a foreign key relationship.
type ForeignKey struct {
	Column           string `json:"column"`
	ReferencesTable  string `json:"references_table"`
	ReferencesColumn string `json:"references_column"`
}

// Table describes a database table.
type Table struct {
	Columns     []Column     `json:"columns"`
	PrimaryKey  []string     `json:"primary_key"`
	ForeignKeys []ForeignKey `json:"foreign_keys"`
}

// Schema is the full introspection result for a database.
type Schema struct {
	Database string           `json:"database"`
	Dialect  string           `json:"dialect"`
	Tables   map[string]Table `json:"tables"`
}

// Introspector loads schema information from a live database.
type Introspector interface {
	Introspect(db *sql.DB, dbName string, tableFilter string) (*Schema, error)
}

// New returns the Introspector for the given dialect.
func New(dialect string) (Introspector, error) {
	switch dialect {
	case "postgres":
		return &postgresIntrospector{}, nil
	case "mariadb", "mysql":
		return &mariadbIntrospector{}, nil
	default:
		return nil, fmt.Errorf("unsupported dialect: %q", dialect)
	}
}
