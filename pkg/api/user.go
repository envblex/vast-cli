package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/takayoshi/vast-cli/pkg/client"
)

// ShowUser retrieves current user profile and balance
func ShowUser(ctx context.Context, c *client.Client) (map[string]interface{}, error) {
	data, err := c.Get(ctx, "/users/current", url.Values{})
	if err != nil {
		return nil, err
	}

	var user map[string]interface{}
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("failed to parse user response: %w", err)
	}
	delete(user, "api_key")
	return user, nil
}
