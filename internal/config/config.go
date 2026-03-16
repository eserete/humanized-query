package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// MaskingRuleConfig holds a single custom masking rule from config.yaml.
type MaskingRuleConfig struct {
	Name        string `yaml:"name"`
	Regex       string `yaml:"regex"`
	Replacement string `yaml:"replacement"`
}

// MaskingConfig holds custom masking rules from config.yaml.
type MaskingConfig struct {
	Rules []MaskingRuleConfig `yaml:"rules"`
}

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
	Masking   *MaskingConfig      `yaml:"masking"`
}

// Load reads and parses the config file at path.
// DSN values containing ${VAR} or $VAR references are expanded via os.Expand.
// Returns an error if any referenced env var is unset.
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
	if err := expandDSNs(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// expandDSNs expands environment variable references in all DSN fields.
// Returns an error if any referenced variable is not set.
func expandDSNs(cfg *Config) error {
	expanded := make(map[string]DBConfig, len(cfg.Databases))
	for name, db := range cfg.Databases {
		var expandErr string
		dsn := os.Expand(db.DSN, func(varName string) string {
			val, ok := os.LookupEnv(varName)
			if !ok {
				expandErr = varName
				return ""
			}
			return val
		})
		if expandErr != "" {
			return fmt.Errorf("config: DSN for database %q references unset env var %s", name, expandErr)
		}
		expanded[name] = DBConfig{DSN: dsn, Dialect: db.Dialect}
	}
	cfg.Databases = expanded
	return nil
}

func (c *Config) DB(name string) (DBConfig, error) {
	db, ok := c.Databases[name]
	if !ok {
		return DBConfig{}, fmt.Errorf("db_not_found: no database configured with name %q", name)
	}
	return db, nil
}
