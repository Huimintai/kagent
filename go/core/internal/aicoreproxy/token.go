package aicoreproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// TokenProvider manages OAuth2 client_credentials tokens with automatic refresh.
type TokenProvider struct {
	authURL      string
	clientID     string
	clientSecret string
	log          *slog.Logger

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

// NewTokenProvider creates a new TokenProvider.
func NewTokenProvider(authURL, clientID, clientSecret string, log *slog.Logger) *TokenProvider {
	return &TokenProvider{
		authURL:      authURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		log:          log,
	}
}

// Token returns a valid access token, refreshing if necessary.
func (tp *TokenProvider) Token(ctx context.Context) (string, error) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	// Return cached token if still valid (with 2-minute buffer)
	if tp.token != "" && time.Now().Before(tp.expiresAt.Add(-2*time.Minute)) {
		return tp.token, nil
	}

	// Build token endpoint URL
	tokenURL := strings.TrimRight(tp.authURL, "/")
	if !strings.HasSuffix(tokenURL, "/oauth/token") {
		tokenURL += "/oauth/token"
	}

	// Request token using client_credentials grant
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {tp.clientID},
		"client_secret": {tp.clientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request returned status %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	tp.token = tokenResp.AccessToken
	if tokenResp.ExpiresIn > 0 {
		tp.expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	} else {
		tp.expiresAt = time.Now().Add(12 * time.Hour)
	}

	tp.log.Info("token refreshed", "expires_in", tokenResp.ExpiresIn)
	return tp.token, nil
}

// Invalidate clears the cached token (e.g., after a 401 response).
func (tp *TokenProvider) Invalidate() {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	tp.token = ""
	tp.expiresAt = time.Time{}
}
