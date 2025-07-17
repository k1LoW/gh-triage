package profile

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/goccy/go-yaml"
)

// testLogHandler is a custom log handler for testing.
type testLogHandler struct {
	messages *[]string
}

func (h *testLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (h *testLogHandler) Handle(ctx context.Context, record slog.Record) error {
	*h.messages = append(*h.messages, record.Message)
	return nil
}

func (h *testLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *testLogHandler) WithGroup(name string) slog.Handler {
	return h
}

func TestLoad_NewInstall(t *testing.T) {
	// Setup test environment
	tempDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tempDir)

	// Test loading with empty profile (should create default.yml)
	profile, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Check that default profile is returned
	if profile.Read.Max != defaultProfile.Read.Max {
		t.Errorf("Expected Read.Max=%d, got %d", defaultProfile.Read.Max, profile.Read.Max)
	}
	if len(profile.Read.Conditions) != len(defaultProfile.Read.Conditions) {
		t.Errorf("Expected Read.Conditions length=%d, got %d", len(defaultProfile.Read.Conditions), len(profile.Read.Conditions))
	}

	// Check that default.yml was created
	expectedPath := filepath.Join(tempDir, "gh-triage", "default.yml")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected config file to be created at %s", expectedPath)
	}

	// Verify the file content
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read created config file: %v", err)
	}
	var savedProfile Profile
	if err := yaml.Unmarshal(data, &savedProfile); err != nil {
		t.Fatalf("Failed to unmarshal saved profile: %v", err)
	}
	if savedProfile.Read.Max != defaultProfile.Read.Max {
		t.Errorf("Saved profile Read.Max=%d, expected %d", savedProfile.Read.Max, defaultProfile.Read.Max)
	}
}

func TestLoad_Migration(t *testing.T) {
	// Setup test environment
	tempDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tempDir)

	// Create old config.yml file
	triageDir := filepath.Join(tempDir, "gh-triage")
	if err := os.MkdirAll(triageDir, 0700); err != nil {
		t.Fatalf("Failed to create triage directory: %v", err)
	}

	// Create a custom profile to test migration
	customProfile := &Profile{
		Read: Action{
			Max:        500,
			Conditions: []string{"custom_condition"},
		},
		Open: Action{
			Max:        2,
			Conditions: []string{"custom_open_condition"},
		},
		List: Action{
			Max:        100,
			Conditions: []string{"custom_list_condition"},
		},
	}

	customData, err := yaml.Marshal(customProfile)
	if err != nil {
		t.Fatalf("Failed to marshal custom profile: %v", err)
	}

	oldConfigPath := filepath.Join(triageDir, "config.yml")
	if err := os.WriteFile(oldConfigPath, customData, 0600); err != nil {
		t.Fatalf("Failed to create old config file: %v", err)
	}

	// Capture logs to verify migration message
	var logMessages []string
	originalHandler := slog.Default().Handler()
	t.Cleanup(func() {
		// Ensure log handler is restored even if test fails
		slog.SetDefault(slog.New(originalHandler))
	})

	testHandler := &testLogHandler{messages: &logMessages}
	slog.SetDefault(slog.New(testHandler))

	// Test loading with empty profile (should migrate config.yml to default.yml)
	profile, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Check that migrated profile is returned
	if profile.Read.Max != 500 {
		t.Errorf("Expected Read.Max=500, got %d", profile.Read.Max)
	}
	if len(profile.Read.Conditions) != 1 || profile.Read.Conditions[0] != "custom_condition" {
		t.Errorf("Expected Read.Conditions=[custom_condition], got %v", profile.Read.Conditions)
	}

	// Check that default.yml was created
	defaultPath := filepath.Join(triageDir, "default.yml")
	if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
		t.Errorf("Expected default.yml to be created at %s", defaultPath)
	}

	// Check that old config.yml was removed
	if _, err := os.Stat(oldConfigPath); !os.IsNotExist(err) {
		t.Errorf("Expected old config.yml to be removed")
	}

	// Check that migration log message was recorded
	found := false
	for _, msg := range logMessages {
		if msg == "migrated config file" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected migration log message, got: %v", logMessages)
	}
}

