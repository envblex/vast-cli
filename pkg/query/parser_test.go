package query

import (
	"reflect"
	"testing"
)

func TestParseQuery(t *testing.T) {
	q, err := ParseQuery("gpu_name=RTX_4090 num_gpus>=1 verified=true dph<=2.0")
	if err != nil {
		t.Fatalf("ParseQuery failed: %v", err)
	}

	if q["gpu_name"]["eq"] != "RTX 4090" {
		t.Errorf("expected gpu_name eq 'RTX 4090', got %v", q["gpu_name"]["eq"])
	}
	if q["num_gpus"]["gte"] != int64(1) {
		t.Errorf("expected num_gpus gte 1, got %v", q["num_gpus"]["gte"])
	}
	if q["verified"]["eq"] != true {
		t.Errorf("expected verified eq true, got %v", q["verified"]["eq"])
	}
	if q["dph_total"]["lte"] != 2.0 {
		t.Errorf("expected dph_total lte 2.0, got %v", q["dph_total"]["lte"])
	}
}

func TestParseOrder(t *testing.T) {
	ord := ParseOrder("dlperf_usd-,dph")
	expected := [][]string{
		{"dlperf_per_dphtotal", "desc"},
		{"dph_total", "asc"},
	}
	if !reflect.DeepEqual(ord, expected) {
		t.Errorf("expected %v, got %v", expected, ord)
	}
}
