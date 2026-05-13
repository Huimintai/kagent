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
	// Jira Secret keys.
	jiraKeyClientID     = "client_id"
	jiraKeyClientSecret = "client_secret"
	jiraKeyRefreshToken = "refresh_token"

	// jiraDefaultTokenEndpoint is the default Atlassian OAuth2 token endpoint.
	jiraDefaultTokenEndpoint = "https://auth.atlassian.com/oauth/token"
)

// jiraTokenResponse represents the Atlassian OAuth2 token response.
type jiraTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// JiraAdapter implements auth.PlatformAdapter for Jira/Atlassian.
// It performs an OAuth2 token refresh using credentials stored in a Secret.
type JiraAdapter struct {
	client     client.Client
	httpClient *http.Client
}

// NewJiraAdapter creates a JiraAdapter with the given controller-runtime client.
func NewJiraAdapter(c client.Client) *JiraAdapter {
	return &JiraAdapter{
		client:     c,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Platform returns the platform identifier.
func (a *JiraAdapter) Platform() string {
	return "jira"
}

// Mint creates a short-lived Jira access token by performing an OAuth2 refresh token exchange.
// It reads the client credentials and refresh token from the referenced Secret,
// then exchanges them at the token endpoint for a new access token.
func (a *JiraAdapter) Mint(ctx context.Context, source v1alpha2.CredentialSource, principal auth.Principal, scopes []string) (*auth.Token, error) {
	if source.SecretRef == nil {
		return nil, fmt.Errorf("%w: SecretRef is required for jira adapter", ErrInvalidSource)
	}

	secret := &corev1.Secret{}
	if err := a.client.Get(ctx, types.NamespacedName{Name: source.SecretRef.Name, Namespace: "default"}, secret); err != nil {
		return nil, fmt.Errorf("failed to get Secret %s: %w", source.SecretRef.Name, err)
	}

	clientID, err := getSecretKey(secret, jiraKeyClientID)
	if err != nil {
		return nil, err
	}
	clientSecret, err := getSecretKey(secret, jiraKeyClientSecret)
	if err != nil {
		return nil, err
	}
	refreshToken, err := getSecretKey(secret, jiraKeyRefreshToken)
	if err != nil {
		return nil, err
	}

	tokenEndpoint := source.TokenEndpoint
	if tokenEndpoint == "" {
		tokenEndpoint = jiraDefaultTokenEndpoint
	}

	accessToken, expiresIn, err := a.refreshAccessToken(ctx, tokenEndpoint, clientID, clientSecret, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh Jira access token: %w", err)
	}

	return &auth.Token{
		Value:     accessToken,
		ExpiresAt: time.Now().Add(time.Duration(expiresIn) * time.Second),
		Platform:  "jira",
		Scopes:    scopes,
		Principal: principal.User.ID,
	}, nil
}

// Validate checks that the credential source has the required configuration for Jira.
func (a *JiraAdapter) Validate(source v1alpha2.CredentialSource) error {
	if source.Type != v1alpha2.CredentialSourceTypeOAuth2 {
		return fmt.Errorf("%w: expected source type OAuth2, got %s", ErrInvalidSource, source.Type)
	}
	if source.SecretRef == nil {
		return fmt.Errorf("%w: SecretRef is required for jira adapter", ErrInvalidSource)
	}
	if source.SecretRef.Name == "" {
		return fmt.Errorf("%w: SecretRef.Name must not be empty", ErrInvalidSource)
	}
	return nil
}

// refreshAccessToken exchanges a refresh token for a new access token at the given endpoint.
func (a *JiraAdapter) refreshAccessToken(ctx context.Context, tokenEndpoint, clientID, clientSecret, refreshToken string) (string, int, error) {
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

	var tokenResp jiraTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", 0, fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", 0, fmt.Errorf("token response missing access_token")
	}

	return tokenResp.AccessToken, tokenResp.ExpiresIn, nil
}