func TestLoad_WithProfile(t *testing.T) {
	// Setup test environment
	tempDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tempDir)

	// Test loading with specific profile
	profileName := "test-profile"
	profile, err := Load(profileName)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Check that default profile is returned (since no profile file exists)
	if profile.Read.Max != defaultProfile.Read.Max {
		t.Errorf("Expected Read.Max=%d, got %d", defaultProfile.Read.Max, profile.Read.Max)
	}

	// Check that profile-specific file was created
	expectedPath := filepath.Join(tempDir, "gh-triage", "test-profile.yml")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected profile config file to be created at %s", expectedPath)
	}

	// Test that no migration happens for profile (create old config.yml)
	triageDir := filepath.Join(tempDir, "gh-triage")
	oldConfigPath := filepath.Join(triageDir, "config.yml")
	if err := os.WriteFile(oldConfigPath, []byte("test: data"), 0600); err != nil {
		t.Fatalf("Failed to create old config file: %v", err)
	}

	// Load with profile again - should not affect the old config.yml
	profile2, err := Load(profileName)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Check that old config.yml still exists (no migration for profiles)
	if _, err := os.Stat(oldConfigPath); os.IsNotExist(err) {
		t.Errorf("Expected old config.yml to still exist when using profile")
	}

	// Check that profile config is still returned
	if profile2.Read.Max != defaultProfile.Read.Max {
		t.Errorf("Expected Read.Max=%d, got %d", defaultProfile.Read.Max, profile2.Read.Max)
	}
}

func TestLoad_ExistingFile(t *testing.T) {
	// Setup test environment
	tempDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tempDir)

	// Create existing config file with custom settings
	triageDir := filepath.Join(tempDir, "gh-triage")
	if err := os.MkdirAll(triageDir, 0700); err != nil {
		t.Fatalf("Failed to create triage directory: %v", err)
	}

	existingProfile := &Profile{
		Read: Action{
			Max:        750,
			Conditions: []string{"existing_condition_1", "existing_condition_2"},
		},
		Open: Action{
			Max:        3,
			Conditions: []string{"existing_open_condition"},
		},
		List: Action{
			Max:        200,
			Conditions: []string{"existing_list_condition"},
		},
	}

	existingData, err := yaml.Marshal(existingProfile)
	if err != nil {
		t.Fatalf("Failed to marshal existing profile: %v", err)
	}

	configPath := filepath.Join(triageDir, "default.yml")
	if err := os.WriteFile(configPath, existingData, 0600); err != nil {
		t.Fatalf("Failed to create existing config file: %v", err)
	}

	// Test loading existing config
	profile, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Check that existing config is returned
	if profile.Read.Max != 750 {
		t.Errorf("Expected Read.Max=750, got %d", profile.Read.Max)
	}
	if len(profile.Read.Conditions) != 2 {
		t.Errorf("Expected Read.Conditions length=2, got %d", len(profile.Read.Conditions))
	}
	if profile.Read.Conditions[0] != "existing_condition_1" || profile.Read.Conditions[1] != "existing_condition_2" {
		t.Errorf("Expected Read.Conditions=[existing_condition_1, existing_condition_2], got %v", profile.Read.Conditions)
	}

	// Test loading existing profile config
	profileConfigPath := filepath.Join(triageDir, "myprofile.yml")
	if err := os.WriteFile(profileConfigPath, existingData, 0600); err != nil {
		t.Fatalf("Failed to create existing profile config file: %v", err)
	}

	profileConfig, err := Load("myprofile")
	if err != nil {
		t.Fatalf("Load failed for profile: %v", err)
	}

	// Check that existing profile config is returned
	if profileConfig.Read.Max != 750 {
		t.Errorf("Expected profile Read.Max=750, got %d", profileConfig.Read.Max)
	}
	if profileConfig.Open.Max != 3 {
		t.Errorf("Expected profile Open.Max=3, got %d", profileConfig.Open.Max)
	}
	if profileConfig.List.Max != 200 {
		t.Errorf("Expected profile List.Max=200, got %d", profileConfig.List.Max)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	// Setup test environment
	tempDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tempDir)

	// Create existing config file with invalid YAML
	triageDir := filepath.Join(tempDir, "gh-triage")
	if err := os.MkdirAll(triageDir, 0700); err != nil {
		t.Fatalf("Failed to create triage directory: %v", err)
	}

	configPath := filepath.Join(triageDir, "default.yml")
	invalidYAML := "invalid: yaml: content: ["
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0600); err != nil {
		t.Fatalf("Failed to create invalid config file: %v", err)
	}

	// Test loading invalid config should return error
	_, err := Load("")
	if err == nil {
		t.Error("Expected error when loading invalid YAML, got nil")
	}
}
