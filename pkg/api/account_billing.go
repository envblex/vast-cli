package api

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/takayoshi/vast-cli/pkg/client"
	"github.com/takayoshi/vast-cli/pkg/query"
)

// ShowAuditLogs lists account audit log history
func ShowAuditLogs(ctx context.Context, c *client.Client) ([]map[string]interface{}, error) {
	data, err := c.Get(ctx, "/audit_logs/", url.Values{})
	if err != nil {
		return nil, err
	}
	var logs []map[string]interface{}
	if err := json.Unmarshal(data, &logs); err != nil {
		return nil, err
	}
	return logs, nil
}

// ShowConnections lists cloud storage connections
func ShowConnections(ctx context.Context, c *client.Client) ([]map[string]interface{}, error) {
	data, err := c.Get(ctx, "/connections/", url.Values{})
	if err != nil {
		return nil, err
	}
	var res []map[string]interface{}
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return res, nil
}

// ShowIPAddrs lists IP addresses associated with account
func ShowIPAddrs(ctx context.Context, c *client.Client) ([]map[string]interface{}, error) {
	data, err := c.Get(ctx, "/users/ipaddrs/", url.Values{})
	if err != nil {
		return nil, err
	}
	var res []map[string]interface{}
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return res, nil
}

// ShowInvoicesV1 returns invoice records
func ShowInvoicesV1(ctx context.Context, c *client.Client, queryParams url.Values) (map[string]interface{}, error) {
	data, err := c.Get(ctx, "/api/v1/invoices/", queryParams)
	if err != nil {
		return nil, err
	}
	var res map[string]interface{}
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return res, nil
}

// SearchBenchmarks searches benchmark results
func SearchBenchmarks(ctx context.Context, c *client.Client, queryStr string) ([]map[string]interface{}, error) {
	q := make(map[string]interface{})
	if queryStr != "" {
		if parsed, err := query.ParseQuery(queryStr); err == nil {
			for k, v := range parsed {
				q[k] = v
			}
		}
	}
	data, err := c.Post(ctx, "/benchmarks/", url.Values{}, q)
	if err != nil {
		return nil, err
	}
	var res []map[string]interface{}
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return res, nil
}
