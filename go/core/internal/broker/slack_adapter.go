/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha2 "github.com/kagent-dev/kagent/go/api/v1alpha2"
	"github.com/kagent-dev/kagent/go/core/pkg/auth"
)

const (
	// Slack Secret keys.
	slackKeyBotToken     = "bot_token"
	slackKeyClientID     = "client_id"
	slackKeyClientSecret = "client_secret"
	slackKeyRefreshToken = "refresh_token"

	// slackDefaultTokenEndpoint is the default Slack OAuth2 token endpoint.
	slackDefaultTokenEndpoint = "https://slack.com/api/oauth.v2.access"
)

// slackTokenResponse represents the Slack OAuth2 token response.
type slackTokenResponse struct {
	OK          bool   `json:"ok"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	Error       string `json:"error"`
}

// SlackAdapter implements auth.PlatformAdapter for Slack.
// It supports two credential modes:
//   - BotToken: directly returns a long-lived bot token from a Secret.
//   - OAuth2UserDelegation: performs an OAuth2 refresh flow for user-scoped tokens.
type SlackAdapter struct {
	client     client.Client
	httpClient *http.Client
}

// NewSlackAdapter creates a SlackAdapter with the given controller-runtime client.
func NewSlackAdapter(c client.Client) *SlackAdapter {
	return &SlackAdapter{
		client:     c,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Platform returns the platform identifier.
func (a *SlackAdapter) Platform() string {
	return "slack"
}

// Mint creates a Slack token based on the credential source type.
// For BotToken sources, it reads the token directly from the Secret.
// For OAuth2UserDelegation sources, it performs a refresh token exchange.
func (a *SlackAdapter) Mint(ctx context.Context, source v1alpha2.CredentialSource, principal auth.Principal, scopes []string) (*auth.Token, error) {
	if source.SecretRef == nil {
		return nil, fmt.Errorf("%w: SecretRef is required for slack adapter", ErrInvalidSource)
	}

	secret := &corev1.Secret{}
	if err := a.client.Get(ctx, types.NamespacedName{Name: source.SecretRef.Name, Namespace: "default"}, secret); err != nil {
		return nil, fmt.Errorf("failed to get Secret %s: %w", source.SecretRef.Name, err)
	}

	switch source.Type {
	case v1alpha2.CredentialSourceTypeBotToken:
		return a.mintBotToken(secret, principal, scopes)
	case v1alpha2.CredentialSourceTypeOAuth2UserDelegation:
		return a.mintOAuth2Token(ctx, source, secret, principal, scopes)
	default:
		return nil, fmt.Errorf("%w: unsupported source type %s for slack adapter", ErrInvalidSource, source.Type)
	}
}

// Validate checks that the credential source has the required configuration for Slack.
func (a *SlackAdapter) Validate(source v1alpha2.CredentialSource) error {
	if source.Type != v1alpha2.CredentialSourceTypeBotToken && source.Type != v1alpha2.CredentialSourceTypeOAuth2UserDelegation {
		return fmt.Errorf("%w: expected source type BotToken or OAuth2UserDelegation, got %s", ErrInvalidSource, source.Type)
	}
	if source.SecretRef == nil {
		return fmt.Errorf("%w: SecretRef is required for slack adapter", ErrInvalidSource)
	}
	if source.SecretRef.Name == "" {
		return fmt.Errorf("%w: SecretRef.Name must not be empty", ErrInvalidSource)
	}
	return nil
}

// mintBotToken reads the bot token directly from the Secret and returns it.
// Bot tokens are long-lived but scope-bound, so no expiry is tracked.
func (a *SlackAdapter) mintBotToken(secret *corev1.Secret, principal auth.Principal, scopes []string) (*auth.Token, error) {
	botToken, err := getSecretKey(secret, slackKeyBotToken)
	if err != nil {
		return nil, err
	}

	return &auth.Token{
		Value:     botToken,
		ExpiresAt: time.Now().Add(24 * time.Hour), // Bot tokens don't expire; use a generous refresh window.
		Platform:  "slack",
		Scopes:    scopes,
		Principal: principal.User.ID,
	}, nil
}

// mintOAuth2Token performs an OAuth2 refresh token exchange for a user-scoped Slack token.
func (a *SlackAdapter) mintOAuth2Token(ctx context.Context, source v1alpha2.CredentialSource, secret *corev1.Secret, principal auth.Principal, scopes []string) (*auth.Token, error) {
	clientID, err := getSecretKey(secret, slackKeyClientID)
	if err != nil {
		return nil, err
	}
	clientSecret, err := getSecretKey(secret, slackKeyClientSecret)
	if err != nil {
		return nil, err
	}
	refreshToken, err := getSecretKey(secret, slackKeyRefreshToken)
	if err != nil {
		return nil, err
	}

	tokenEndpoint := source.TokenEndpoint
	if tokenEndpoint == "" {
		tokenEndpoint = slackDefaultTokenEndpoint
	}

	accessToken, expiresIn, err := a.refreshAccessToken(ctx, tokenEndpoint, clientID, clientSecret, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh Slack access token: %w", err)
	}

	return &auth.Token{
		Value:     accessToken,
		ExpiresAt: time.Now().Add(time.Duration(expiresIn) * time.Second),
		Platform:  "slack",
		Scopes:    scopes,
		Principal: principal.User.ID,
	}, nil
}

// refreshAccessToken exchanges a refresh token for a new access token at the Slack token endpoint.
func (a *SlackAdapter) refreshAccessToken(ctx context.Context, tokenEndpoint, clientID, clientSecret, refreshToken string) (string, int, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"refresh_token": {refreshToken},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", 0, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("failed to call token endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", 0, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(respBody))
	}

	var tokenResp slackTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", 0, fmt.Errorf("failed to decode token response: %w", err)
	}

	if !tokenResp.OK {
		return "", 0, fmt.Errorf("Slack API error: %s", tokenResp.Error)
	}

	if tokenResp.AccessToken == "" {
		return "", 0, fmt.Errorf("token response missing access_token")
	}

	return tokenResp.AccessToken, tokenResp.ExpiresIn, nil
}
