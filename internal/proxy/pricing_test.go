package proxy

import (
	"math"
	"testing"
)

func TestPricingTable_LoadFromFile(t *testing.T) {
	pt := NewPricingTable()
	err := pt.LoadFromFile("../../configs/pricing.json")
	if err != nil {
		t.Fatalf("failed to load pricing file: %v", err)
	}

	p, ok := pt.Get("gpt-4o")
	if !ok {
		t.Fatal("expected gpt-4o in pricing table")
	}
	if p.InputPer1K <= 0 {
		t.Errorf("expected positive input price, got %f", p.InputPer1K)
	}
}

func TestPricingTable_CalculateCost(t *testing.T) {
	pt := NewPricingTable()
	pt.LoadFromFile("../../configs/pricing.json")

	tests := []struct {
		name         string
		model        string
		input        int
		output       int
		wantPositive bool
	}{
		{"gpt-4o", "gpt-4o", 1000, 500, true},
		{"claude-sonnet", "claude-sonnet-4-0", 2000, 1000, true},
		{"unknown model", "unknown-model-xyz", 1000, 500, false},
		{"zero tokens", "gpt-4o", 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := pt.CalculateCost(tt.model, tt.input, tt.output)
			if tt.wantPositive && cost <= 0 {
				t.Errorf("expected positive cost for %s, got %f", tt.model, cost)
			}
			if !tt.wantPositive && cost != 0 {
				t.Errorf("expected zero cost, got %f", cost)
			}
		})
	}
}

func TestPricingTable_Accuracy(t *testing.T) {
	pt := NewPricingTable()
	pt.LoadFromFile("../../configs/pricing.json")

	// gpt-4o: input $0.0025/1K, output $0.01/1K
	// 1000 input + 500 output = $0.0025 + $0.005 = $0.0075
	cost := pt.CalculateCost("gpt-4o", 1000, 500)
	expected := 0.0075
	if math.Abs(cost-expected) > 0.0001 {
		t.Errorf("expected cost %f, got %f", expected, cost)
	}
}

func TestPricingTable_UnknownModel(t *testing.T) {
	pt := NewPricingTable()
	_, ok := pt.Get("nonexistent")
	if ok {
		t.Error("expected false for unknown model")
	}
}

func TestPricingTable_FileNotFound(t *testing.T) {
	pt := NewPricingTable()
	err := pt.LoadFromFile("/nonexistent/path.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
