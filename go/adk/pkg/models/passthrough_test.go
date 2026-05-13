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
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStaticTokenProvider(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		wantErr  bool
		wantTok  string
		platform string
	}{
		{name: "valid token", token: "my-secret-key", platform: "openai", wantTok: "my-secret-key"},
		{name: "empty token errors", token: "", platform: "openai", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &StaticTokenProvider{Token: tt.token}
			got, err := p.GetToken(context.Background(), tt.platform)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantTok {
				t.Errorf("GetToken() = %q, want %q", got, tt.wantTok)
			}
		})
	}
}

func TestBrokerTokenProvider(t *testing.T) {
	t.Run("successful token fetch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}
			var req brokerTokenRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("failed to decode request: %v", err)
			}
			if req.Principal != "test-user" {
				t.Errorf("expected principal 'test-user', got %q", req.Principal)
			}
			if req.Platform != "openai" {
				t.Errorf("expected platform 'openai', got %q", req.Platform)
			}
			json.NewEncoder(w).Encode(brokerTokenResponse{
				Token:     "broker-token-123",
				ExpiresIn: 3600,
			})
		}))
		defer server.Close()

		provider := NewBrokerTokenProvider(server.URL, "test-user")
		token, err := provider.GetToken(context.Background(), "openai")
		if err != nil {
			t.Fatalf("GetToken() error = %v", err)
		}
		if token != "broker-token-123" {
			t.Errorf("GetToken() = %q, want %q", token, "broker-token-123")
		}

		// Second call should use cache
		token2, err := provider.GetToken(context.Background(), "openai")
		if err != nil {
			t.Fatalf("GetToken() cached error = %v", err)
		}
		if token2 != "broker-token-123" {
			t.Errorf("cached GetToken() = %q, want %q", token2, "broker-token-123")
		}
	})

	t.Run("broker returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("access denied"))
		}))
		defer server.Close()

		provider := NewBrokerTokenProvider(server.URL, "test-user")
		_, err := provider.GetToken(context.Background(), "openai")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("empty endpoint errors", func(t *testing.T) {
		provider := NewBrokerTokenProvider("", "test-user")
		_, err := provider.GetToken(context.Background(), "openai")
		if err == nil {
			t.Fatal("expected error for empty endpoint")
		}
	})

	t.Run("empty principal errors", func(t *testing.T) {
		provider := NewBrokerTokenProvider("http://localhost:8080", "")
		_, err := provider.GetToken(context.Background(), "openai")
		if err == nil {
			t.Fatal("expected error for empty principal")
		}
	})
}

func TestTokenProviderTransport(t *testing.T) {
	var capturedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := &StaticTokenProvider{Token: "injected-token"}
	client := NewTokenProviderHTTPClient(provider, "test-platform", nil)

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if capturedAuth != "Bearer injected-token" {
		t.Errorf("expected Authorization 'Bearer injected-token', got %q", capturedAuth)
	}
}
