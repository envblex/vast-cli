package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/takayoshi/vast-cli/pkg/client"
	"github.com/takayoshi/vast-cli/pkg/query"
)

type SearchOffersOptions struct {
	QueryStr        string
	Type            string // "on-demand", "reserved", "bid"
	Order           string // e.g. "dlperf_usd-"
	Limit           int
	Storage         float64 // default 5.0
	NoDefault       bool
	DisableBundling bool
}

// SearchOffers queries available GPU offers on Vast.ai
func SearchOffers(ctx context.Context, c *client.Client, opts SearchOffersOptions) ([]map[string]interface{}, error) {
	q := make(map[string]interface{})

	if !opts.NoDefault {
		q["verified"] = map[string]interface{}{"eq": true}
		q["external"] = map[string]interface{}{"eq": false}
		q["rentable"] = map[string]interface{}{"eq": true}
		q["rented"] = map[string]interface{}{"eq": false}
	}

	if opts.QueryStr != "" {
		parsedQuery, err := query.ParseQuery(opts.QueryStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse query: %w", err)
		}
		for k, v := range parsedQuery {
			q[k] = v
		}
	}

	order := query.ParseOrder(opts.Order)
	q["order"] = order

	offerType := opts.Type
	if offerType == "" {
		offerType = "on-demand"
	} else if offerType == "interruptible" {
		offerType = "bid"
	}
	q["type"] = offerType

	if opts.Limit > 0 {
		q["limit"] = opts.Limit
	}

	storage := opts.Storage
	if storage <= 0 {
		storage = 5.0
	}
	q["allocated_storage"] = storage

	if opts.DisableBundling {
		q["disable_bundling"] = true
	}

	data, err := c.Post(ctx, "/bundles/", url.Values{}, q)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Offers []map[string]interface{} `json:"offers"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode offers response: %w", err)
	}

	return resp.Offers, nil
}
