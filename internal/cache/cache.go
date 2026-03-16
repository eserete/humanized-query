package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// usageData maps db -> table -> count
type usageData map[string]map[string]int

// Cache manages table_usage.json.
type Cache struct {
	path string
}

// New returns a Cache backed by the given file path.
func New(path string) *Cache {
	return &Cache{path: path}
}

// Increment adds 1 to each table's count for the given db.
// Non-fatal: returns error but callers should log to stderr and continue.
func (c *Cache) Increment(db string, tables []string) error {
	data, err := c.load()
	if err != nil {
		return err
	}
	if data[db] == nil {
		data[db] = make(map[string]int)
	}
	for _, t := range tables {
		data[db][t]++
	}
	return c.save(data)
}

// TopN returns the N most-used tables for db, as table->count map.
func (c *Cache) TopN(db string, n int) (map[string]int, error) {
	data, err := c.load()
	if err != nil {
		return nil, err
	}
	tables := data[db]
	if tables == nil {
		return map[string]int{}, nil
	}

	type entry struct {
		name  string
		count int
	}
	entries := make([]entry, 0, len(tables))
	for name, count := range tables {
		entries = append(entries, entry{name, count})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})

	result := make(map[string]int)
	for i, e := range entries {
		if i >= n {
			break
		}
		result[e.name] = e.count
	}
	return result, nil
}

func (c *Cache) load() (usageData, error) {
	data := make(usageData)
	b, err := os.ReadFile(c.path)
	if os.IsNotExist(err) {
		return data, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cache: read failed: %w", err)
	}
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("cache: parse failed: %w", err)
	}
	return data, nil
}

func (c *Cache) save(data usageData) error {
	if err := os.MkdirAll(filepath.Dir(c.path), 0700); err != nil {
		return fmt.Errorf("cache: mkdir failed: %w", err)
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("cache: marshal failed: %w", err)
	}
	return os.WriteFile(c.path, b, 0600)
}
