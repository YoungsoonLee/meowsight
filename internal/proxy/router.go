package proxy

import (
	"fmt"
	"log/slog"
	"net/http"
)

// Router multiplexes requests to the correct LLM provider handler.
type Router struct {
	mux      *http.ServeMux
	emitter  EventEmitter
}

// NewRouter creates a proxy router with the given event emitter.
func NewRouter(emitter EventEmitter) *Router {
	return &Router{
		mux:     http.NewServeMux(),
		emitter: emitter,
	}
}

// RegisterProvider mounts a provider handler at /providerName/.
func (r *Router) RegisterProvider(name string, handler http.Handler) {
	pattern := fmt.Sprintf("/%s/", name)
	r.mux.Handle(pattern, handler)
	slog.Info("registered provider", "provider", name, "pattern", pattern)
}

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Health check
	if req.URL.Path == "/healthz" {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
		return
	}

	r.mux.ServeHTTP(w, req)
}
