package config

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

type Action struct {
	Max        int      `yaml:"max"`        // Maximum number of issues/pull requests to process
	Conditions []string `yaml:"conditions"` // Conditions to match issues/pull requests
}

type Config struct {
	Read Action `yaml:"read"` // Mark as read issues/pull requests that match the conditions
	Open Action `yaml:"open"` // Open issues/pull requests that match the conditions
	List Action `yaml:"list"` // List issues/pull requests that match the conditions
}

var defaultConfig = &Config{
	Read: Action{
		Max: 1000,
		Conditions: []string{
			"merged",
		},
	},
	Open: Action{
		Max:        1,
		Conditions: []string{"is_pull_request && me in reviewers && passed && !approved && !draft && !closed && !merged"},
	},
	List: Action{
		Max:        1000, // Maximum number of issues/pull requests to list
		Conditions: []string{"*"},
	}, // List all issues/pull requests
}

func configPath() string {
	var dataHomePath string
	if os.Getenv("XDG_DATA_HOME") != "" {
		dataHomePath = filepath.Join(os.Getenv("XDG_DATA_HOME"), "gh-triage")
	} else {
		dataHomePath = filepath.Join(os.Getenv("HOME"), ".local", "share", "gh-triage")
	}
	return filepath.Join(dataHomePath, "config.yml")
}

func Load() (*Config, error) {
	p := configPath()
	if _, err := os.Stat(configPath()); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			return nil, err
		}
		b, err := yaml.Marshal(defaultConfig)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(p, b, 0600); err != nil {
			return nil, err
		}
		slog.Info("created config file", "path", p)
		return defaultConfig, nil
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
