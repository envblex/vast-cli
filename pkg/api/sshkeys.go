package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/takayoshi/vast-cli/pkg/client"
)

// ShowSSHKeys fetches account SSH keys
func ShowSSHKeys(ctx context.Context, c *client.Client) ([]map[string]interface{}, error) {
	data, err := c.Get(ctx, "/ssh-keys/", url.Values{})
	if err != nil {
		return nil, err
	}

	var keys []map[string]interface{}
	if err := json.Unmarshal(data, &keys); err != nil {
		return nil, fmt.Errorf("failed to parse ssh-keys: %w", err)
	}
	return keys, nil
}

// CreateSSHKey registers a new SSH key
func CreateSSHKey(ctx context.Context, c *client.Client, keyOrPath string) (map[string]interface{}, error) {
	keyContent := strings.TrimSpace(keyOrPath)

	// If path like ~/.ssh/id_rsa.pub or existing file
	if strings.HasPrefix(keyContent, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			keyContent = strings.Replace(keyContent, "~", home, 1)
		}
	}

	if _, err := os.Stat(keyContent); err == nil {
		data, err := os.ReadFile(keyContent)
		if err != nil {
			return nil, fmt.Errorf("failed to read SSH key file: %w", err)
		}
		keyContent = strings.TrimSpace(string(data))
	}

	if keyContent == "" {
		return nil, fmt.Errorf("SSH key content cannot be empty")
	}

	payload := map[string]string{"ssh_key": keyContent}
	data, err := c.Post(ctx, "/ssh-keys/", url.Values{}, payload)
	if err != nil {
		return nil, err
	}

	var res map[string]interface{}
	_ = json.Unmarshal(data, &res)
	return res, nil
}

// DeleteSSHKey removes an SSH key by ID
func DeleteSSHKey(ctx context.Context, c *client.Client, id int64) (map[string]interface{}, error) {
	data, err := c.Delete(ctx, fmt.Sprintf("/ssh-keys/%d/", id), url.Values{}, map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	var res map[string]interface{}
	_ = json.Unmarshal(data, &res)
	return res, nil
}
