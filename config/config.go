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

	LLM   LLMConfig    `json:"llm"`
	Agent AgentConfig  `json:"agent"`
	Tools []ToolConfig `json:"tools"`

	RateLimiting RateLimitingConfig `json:"rate_limiting,omitempty"`
	Logging      LoggingConfig      `json:"logging,omitempty"`
	Scheduler    SchedulerConfig    `json:"scheduler,omitempty"`
}

type LLMConfig struct {
	Provider    string  `json:"provider"`
	Endpoint    string  `json:"endpoint"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	APIKeyPath  string  `json:"api_key_path,omitempty"`
}

type AgentConfig struct {
	Mode          string `json:"mode"`
	MaxIterations string `json:"max_iterations"`
	SystemPrompt  string `json:"system_prompt"`
}
type ToolConfig struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	RequiredArgs   []string `json:"required_args,omitempty"`
	TimeoutSeconds int      `json:"timeout_seconds,omitempty"`
	RetryCount     int      `json:"retry_count,omitempty"`
	RetryBackoffMS int      `json:"retry_backoff_ms,omitempty"`
}
type RateLimitingConfig struct {
	TockenBucket  TockenBucketConfig  `json:"token_bucket"`
	SlidingWindow SlidingWindowConfig `json:"sliding_window"`
}
type TockenBucketConfig struct {
	Capacity         int     `json:"capacity"`
	RefillRatePerSec float64 `json:"refill_rate_per_sec"`
}
type SlidingWindowConfig struct {
	ToolLimit   int `json:"tool_limit"`
	WindowHours int `json:"window_hours"`
}
type LoggingConfig struct {
	Level          string `json:"level"`
	Output         string `json:"output"`
	FilePath       string `json:"file_path"`
	MaxFileSizeMB  int    `json:"max_file_size_mb"`
	MaxBackupFiles int    `json:"max_backup_files"`
}
type SchedulerConfig struct {
	Mode string `json:"mode"`
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
