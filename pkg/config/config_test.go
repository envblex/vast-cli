package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFormatKeySuffix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "(empty)"},
		{"abc", "abc"},
		{"abcdef", "...cdef"},
		{"1234", "...1234"},
	}
	for _, tt := range tests {
		got := FormatKeySuffix(tt.input)
		if got != tt.expected {
			t.Errorf("FormatKeySuffix(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestGetServerURL(t *testing.T) {
	if got := GetServerURL("https://custom.vast.ai/"); got != "https://custom.vast.ai" {
		t.Errorf("expected trimmed url, got %q", got)
	}

	orig := os.Getenv("VAST_URL")
	defer os.Setenv("VAST_URL", orig)

	os.Setenv("VAST_URL", "https://env.vast.ai/")
	if got := GetServerURL(""); got != "https://env.vast.ai" {
		t.Errorf("expected env url, got %q", got)
	}

	os.Unsetenv("VAST_URL")
	if got := GetServerURL(""); got != DefaultServerURL {
		t.Errorf("expected default url, got %q", got)
	}
}

func TestGetAPIKeyPriority(t *testing.T) {
	origEnv := os.Getenv("VAST_API_KEY")
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		os.Setenv("VAST_API_KEY", origEnv)
		os.Setenv("XDG_CONFIG_HOME", origXDG)
	}()

	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	vastDir := filepath.Join(tmpDir, "vastai")
	_ = os.MkdirAll(vastDir, 0700)
	_ = os.WriteFile(filepath.Join(vastDir, "vast_api_key"), []byte("file_key_1234"), 0600)

	// 1. CLI Override wins
	os.Setenv("VAST_API_KEY", "env_key")
	if got := GetAPIKey("cli_override"); got != "cli_override" {
		t.Errorf("expected cli_override, got %q", got)
	}

	// 2. Env wins over file
	if got := GetAPIKey(""); got != "env_key" {
		t.Errorf("expected env_key, got %q", got)
	}

	// 3. File wins when env is unset
	os.Unsetenv("VAST_API_KEY")
	if got := GetAPIKey(""); got != "file_key_1234" {
		t.Errorf("expected file_key_1234, got %q", got)
	}
}
