// internal/executor/executor.go
package executor

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"time"
)

// LimitExceededError is returned when a query requests more rows than allowed.
type LimitExceededError struct {
	Requested  int
	MaxAllowed int
	Query      string
}

func (e *LimitExceededError) Error() string {
	return fmt.Sprintf("limit_exceeded: requested %d, max_allowed %d", e.Requested, e.MaxAllowed)
}

// Pagination holds metadata for paginated results.
type Pagination struct {
	Page    int
	Offset  int
	MaxRows int
}

// NextOffset returns the offset for the next page.
func (p *Pagination) NextOffset() int {
	return p.Offset + p.MaxRows
}

// BuildQuery applies limit logic and returns the final query + optional pagination info.
// Returns LimitExceededError if user-specified LIMIT exceeds maxRows.
func BuildQuery(a Adapter, query string, offset, maxRows int) (string, *Pagination, error) {
	has, n, err := a.HasLimit(query)
	if err != nil {
		return "", nil, err
	}
	if has {
		if n > maxRows {
			return "", nil, &LimitExceededError{
				Requested:  n,
				MaxAllowed: maxRows,
				Query:      query,
			}
		}
		// User LIMIT is within bounds — execute as-is, no pagination
		return query, nil, nil
	}
	// No LIMIT — inject and enable pagination
	q := a.InjectLimit(query, maxRows, offset)
	page := (offset / maxRows) + 1
	return q, &Pagination{Page: page, Offset: offset, MaxRows: maxRows}, nil
}

// Result holds streaming query results.
type Result struct {
	Columns  []string
	RowCount int
	Duration time.Duration
}

// StreamCSV executes query against db, writes CSV rows to w, and returns Result.
// If includeHeader is true, writes column names as first CSV line.
func StreamCSV(ctx context.Context, db *sql.DB, query string, includeHeader bool, w io.Writer) (*Result, error) {
	start := time.Now()
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("columns error: %w", err)
	}

	cw := csv.NewWriter(w)
	if includeHeader {
		if err := cw.Write(cols); err != nil {
			return nil, err
		}
		cw.Flush()
	}

	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}

	count := 0
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		record := make([]string, len(cols))
		for i, v := range vals {
			record[i] = fmt.Sprintf("%v", v)
		}
		if err := cw.Write(record); err != nil {
			return nil, err
		}
		cw.Flush()
		count++
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return &Result{
		Columns:  cols,
		RowCount: count,
		Duration: time.Since(start),
	}, nil
}
