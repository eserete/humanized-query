package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/eduardoserete/humanized-query/internal/executor"
)

func hqDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".hq"), nil
}

func configPath() (string, error) {
	dir, err := hqDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

func writeError(code, detail string) error {
	b, _ := json.Marshal(map[string]string{"error": code, "detail": detail})
	fmt.Fprintln(os.Stderr, string(b))
	os.Exit(1)
	return nil
}

func writeLimitExceeded(le *executor.LimitExceededError) error {
	b, _ := json.Marshal(map[string]interface{}{
		"error":       "limit_exceeded",
		"requested":   le.Requested,
		"max_allowed": le.MaxAllowed,
		"query":       le.Query,
	})
	fmt.Fprintln(os.Stderr, string(b))
	os.Exit(1)
	return nil
}
