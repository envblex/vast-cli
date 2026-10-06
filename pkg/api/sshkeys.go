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
	data, err := c.Get(ctx, "/ssh/", url.Values{})
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
	data, err := c.Post(ctx, "/ssh/", url.Values{}, payload)
	if err != nil {
		return nil, err
	}

	var res map[string]interface{}
	_ = json.Unmarshal(data, &res)
	return res, nil
}

// DeleteSSHKey removes an SSH key by ID
func DeleteSSHKey(ctx context.Context, c *client.Client, id int64) (map[string]interface{}, error) {
	data, err := c.Delete(ctx, fmt.Sprintf("/ssh/%d/", id), url.Values{}, map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	var res map[string]interface{}
	_ = json.Unmarshal(data, &res)
	return res, nil
}

// AttachSSH attaches an SSH public key to a specific running instance
func AttachSSH(ctx context.Context, c *client.Client, instanceID int64, sshKey string) (map[string]interface{}, error) {
	payload := map[string]string{"ssh_key": strings.TrimSpace(sshKey)}
	data, err := c.Post(ctx, fmt.Sprintf("/instances/%d/ssh/", instanceID), url.Values{}, payload)
	if err != nil {
		return nil, err
	}
	var res map[string]interface{}
	_ = json.Unmarshal(data, &res)
	return res, nil
}

// DetachSSH detaches an SSH key from a specific instance
func DetachSSH(ctx context.Context, c *client.Client, instanceID int64, sshKeyID string) (map[string]interface{}, error) {
	data, err := c.Delete(ctx, fmt.Sprintf("/instances/%d/ssh/%s/", instanceID, sshKeyID), url.Values{}, map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	var res map[string]interface{}
	_ = json.Unmarshal(data, &res)
	return res, nil
}

// ShowAPIKeys lists all API keys
func ShowAPIKeys(ctx context.Context, c *client.Client) ([]map[string]interface{}, error) {
	data, err := c.Get(ctx, "/auth/apikeys/", url.Values{})
	if err != nil {
		return nil, err
	}

	var resp struct {
		APIKeys []map[string]interface{} `json:"apikeys"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		var list []map[string]interface{}
		if err2 := json.Unmarshal(data, &list); err2 == nil {
			return list, nil
		}
		return nil, fmt.Errorf("failed to parse api-keys: %w", err)
	}
	return resp.APIKeys, nil
}

// CreateAPIKey creates a new API key on Vast.ai
func CreateAPIKey(ctx context.Context, c *client.Client, name string, permissions map[string]interface{}) (map[string]interface{}, error) {
	payload := map[string]interface{}{"name": name}
	if permissions != nil {
		payload["permissions"] = permissions
	}
	data, err := c.Post(ctx, "/auth/apikeys/", url.Values{}, payload)
	if err != nil {
		return nil, err
	}
	var res map[string]interface{}
	_ = json.Unmarshal(data, &res)
	return res, nil
}

// DeleteAPIKey removes an API key by ID
func DeleteAPIKey(ctx context.Context, c *client.Client, id int64) (map[string]interface{}, error) {
	data, err := c.Delete(ctx, fmt.Sprintf("/auth/apikeys/%d/", id), url.Values{}, map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	var res map[string]interface{}
	_ = json.Unmarshal(data, &res)
	return res, nil
}

// ResetAPIKey resets the account's master API key
func ResetAPIKey(ctx context.Context, c *client.Client) (map[string]interface{}, error) {
	data, err := c.Put(ctx, "/commands/reset_apikey/", url.Values{}, map[string]string{"client_id": "me"})
	if err != nil {
		return nil, err
	}
	var res map[string]interface{}
	_ = json.Unmarshal(data, &res)
	return res, nil
}
