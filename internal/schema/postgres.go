package schema

import (
	"database/sql"
	"fmt"
)

type postgresIntrospector struct {
	dialect string
}

func (p *postgresIntrospector) Introspect(db *sql.DB, dbName, tableFilter string) (*Schema, error) {
	s := &Schema{
		Database: dbName,
		Dialect:  p.dialect,
		Tables:   make(map[string]Table),
	}

	// loadColumns must run first — loadPrimaryKeys and loadForeignKeys reference tables by name
	if err := p.loadColumns(db, s, tableFilter); err != nil {
		return nil, err
	}
	if err := p.loadPrimaryKeys(db, s, tableFilter); err != nil {
		return nil, err
	}
	if err := p.loadForeignKeys(db, s, tableFilter); err != nil {
		return nil, err
	}
	return s, nil
}

func (p *postgresIntrospector) loadColumns(db *sql.DB, s *Schema, tableFilter string) error {
	var (
		rows *sql.Rows
		err  error
	)
	if tableFilter != "" {
		rows, err = db.Query(`
			SELECT table_name, column_name, data_type, is_nullable
			FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = $1
			ORDER BY ordinal_position`, tableFilter)
	} else {
		rows, err = db.Query(`
			SELECT table_name, column_name, data_type, is_nullable
			FROM information_schema.columns
			WHERE table_schema = 'public'
			ORDER BY table_name, ordinal_position`)
	}
	if err != nil {
		return fmt.Errorf("schema: column query failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, colName, dataType, isNullable string
		if err := rows.Scan(&tableName, &colName, &dataType, &isNullable); err != nil {
			return fmt.Errorf("schema: scanning column row: %w", err)
		}
		t := s.Tables[tableName]
		t.Columns = append(t.Columns, Column{
			Name:     colName,
			Type:     dataType,
			Nullable: isNullable == "YES",
		})
		s.Tables[tableName] = t
	}
	return rows.Err()
}

func (p *postgresIntrospector) loadPrimaryKeys(db *sql.DB, s *Schema, tableFilter string) error {
	var (
		rows *sql.Rows
		err  error
	)
	base := `
		SELECT kcu.table_name, kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name
		 AND tc.table_schema = kcu.table_schema
		WHERE tc.constraint_type = 'PRIMARY KEY'
		  AND tc.table_schema = 'public'`
	orderBy := ` ORDER BY kcu.ordinal_position`
	if tableFilter != "" {
		rows, err = db.Query(base+` AND kcu.table_name = $1`+orderBy, tableFilter)
	} else {
		rows, err = db.Query(base + orderBy)
	}
	if err != nil {
		return fmt.Errorf("schema: pk query failed: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var tableName, colName string
		if err := rows.Scan(&tableName, &colName); err != nil {
			return fmt.Errorf("schema: scanning pk row: %w", err)
		}
		t := s.Tables[tableName]
		t.PrimaryKey = append(t.PrimaryKey, colName)
		s.Tables[tableName] = t
	}
	return rows.Err()
}

func (p *postgresIntrospector) loadForeignKeys(db *sql.DB, s *Schema, tableFilter string) error {
	var (
		rows *sql.Rows
		err  error
	)
	base := `
		SELECT
			kcu.table_name,
			kcu.column_name,
			ccu.table_name AS foreign_table,
			ccu.column_name AS foreign_column
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name
		 AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu
		  ON ccu.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
		  AND tc.table_schema = 'public'`
	if tableFilter != "" {
		rows, err = db.Query(base+` AND kcu.table_name = $1`, tableFilter)
	} else {
		rows, err = db.Query(base)
	}
	if err != nil {
		return fmt.Errorf("schema: fk query failed: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var tableName, colName, foreignTable, foreignCol string
		if err := rows.Scan(&tableName, &colName, &foreignTable, &foreignCol); err != nil {
			return fmt.Errorf("schema: scanning fk row: %w", err)
		}
		t := s.Tables[tableName]
		t.ForeignKeys = append(t.ForeignKeys, ForeignKey{
			Column:           colName,
			ReferencesTable:  foreignTable,
			ReferencesColumn: foreignCol,
		})
		s.Tables[tableName] = t
	}
	return rows.Err()
}
