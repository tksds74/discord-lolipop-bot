package permission

import (
	"encoding/json"
	"fmt"
	"os"
)

type Mode string

const (
	ModeOpen  Mode = "open"
	ModeAllow Mode = "allow"
	ModeDeny  Mode = "deny"
)

type Config struct {
	Mode Mode     `json:"mode"`
	IDs  []string `json:"ids"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read permission config: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse permission config: %w", err)
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *Config) validate() error {
	switch c.Mode {
	case ModeOpen, ModeAllow, ModeDeny:
		return nil
	default:
		return fmt.Errorf("invalid permission mode: %s", c.Mode)
	}
}

func (c *Config) IsAllowed(userID string) bool {
	switch c.Mode {
	case ModeOpen:
		return true
	case ModeAllow:
		return contains(c.IDs, userID)
	case ModeDeny:
		return !contains(c.IDs, userID)
	default:
		return false
	}
}

func contains(ids []string, target string) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}
