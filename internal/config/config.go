package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	PollInterval time.Duration     `yaml:"poll_interval"`
	LogTail      int               `yaml:"log_tail"`
	MaxLogLines  int               `yaml:"max_log_lines"`
	KeyBindings  map[string]string `yaml:"keybindings"`
}

func Default() *Config {
	return &Config{
		PollInterval: 2 * time.Second,
		LogTail:      100,
		MaxLogLines:  1000,
	}
}

func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return Default(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 2 * time.Second
	}
	if cfg.LogTail <= 0 {
		cfg.LogTail = 100
	}
	if cfg.MaxLogLines <= 0 {
		cfg.MaxLogLines = 1000
	}

	return cfg, nil
}

func Path() string {
	p, err := configPath()
	if err != nil {
		return "(unknown)"
	}
	return p
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "wharf", "config.yaml"), nil
}

func (c *Config) String() string {
	out := fmt.Sprintf("Config path: %s\n\n", Path())
	out += fmt.Sprintf("poll_interval: %s\n", c.PollInterval)
	out += fmt.Sprintf("log_tail: %d\n", c.LogTail)
	out += fmt.Sprintf("max_log_lines: %d\n", c.MaxLogLines)
	if len(c.KeyBindings) > 0 {
		out += "keybindings:\n"
		for k, v := range c.KeyBindings {
			out += fmt.Sprintf("  %s: %s\n", k, v)
		}
	}
	return out
}
