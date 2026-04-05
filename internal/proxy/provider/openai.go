package provider

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/YoungsoonLee/meowsight/internal/proxy"
)

// OpenAI implements a reverse proxy for the OpenAI API.
type OpenAI struct {
	name        string
	prefix      string
	target      *url.URL
	pricing     *proxy.PricingTable
	emitter     proxy.EventEmitter
	keyResolver *proxy.KeyResolver
}

func NewOpenAI(name, baseURL string, pricing *proxy.PricingTable, emitter proxy.EventEmitter) *OpenAI {
	u, _ := url.Parse(baseURL)
	return &OpenAI{name: name, prefix: "/" + name, target: u, pricing: pricing, emitter: emitter}
}

// SetKeyResolver sets the API key resolver for key-based agent identification.
func (o *OpenAI) SetKeyResolver(kr *proxy.KeyResolver) { o.keyResolver = kr }

func (o *OpenAI) Name() string { return o.name }

func (o *OpenAI) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		tag := proxy.TagFromRequestWithKey(r, o.keyResolver)
		tenantID, agentID := tag.TenantID, tag.AgentID

		// Read request body to detect streaming and model
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		var reqBody struct {
			Model         string `json:"model"`
			Stream        bool   `json:"stream"`
			StreamOptions *struct {
				IncludeUsage bool `json:"include_usage"`
			} `json:"stream_options"`
		}
		json.Unmarshal(bodyBytes, &reqBody)

		// If streaming, inject stream_options.include_usage=true so we get token counts
		if reqBody.Stream && reqBody.StreamOptions == nil {
			var raw map[string]any
			if json.Unmarshal(bodyBytes, &raw) == nil {
				raw["stream_options"] = map[string]any{"include_usage": true}
				if modified, err := json.Marshal(raw); err == nil {
					bodyBytes = modified
					r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
					r.ContentLength = int64(len(bodyBytes))
				}
			}
		}

		// Strip the /openai prefix from the path
		r.URL.Path = strings.TrimPrefix(r.URL.Path, o.prefix)
		r.URL.Host = o.target.Host
		r.URL.Scheme = o.target.Scheme
		r.Host = o.target.Host

		// Remove MeowSight headers before forwarding
		r.Header.Del("X-Meowsight-Agent")
		r.Header.Del("X-Meowsight-Tenant")

		// Swap MeowSight API key with real upstream key if resolved via key-based auth
		if tag.UpstreamAPIKey != "" {
			r.Header.Set("Authorization", "Bearer "+tag.UpstreamAPIKey)
		}

		if reqBody.Stream {
			o.handleStreaming(w, r, tenantID, agentID, reqBody.Model, start)
		} else {
			o.handleNonStreaming(w, r, tenantID, agentID, reqBody.Model, start)
		}
	})
}

func (o *OpenAI) handleNonStreaming(w http.ResponseWriter, r *http.Request, tenantID, agentID, model string, start time.Time) {
	rp := &httputil.ReverseProxy{
		Director: func(req *http.Request) {},
		ModifyResponse: func(resp *http.Response) error {
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

			var result struct {
				Model string `json:"model"`
				Usage struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
				} `json:"usage"`
			}

			if json.Unmarshal(bodyBytes, &result) == nil && result.Usage.PromptTokens > 0 {
				m := result.Model
				if m == "" {
					m = model
				}
				o.emitter.Emit(proxy.RequestEvent{
					TenantID:     tenantID,
					AgentID:      agentID,
					Provider:     "openai",
					Model:        m,
					InputTokens:  result.Usage.PromptTokens,
					OutputTokens: result.Usage.CompletionTokens,
					CostUSD:      o.pricing.CalculateCost(m, result.Usage.PromptTokens, result.Usage.CompletionTokens),
					LatencyMs:    time.Since(start).Milliseconds(),
					StatusCode:   resp.StatusCode,
					Streaming:    false,
					Timestamp:    start,
				})
			}

			return nil
		},
	}
	rp.ServeHTTP(w, r)
}

func (o *OpenAI) handleStreaming(w http.ResponseWriter, r *http.Request, tenantID, agentID, model string, start time.Time) {
	// Make the upstream request ourselves for streaming
	client := &http.Client{Timeout: 5 * time.Minute}
	upstreamURL := o.target.Scheme + "://" + o.target.Host + r.URL.Path
	if r.URL.RawQuery != "" {
		upstreamURL += "?" + r.URL.RawQuery
	}

	upReq, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL, r.Body)
	if err != nil {
		http.Error(w, "failed to create upstream request", http.StatusInternalServerError)
		return
	}
	upReq.Header = r.Header.Clone()

	resp, err := client.Do(upReq)
	if err != nil {
		http.Error(w, "upstream request failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for k, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	flusher, ok := w.(http.Flusher)
	if !ok {
		io.Copy(w, resp.Body)
		return
	}

	// Stream SSE lines through, parsing for usage data
	var inputTokens, outputTokens int
	var respModel string

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		w.Write([]byte(line + "\n"))
		flusher.Flush()

		// Parse SSE data lines for usage
		if data, ok := strings.CutPrefix(line, "data: "); ok {
			if data == "[DONE]" {
				continue
			}
			var chunk struct {
				Model string `json:"model"`
				Usage *struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
				} `json:"usage"`
			}
			if json.Unmarshal([]byte(data), &chunk) == nil {
				if chunk.Model != "" {
					respModel = chunk.Model
				}
				if chunk.Usage != nil {
					inputTokens = chunk.Usage.PromptTokens
					outputTokens = chunk.Usage.CompletionTokens
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Error("streaming scan error", "error", err)
	}

	m := respModel
	if m == "" {
		m = model
	}

	o.emitter.Emit(proxy.RequestEvent{
		TenantID:     tenantID,
		AgentID:      agentID,
		Provider:     "openai",
		Model:        m,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		CostUSD:      o.pricing.CalculateCost(m, inputTokens, outputTokens),
		LatencyMs:    time.Since(start).Milliseconds(),
		StatusCode:   resp.StatusCode,
		Streaming:    true,
		Timestamp:    start,
	})
}
