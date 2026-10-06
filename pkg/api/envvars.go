package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/takayoshi/vast-cli/pkg/client"
)

// ShowEnvVars fetches user environment secrets
func ShowEnvVars(ctx context.Context, c *client.Client) (map[string]string, error) {
	data, err := c.Get(ctx, "/secrets/", url.Values{})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Secrets map[string]string `json:"secrets"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse env vars: %w", err)
	}
	return resp.Secrets, nil
}

// CreateEnvVar creates or updates a secret
func CreateEnvVar(ctx context.Context, c *client.Client, name, value string) (map[string]interface{}, error) {
	payload := map[string]string{"key": name, "value": value}
	data, err := c.Post(ctx, "/secrets/", url.Values{}, payload)
	if err != nil {
		return nil, err
	}
	var res map[string]interface{}
	_ = json.Unmarshal(data, &res)
	return res, nil
}

// DeleteEnvVar deletes a secret
func DeleteEnvVar(ctx context.Context, c *client.Client, name string) (map[string]interface{}, error) {
	payload := map[string]string{"key": name}
	data, err := c.Delete(ctx, "/secrets/", url.Values{}, payload)
	if err != nil {
		return nil, err
	}
	var res map[string]interface{}
	_ = json.Unmarshal(data, &res)
	return res, nil
}
