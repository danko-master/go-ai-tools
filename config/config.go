// Configuration loader
// Usage:
// cfg, err := config.Load("configs/basic.json")

// scenario := cfg.Scenario
//
//	basic, production, cicd, aaas (agents as a service)
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Scenario    string `json:"scenario"`
}

// Load reads and parse a JSON config file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)

	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	// Infer scenario from path if not set
	if cfg.Scenario == "" {
		if strings.Contains(path, "basic") {
			cfg.Scenario = "basic"
		} else if strings.Contains(path, "production") {
			cfg.Scenario = "production"
		}
	}

	return &cfg, nil
}

// Print config summary
func (c *Config) Print() {
	fmt.Println("Config:")
	fmt.Printf(" %s v%s (%s)\n", c.Name, c.Version, c.Scenario)
}
