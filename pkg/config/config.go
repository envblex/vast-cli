package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultServerURL = "https://console.vast.ai"
	Version          = "1.8.3-go"
	OfficialVersion  = "1.8.3"
)

// GetConfigDir returns ~/.config/vastai or $XDG_CONFIG_HOME/vastai
func GetConfigDir() (string, error) {
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg != "" {
		return filepath.Join(xdg, "vastai"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "vastai"), nil
}

// GetAPIKey resolves the API key with standard fallback order:
// 1. CLI flag override
// 2. VAST_API_KEY environment variable
// 3. $XDG_CONFIG_HOME/vastai/vast_api_key (~/.config/vastai/vast_api_key)
// 4. Legacy ~/.vast_api_key
func GetAPIKey(override string) string {
	if strings.TrimSpace(override) != "" {
		return strings.TrimSpace(override)
	}

	if envKey := strings.TrimSpace(os.Getenv("VAST_API_KEY")); envKey != "" {
		return envKey
	}

	if cfgDir, err := GetConfigDir(); err == nil {
		path := filepath.Join(cfgDir, "vast_api_key")
		if data, err := os.ReadFile(path); err == nil {
			k := strings.TrimSpace(string(data))
			if k != "" {
				return k
			}
		}
	}

	if home, err := os.UserHomeDir(); err == nil {
		legacyPath := filepath.Join(home, ".vast_api_key")
		if data, err := os.ReadFile(legacyPath); err == nil {
			k := strings.TrimSpace(string(data))
			if k != "" {
				return k
			}
		}
	}

	return ""
}

// SetAPIKey saves the API key to ~/.config/vastai/vast_api_key
// and cleans up legacy file if present.
func SetAPIKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", fmt.Errorf("API key cannot be empty")
	}

	cfgDir, err := GetConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine config directory: %w", err)
	}

	if err := os.MkdirAll(cfgDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create config directory %s: %w", cfgDir, err)
	}

	keyFile := filepath.Join(cfgDir, "vast_api_key")
	if err := os.WriteFile(keyFile, []byte(key), 0600); err != nil {
		return "", fmt.Errorf("failed to write API key to %s: %w", keyFile, err)
	}

	// Remove legacy file if present
	if home, err := os.UserHomeDir(); err == nil {
		legacyFile := filepath.Join(home, ".vast_api_key")
		_ = os.Remove(legacyFile)
	}

	return keyFile, nil
}

// GetServerURL resolves the server URL from CLI override, env VAST_URL, or default
func GetServerURL(override string) string {
	if strings.TrimSpace(override) != "" {
		return strings.TrimRight(strings.TrimSpace(override), "/")
	}
	if envURL := strings.TrimSpace(os.Getenv("VAST_URL")); envURL != "" {
		return strings.TrimRight(envURL, "/")
	}
	return DefaultServerURL
}

// FormatKeySuffix returns the masked key ending in last 4 chars
func FormatKeySuffix(k string) string {
	if len(k) >= 4 {
		return "..." + k[len(k)-4:]
	}
	if k == "" {
		return "(empty)"
	}
	return k
}
