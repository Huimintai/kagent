// Package aicoreproxy implements a lightweight reverse proxy that translates
// Anthropic/OpenAI API requests to SAP AI Core endpoints, handling OAuth2
// token management and deployment URL resolution.
package aicoreproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// Config holds the proxy configuration, typically sourced from environment variables.
type Config struct {
	Port          int    // Listen port (default 9090)
	Provider      string // "anthropic" or "openai"
	BaseURL       string // SAP AI Core API base URL
	AuthURL       string // OAuth2 token endpoint
	ClientID      string // OAuth2 client ID
	ClientSecret  string // OAuth2 client secret
	ResourceGroup string // AI Core resource group (default "default")
	Model         string // Model name for deployment lookup
}

// Proxy is the main proxy server.
type Proxy struct {
	config     Config
	token      *TokenProvider
	deployment *DeploymentResolver
	log        *slog.Logger
}

// New creates a new Proxy instance.
func New(cfg Config, log *slog.Logger) (*Proxy, error) {
	if cfg.Port == 0 {
		cfg.Port = 9090
	}
	if cfg.ResourceGroup == "" {
		cfg.ResourceGroup = "default"
	}
	if cfg.Provider == "" {
		return nil, fmt.Errorf("AICORE_PROXY_PROVIDER must be set to 'anthropic' or 'openai'")
	}
	if cfg.BaseURL == "" || cfg.AuthURL == "" {
		return nil, fmt.Errorf("SAP_AI_CORE_BASE_URL and SAP_AI_CORE_AUTH_URL are required")
	}
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("SAP_AI_CORE_CLIENT_ID and SAP_AI_CORE_CLIENT_SECRET are required")
	}

	tp := NewTokenProvider(cfg.AuthURL, cfg.ClientID, cfg.ClientSecret, log)
	dr := NewDeploymentResolver(cfg.BaseURL, cfg.ResourceGroup, tp, log)

	return &Proxy{
		config:     cfg,
		token:      tp,
		deployment: dr,
		log:        log,
	}, nil
}

// ListenAndServe starts the proxy HTTP server.
func (p *Proxy) ListenAndServe(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", p.handleHealth)

	switch p.config.Provider {
	case "anthropic":
		mux.HandleFunc("/v1/messages", p.handleAnthropicMessages)
		// Claude Code also calls /v1/messages with POST
	case "openai":
		mux.HandleFunc("/v1/chat/completions", p.handleOpenAICompletions)
		mux.HandleFunc("/v1/responses", p.handleOpenAIResponses)
		mux.HandleFunc("/responses", p.handleOpenAIResponses)
		mux.HandleFunc("/v1/models", p.handleOpenAIModels)
	}

	addr := fmt.Sprintf(":%d", p.config.Port)
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	p.log.Info("aicore-proxy starting", "addr", addr, "provider", p.config.Provider, "model", p.config.Model)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx) //nolint:errcheck
	}()

	return server.ListenAndServe()
}

func (p *Proxy) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
}

// handleAnthropicMessages proxies Anthropic Messages API requests to AI Core Bedrock endpoint.
func (p *Proxy) handleAnthropicMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Read and parse the request body to determine streaming
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	var req anthropicRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	// Apply Bedrock fixups
	body = fixAnthropicRequestForBedrock(body, &req)

	// Resolve deployment URL
	deploymentURL, err := p.deployment.Resolve(ctx, p.config.Model)
	if err != nil {
		p.log.Error("failed to resolve deployment", "error", err, "model", p.config.Model)
		http.Error(w, fmt.Sprintf("deployment resolution failed: %v", err), http.StatusBadGateway)
		return
	}

	// Build target URL
	targetPath := "/invoke"
	if req.Stream {
		targetPath = "/invoke-with-response-stream"
	}
	targetURL, err := url.Parse(deploymentURL + targetPath)
	if err != nil {
		http.Error(w, "invalid deployment URL", http.StatusInternalServerError)
		return
	}

	// Get token
	token, err := p.token.Token(ctx)
	if err != nil {
		p.log.Error("failed to get token", "error", err)
		http.Error(w, "authentication failed", http.StatusBadGateway)
		return
	}

	// Create upstream request
	upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL.String(), strings.NewReader(string(body)))
	if err != nil {
		http.Error(w, "failed to create upstream request", http.StatusInternalServerError)
		return
	}

	// Copy relevant headers from original request
	for _, h := range []string{"Content-Type", "Anthropic-Version", "Anthropic-Beta"} {
		if v := r.Header.Get(h); v != "" {
			upstreamReq.Header.Set(h, v)
		}
	}
	upstreamReq.Header.Set("Authorization", "Bearer "+token)
	upstreamReq.Header.Set("AI-Resource-Group", p.config.ResourceGroup)
	if upstreamReq.Header.Get("Content-Type") == "" {
		upstreamReq.Header.Set("Content-Type", "application/json")
	}

	// Execute request
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(upstreamReq)
	if err != nil {
		p.log.Error("upstream request failed", "error", err)
		http.Error(w, "upstream request failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// Stream response body directly (works for both streaming and non-streaming)
	if f, ok := w.(http.Flusher); ok && req.Stream {
		buf := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				w.Write(buf[:n]) //nolint:errcheck
				f.Flush()
			}
			if err != nil {
				break
			}
		}
	} else {
		io.Copy(w, resp.Body) //nolint:errcheck
	}
}

