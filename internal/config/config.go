// Package config handles application configuration loading and saving.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all application settings loaded from config.yaml.
type Config struct {
	PollInterval time.Duration     `yaml:"poll_interval"`
	LogTail      int               `yaml:"log_tail"`
	MaxLogLines  int               `yaml:"max_log_lines"`
	KeyBindings  map[string]string `yaml:"keybindings"`
	Bookmarks    []string          `yaml:"bookmarks"`
	Theme        string            `yaml:"theme"`
	DockerHost   string            `yaml:"docker_host"`
	Mouse        bool              `yaml:"mouse"`
}

// Default returns a Config with default values.
func Default() *Config {
	return &Config{
		PollInterval: 2 * time.Second,
		LogTail:      100,
		MaxLogLines:  1000,
		Theme:        "auto",
	}
}

// Load reads configuration from ~/.config/wharf/config.yaml.
// Returns default config if the file does not exist.
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
	if cfg.Theme == "" {
		cfg.Theme = "auto"
	}

	return cfg, nil
}

func (c *Config) IsBookmarked(name string) bool {
	for _, b := range c.Bookmarks {
		if b == name {
			return true
		}
	}
	return false
}

func (c *Config) ToggleBookmark(name string) {
	for i, b := range c.Bookmarks {
		if b == name {
			c.Bookmarks = append(c.Bookmarks[:i], c.Bookmarks[i+1:]...)
			return
		}
	}
	c.Bookmarks = append(c.Bookmarks, name)
}

// Save writes the current configuration to disk.
func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// Path returns the config file path.
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
	out += fmt.Sprintf("theme: %s\n", c.Theme)
	if len(c.Bookmarks) > 0 {
		out += fmt.Sprintf("bookmarks: %v\n", c.Bookmarks)
	}
	if len(c.KeyBindings) > 0 {
		out += "keybindings:\n"
		for k, v := range c.KeyBindings {
			out += fmt.Sprintf("  %s: %s\n", k, v)
		}
	}
	return out
}
