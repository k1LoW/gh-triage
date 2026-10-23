package cmd

import (
	"os"
	"testing"
)

func TestApplyProfileHost(t *testing.T) {
	for _, tt := range []struct{ name, profile, environment, want string }{
		{"profile host", "git.example.com", "", "git.example.com"},
		{"explicit environment wins", "git.example.com", "other.example.com", "other.example.com"},
		{"empty profile", "", "github.com", "github.com"},
		{"no host", "", "", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GH_HOST", tt.environment)
			restore, err := applyProfileHost(tt.profile)
			if err != nil {
				t.Fatal(err)
			}
			if got := os.Getenv("GH_HOST"); got != tt.want {
				t.Errorf("GH_HOST = %q, want %q", got, tt.want)
			}
			restore()
			if got := os.Getenv("GH_HOST"); got != tt.environment {
				t.Errorf("restored GH_HOST = %q, want %q", got, tt.environment)
			}
		})
	}
}

func TestApplyProfileHostRestoresUnsetVariable(t *testing.T) {
	t.Setenv("GH_HOST", "")
	if err := os.Unsetenv("GH_HOST"); err != nil {
		t.Fatal(err)
	}
	restore, err := applyProfileHost("git.example.com")
	if err != nil {
		t.Fatal(err)
	}
	restore()
	if _, exists := os.LookupEnv("GH_HOST"); exists {
		t.Fatal("GH_HOST should be unset after restore")
	}
}
