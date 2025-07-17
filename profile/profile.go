package profile

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

type Profile struct {
	Read Action `yaml:"read"` // Mark as read issues/pull requests that match the conditions
	Open Action `yaml:"open"` // Open issues/pull requests that match the conditions
	List Action `yaml:"list"` // List issues/pull requests that match the conditions
}

var defaultProfile = &Profile{
	Read: Action{
		Max: 1000,
		Conditions: []string{
			"merged",
		},
	},
	Open: Action{
		Max:        1,
		Conditions: []string{"is_pull_request && me in reviewers && passed && !approved && open && !draft"},
	},
	List: Action{
		Max:        1000,
		Conditions: []string{"*"},
	},
}

func profilePathWithName(name string) string {
	var dataHomePath string
	if os.Getenv("XDG_DATA_HOME") != "" {
		dataHomePath = filepath.Join(os.Getenv("XDG_DATA_HOME"), "gh-triage")
	} else {
		dataHomePath = filepath.Join(os.Getenv("HOME"), ".local", "share", "gh-triage")
	}

	var configFile string
	if name == "" {
		configFile = "default.yml"
	} else {
		configFile = name + ".yml"
	}
	return filepath.Join(dataHomePath, configFile)
}

func Load(name string) (*Profile, error) {
	p := profilePathWithName(name)

	// Migration: config.yml -> default.yml (only for empty name)
	if name == "" {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			// Check if old config.yml exists
			oldConfigPath := filepath.Join(filepath.Dir(p), "config.yml")
			if _, err := os.Stat(oldConfigPath); err == nil {
				// Copy config.yml to default.yml
				if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
					return nil, err
				}
				data, err := os.ReadFile(oldConfigPath)
				if err != nil {
					return nil, err
				}
				if err := os.WriteFile(p, data, 0600); err != nil {
					return nil, err
				}
				// Remove old config.yml
				if err := os.Remove(oldConfigPath); err != nil {
					return nil, err
				}
				slog.Info("migrated config file", "from", oldConfigPath, "to", p)
			}
		}
	}

	if _, err := os.Stat(p); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			return nil, err
		}
		b, err := yaml.Marshal(defaultProfile)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(p, b, 0600); err != nil {
			return nil, err
		}
		slog.Info("created config file", "path", p)
		return defaultProfile, nil
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var p2 Profile
	if err := yaml.Unmarshal(b, &p2); err != nil {
		return nil, err
	}
	return &p2, nil
}
