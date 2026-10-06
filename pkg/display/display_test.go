package display

import (
	"bytes"
	"testing"
)

func TestDisplayOffers_NilAndMissingFields(t *testing.T) {
	// Must not panic on unexpected, nil, or missing fields
	offers := []map[string]interface{}{
		{
			"id":             12345,
			"cuda_max_good":  12.4,
			"num_gpus":       float64(1), // float instead of int
			"gpu_name":       "RTX 4090",
			"score":          nil, // nil field
			"reliability":    0.99,
			"inet_up":        1000.0,
			"inet_down":      2000.0,
			"direct_port_count": 50,
			"duration":       86400.0,
		},
		{}, // Completely empty row
	}

	var buf bytes.Buffer
	err := DisplayOffers(&buf, offers, false)
	if err != nil {
		t.Fatalf("DisplayOffers failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Errorf("expected non-empty output")
	}

	// Test raw JSON output
	var jsonBuf bytes.Buffer
	err = DisplayOffers(&jsonBuf, offers, true)
	if err != nil {
		t.Fatalf("DisplayOffers (raw) failed: %v", err)
	}
}

func TestDisplayInstances_NilAndMissingFields(t *testing.T) {
	instances := []map[string]interface{}{
		{
			"id":              9999,
			"intended_status": "running",
			"actual_status":   "running",
			"gpu_name":        "RTX 3090",
			"num_gpus":        1,
			"ssh_host":        "1.2.3.4",
			"ssh_port":        float64(2222),
		},
		{}, // Empty row
	}

	var buf bytes.Buffer
	err := DisplayInstances(&buf, instances, false)
	if err != nil {
		t.Fatalf("DisplayInstances failed: %v", err)
	}
}
