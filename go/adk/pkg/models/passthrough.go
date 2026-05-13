// Copyright 2025 Kagent Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// TokenProvider supplies tokens for model API calls.
// Implementations can obtain tokens from environment variables, credential brokers,
// service account key files, or any other source.
type TokenProvider interface {
	// GetToken returns a bearer token for the given platform.
	// The platform string identifies the target service (e.g., "openai", "azure", "bedrock").
	GetToken(ctx context.Context, platform string) (string, error)
}

// StaticTokenProvider returns a fixed token regardless of platform.
// Useful for testing or when a single API key is shared across all calls.
type StaticTokenProvider struct {
	Token string
}

// GetToken implements TokenProvider by returning the static token.
func (p *StaticTokenProvider) GetToken(_ context.Context, _ string) (string, error) {
	if p.Token == "" {
		return "", fmt.Errorf("static token provider has no token configured")
	}
	return p.Token, nil
}

// BrokerTokenProvider gets tokens from an external credential broker service.
// The broker is expected to expose an HTTP endpoint that returns tokens for a
// given principal and platform combination.
type BrokerTokenProvider struct {
	brokerEndpoint string
	principal      string
	httpClient     *http.Client

	mu          sync.Mutex
	cachedToken map[string]*cachedToken
}

type cachedToken struct {
	token     string
	expiresAt time.Time
}

// brokerTokenRequest is the JSON body sent to the credential broker.
type brokerTokenRequest struct {
	Principal string `json:"principal"`
	Platform  string `json:"platform"`
}

// brokerTokenResponse is the JSON response from the credential broker.
type brokerTokenResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in,omitempty"` // seconds until expiry
}

// NewBrokerTokenProvider creates a BrokerTokenProvider that obtains tokens from
// the specified credential broker endpoint for the given principal identity.
func NewBrokerTokenProvider(endpoint, principal string) *BrokerTokenProvider {
	return &BrokerTokenProvider{
		brokerEndpoint: endpoint,
		principal:      principal,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		cachedToken:    make(map[string]*cachedToken),
	}
}

// GetToken implements TokenProvider by calling the credential broker.
// Tokens are cached until 2 minutes before their expiry time.
func (p *BrokerTokenProvider) GetToken(ctx context.Context, platform string) (string, error) {
	if p.brokerEndpoint == "" {
		return "", fmt.Errorf("broker endpoint is not configured")
	}
	if p.principal == "" {
		return "", fmt.Errorf("principal is not configured")
	}

	// Check cache
	p.mu.Lock()
	if cached, ok := p.cachedToken[platform]; ok {
		if time.Now().Before(cached.expiresAt.Add(-2 * time.Minute)) {
			token := cached.token
			p.mu.Unlock()
			return token, nil
		}
	}
	p.mu.Unlock()

	// Fetch new token from broker
	reqBody := brokerTokenRequest{
		Principal: p.principal,
		Platform:  platform,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal broker request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.brokerEndpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create broker request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("broker request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("broker returned HTTP %d: %s", resp.StatusCode, string(errBody))
	}

	var tokenResp brokerTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode broker response: %w", err)
	}

	if tokenResp.Token == "" {
		return "", fmt.Errorf("broker returned empty token for platform %q", platform)
	}

	// Cache the token
	expiresAt := time.Now().Add(1 * time.Hour) // default 1 hour
	if tokenResp.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	p.mu.Lock()
	p.cachedToken[platform] = &cachedToken{
		token:     tokenResp.Token,
		expiresAt: expiresAt,
	}
	p.mu.Unlock()

	return tokenResp.Token, nil
}

// TokenProviderTransport is an http.RoundTripper that injects a bearer token
// obtained from a TokenProvider into outgoing requests.
type TokenProviderTransport struct {
	Base     http.RoundTripper
	Provider TokenProvider
	Platform string
}

// RoundTrip implements http.RoundTripper by injecting the Authorization header.
func (t *TokenProviderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.Provider.GetToken(req.Context(), t.Platform)
	if err != nil {
		return nil, fmt.Errorf("failed to get token for platform %q: %w", t.Platform, err)
	}

	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+token)
	return t.Base.RoundTrip(req)
}

// NewTokenProviderHTTPClient creates an http.Client that automatically injects
// bearer tokens from the given TokenProvider for the specified platform.
func NewTokenProviderHTTPClient(provider TokenProvider, platform string, base *http.Client) *http.Client {
	if base == nil {
		base = &http.Client{}
	}
	transport := base.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &http.Client{
		Timeout: base.Timeout,
		Transport: &TokenProviderTransport{
			Base:     transport,
			Provider: provider,
			Platform: platform,
		},
	}
}
