package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/takayoshi/vast-cli/pkg/client"
	"github.com/takayoshi/vast-cli/pkg/query"
)

// SearchTemplates searches public and private templates
func SearchTemplates(ctx context.Context, c *client.Client, queryStr string) ([]map[string]interface{}, error) {
	q := make(map[string]interface{})
	if queryStr != "" {
		parsed, err := query.ParseQuery(queryStr)
		if err == nil {
			for k, v := range parsed {
				q[k] = v
			}
		} else {
			// If not a full query, filter by name
			q["name"] = map[string]interface{}{"eq": queryStr}
		}
	}

	data, err := c.Post(ctx, "/templates/", url.Values{}, q)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Templates []map[string]interface{} `json:"templates"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		// Try raw array
		var arr []map[string]interface{}
		if err2 := json.Unmarshal(data, &arr); err2 == nil {
			return arr, nil
		}
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}
	return resp.Templates, nil
}

// SearchVolumes searches volume offers
func SearchVolumes(ctx context.Context, c *client.Client, queryStr string) ([]map[string]interface{}, error) {
	q := make(map[string]interface{})
	if queryStr != "" {
		parsed, err := query.ParseQuery(queryStr)
		if err == nil {
			for k, v := range parsed {
				q[k] = v
			}
		}
	}

	data, err := c.Post(ctx, "/volumes/", url.Values{}, q)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Volumes []map[string]interface{} `json:"volumes"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse volumes: %w", err)
	}
	return resp.Volumes, nil
}

// ShowVolumes lists user's volumes
func ShowVolumes(ctx context.Context, c *client.Client) ([]map[string]interface{}, error) {
	data, err := c.Get(ctx, "/users/current/volumes/", url.Values{})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Volumes []map[string]interface{} `json:"volumes"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse user volumes: %w", err)
	}
	return resp.Volumes, nil
}