// handleOpenAICompletions proxies OpenAI Chat Completions API requests to AI Core.
func (p *Proxy) handleOpenAICompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Resolve deployment URL
	deploymentURL, err := p.deployment.Resolve(ctx, p.config.Model)
	if err != nil {
		p.log.Error("failed to resolve deployment", "error", err, "model", p.config.Model)
		http.Error(w, fmt.Sprintf("deployment resolution failed: %v", err), http.StatusBadGateway)
		return
	}

	// Build target URL — OpenAI format on AI Core
	targetURL, err := url.Parse(deploymentURL + "/chat/completions")
	if err != nil {
		http.Error(w, "invalid deployment URL", http.StatusInternalServerError)
		return
	}
	q := targetURL.Query()
	q.Set("api-version", "2024-02-01")
	targetURL.RawQuery = q.Encode()

	// Get token
	token, err := p.token.Token(ctx)
	if err != nil {
		p.log.Error("failed to get token", "error", err)
		http.Error(w, "authentication failed", http.StatusBadGateway)
		return
	}

	// Use httputil.ReverseProxy for streaming support
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL = targetURL
			req.Host = targetURL.Host
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("AI-Resource-Group", p.config.ResourceGroup)
			// Remove the fake API key that the CLI sends
			req.Header.Del("OpenAI-Organization")
		},
		FlushInterval: -1, // Flush immediately for streaming
	}

	proxy.ServeHTTP(w, r)
}

// handleOpenAIResponses proxies OpenAI Responses API requests to AI Core.
// The Responses API (POST /v1/responses) is used by newer Codex CLI versions.
func (p *Proxy) handleOpenAIResponses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Resolve deployment URL
	deploymentURL, err := p.deployment.Resolve(ctx, p.config.Model)
	if err != nil {
		p.log.Error("failed to resolve deployment", "error", err, "model", p.config.Model)
		http.Error(w, fmt.Sprintf("deployment resolution failed: %v", err), http.StatusBadGateway)
		return
	}

	// Build target URL — /responses on AI Core
	targetURL, err := url.Parse(deploymentURL + "/responses")
	if err != nil {
		http.Error(w, "invalid deployment URL", http.StatusInternalServerError)
		return
	}

	// Get token
	token, err := p.token.Token(ctx)
	if err != nil {
		p.log.Error("failed to get token", "error", err)
		http.Error(w, "authentication failed", http.StatusBadGateway)
		return
	}

	// Use httputil.ReverseProxy for streaming support
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL = targetURL
			req.Host = targetURL.Host
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("AI-Resource-Group", p.config.ResourceGroup)
			req.Header.Del("OpenAI-Organization")
		},
		FlushInterval: -1, // Flush immediately for streaming
	}

	proxy.ServeHTTP(w, r)
}

// handleOpenAIModels returns a minimal models list (Codex CLI sometimes calls this).
func (p *Proxy) handleOpenAIModels(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{
		"object": "list",
		"data": []map[string]any{
			{
				"id":       p.config.Model,
				"object":   "model",
				"owned_by": "sap-ai-core",
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}
