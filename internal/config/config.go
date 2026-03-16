package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type DBConfig struct {
	DSN     string `yaml:"dsn"`
	Dialect string `yaml:"dialect"`
}

type ExecutionConfig struct {
	MaxRows        int      `yaml:"max_rows"`
	TimeoutSeconds int      `yaml:"timeout_seconds"`
	AllowedSchemas []string `yaml:"allowed_schemas"`
}

type KnowledgeConfig struct {
	CacheTopN int `yaml:"cache_top_n"`
}

type Config struct {
	Databases map[string]DBConfig `yaml:"databases"`
	Execution ExecutionConfig     `yaml:"execution"`
	Knowledge KnowledgeConfig     `yaml:"knowledge"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: cannot read %s: %w", path, err)
	}
	cfg := &Config{
		Execution: ExecutionConfig{
			MaxRows:        1000,
			TimeoutSeconds: 30,
		},
		Knowledge: KnowledgeConfig{
			CacheTopN: 10,
		},
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("config: invalid yaml: %w", err)
	}
	return cfg, nil
}

func (c *Config) DB(name string) (DBConfig, error) {
	db, ok := c.Databases[name]
	if !ok {
		return DBConfig{}, fmt.Errorf("db_not_found: no database configured with name %q", name)
	}
	return db, nil
}
