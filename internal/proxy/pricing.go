package proxy

import (
	"encoding/json"
	"log/slog"
	"os"
	"sync"
)

// ModelPricing holds per-1K-token pricing in USD.
type ModelPricing struct {
	Provider    string  `json:"provider"`
	InputPer1K  float64 `json:"input_per_1k"`
	OutputPer1K float64 `json:"output_per_1k"`
}

// PricingTable provides thread-safe access to model pricing data.
// Load from a JSON file at startup; reload without restart when prices change.
type PricingTable struct {
	mu     sync.RWMutex
	models map[string]ModelPricing
}

// pricingFile is the JSON structure on disk.
type pricingFile struct {
	Models map[string]ModelPricing `json:"models"`
}

// NewPricingTable creates an empty pricing table.
func NewPricingTable() *PricingTable {
	return &PricingTable{models: make(map[string]ModelPricing)}
}

// LoadFromFile reads pricing data from a JSON file.
func (pt *PricingTable) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var f pricingFile
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}

	pt.mu.Lock()
	pt.models = f.Models
	pt.mu.Unlock()

	slog.Info("pricing table loaded", "path", path, "models", len(f.Models))
	return nil
}

// Get returns the pricing for a model. Returns zero pricing if not found.
func (pt *PricingTable) Get(model string) (ModelPricing, bool) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	p, ok := pt.models[model]
	return p, ok
}

// CalculateCost returns estimated cost in USD for the given model and token counts.
func (pt *PricingTable) CalculateCost(model string, inputTokens, outputTokens int) float64 {
	p, ok := pt.Get(model)
	if !ok {
		return 0
	}
	return (float64(inputTokens)/1000)*p.InputPer1K + (float64(outputTokens)/1000)*p.OutputPer1K
}
