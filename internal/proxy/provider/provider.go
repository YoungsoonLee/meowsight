package provider

import "net/http"

// Provider defines a reverse proxy handler for a specific LLM API.
type Provider interface {
	// Name returns the provider identifier (e.g. "openai", "anthropic").
	Name() string
	// Handler returns the HTTP handler that proxies requests to this provider.
	Handler() http.Handler
}
