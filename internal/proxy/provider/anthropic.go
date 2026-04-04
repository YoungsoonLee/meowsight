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

// Anthropic implements a reverse proxy for the Anthropic API.
type Anthropic struct {
	name    string
	prefix  string
	target  *url.URL
	pricing *proxy.PricingTable
	emitter proxy.EventEmitter
}

func NewAnthropic(name, baseURL string, pricing *proxy.PricingTable, emitter proxy.EventEmitter) *Anthropic {
	u, _ := url.Parse(baseURL)
	return &Anthropic{name: name, prefix: "/" + name, target: u, pricing: pricing, emitter: emitter}
}

func (a *Anthropic) Name() string { return a.name }

func (a *Anthropic) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		tenantID, agentID := proxy.TagFromRequest(r)

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		var reqBody struct {
			Model  string `json:"model"`
			Stream bool   `json:"stream"`
		}
		json.Unmarshal(bodyBytes, &reqBody)

		// Strip the /anthropic prefix
		r.URL.Path = strings.TrimPrefix(r.URL.Path, a.prefix)
		r.URL.Host = a.target.Host
		r.URL.Scheme = a.target.Scheme
		r.Host = a.target.Host

		r.Header.Del("X-Meowsight-Agent")
		r.Header.Del("X-Meowsight-Tenant")

		if reqBody.Stream {
			a.handleStreaming(w, r, tenantID, agentID, reqBody.Model, start)
		} else {
			a.handleNonStreaming(w, r, tenantID, agentID, reqBody.Model, start)
		}
	})
}

// anthropicUsage is shared between streaming and non-streaming responses.
type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

func (a *Anthropic) handleNonStreaming(w http.ResponseWriter, r *http.Request, tenantID, agentID, model string, start time.Time) {
	rp := &httputil.ReverseProxy{
		Director: func(req *http.Request) {},
		ModifyResponse: func(resp *http.Response) error {
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

			var result struct {
				Model string         `json:"model"`
				Usage anthropicUsage `json:"usage"`
			}

			if json.Unmarshal(bodyBytes, &result) == nil && result.Usage.InputTokens > 0 {
				m := result.Model
				if m == "" {
					m = model
				}
				a.emitter.Emit(proxy.RequestEvent{
					TenantID:     tenantID,
					AgentID:      agentID,
					Provider:     "anthropic",
					Model:        m,
					InputTokens:  result.Usage.InputTokens,
					OutputTokens: result.Usage.OutputTokens,
					CostUSD:      a.pricing.CalculateCost(m, result.Usage.InputTokens, result.Usage.OutputTokens),
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

func (a *Anthropic) handleStreaming(w http.ResponseWriter, r *http.Request, tenantID, agentID, model string, start time.Time) {
	client := &http.Client{Timeout: 5 * time.Minute}
	upstreamURL := a.target.Scheme + "://" + a.target.Host + r.URL.Path
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

	// Anthropic streaming format:
	// event: message_start     → has usage.input_tokens
	// event: content_block_delta
	// event: message_delta     → has usage.output_tokens
	// event: message_stop

	var inputTokens, outputTokens int
	var respModel string

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var currentEvent string

	for scanner.Scan() {
		line := scanner.Text()
		w.Write([]byte(line + "\n"))
		flusher.Flush()

		if ev, ok := strings.CutPrefix(line, "event: "); ok {
			currentEvent = ev
			continue
		}

		if data, ok := strings.CutPrefix(line, "data: "); ok {

			switch currentEvent {
			case "message_start":
				var msg struct {
					Message struct {
						Model string         `json:"model"`
						Usage anthropicUsage `json:"usage"`
					} `json:"message"`
				}
				if json.Unmarshal([]byte(data), &msg) == nil {
					respModel = msg.Message.Model
					inputTokens = msg.Message.Usage.InputTokens
				}

			case "message_delta":
				var delta struct {
					Usage struct {
						OutputTokens int `json:"output_tokens"`
					} `json:"usage"`
				}
				if json.Unmarshal([]byte(data), &delta) == nil {
					outputTokens = delta.Usage.OutputTokens
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

	a.emitter.Emit(proxy.RequestEvent{
		TenantID:     tenantID,
		AgentID:      agentID,
		Provider:     "anthropic",
		Model:        m,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		CostUSD:      a.pricing.CalculateCost(m, inputTokens, outputTokens),
		LatencyMs:    time.Since(start).Milliseconds(),
		StatusCode:   resp.StatusCode,
		Streaming:    true,
		Timestamp:    start,
	})
}
