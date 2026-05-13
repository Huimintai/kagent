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
	// Outlook Secret keys.
	outlookKeyTenantID     = "tenant_id"
	outlookKeyClientID     = "client_id"
	outlookKeyClientSecret = "client_secret"

	// outlookDefaultScope is the default Microsoft Graph scope for client_credentials.
	outlookDefaultScope = "https://graph.microsoft.com/.default"
)

// outlookTokenResponse represents the Azure AD OAuth2 token response.
type outlookTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// OutlookAdapter implements auth.PlatformAdapter for Microsoft Outlook/Graph.
// It performs an OAuth2 client_credentials flow to obtain an app-only token
// for sending email via a central technical user.
type OutlookAdapter struct {
	client     client.Client
	httpClient *http.Client
}

// NewOutlookAdapter creates an OutlookAdapter with the given controller-runtime client.
func NewOutlookAdapter(c client.Client) *OutlookAdapter {
	return &OutlookAdapter{
		client:     c,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Platform returns the platform identifier.
func (a *OutlookAdapter) Platform() string {
	return "outlook"
}

// Mint creates a short-lived Microsoft Graph access token using the OAuth2 client_credentials flow.
// It reads the tenant ID, client ID, and client secret from the referenced Secret,
// then exchanges them at the Azure AD token endpoint for an app-only access token.
func (a *OutlookAdapter) Mint(ctx context.Context, source v1alpha2.CredentialSource, principal auth.Principal, scopes []string) (*auth.Token, error) {
	if source.SecretRef == nil {
		return nil, fmt.Errorf("%w: SecretRef is required for outlook adapter", ErrInvalidSource)
	}

	secret := &corev1.Secret{}
	if err := a.client.Get(ctx, types.NamespacedName{Name: source.SecretRef.Name, Namespace: "default"}, secret); err != nil {
		return nil, fmt.Errorf("failed to get Secret %s: %w", source.SecretRef.Name, err)
	}

	tenantID, err := getSecretKey(secret, outlookKeyTenantID)
	if err != nil {
		return nil, err
	}
	clientID, err := getSecretKey(secret, outlookKeyClientID)
	if err != nil {
		return nil, err
	}
	clientSecret, err := getSecretKey(secret, outlookKeyClientSecret)
	if err != nil {
		return nil, err
	}

	tokenEndpoint := source.TokenEndpoint
	if tokenEndpoint == "" {
		tokenEndpoint = fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)
	}

	accessToken, expiresIn, err := a.acquireToken(ctx, tokenEndpoint, clientID, clientSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire Outlook access token: %w", err)
	}

	return &auth.Token{
		Value:     accessToken,
		ExpiresAt: time.Now().Add(time.Duration(expiresIn) * time.Second),
		Platform:  "outlook",
		Scopes:    scopes,
		Principal: principal.User.ID,
	}, nil
}

// Validate checks that the credential source has the required configuration for Outlook.
func (a *OutlookAdapter) Validate(source v1alpha2.CredentialSource) error {
	if source.Type != v1alpha2.CredentialSourceTypeOAuth2 {
		return fmt.Errorf("%w: expected source type OAuth2, got %s", ErrInvalidSource, source.Type)
	}
	if source.SecretRef == nil {
		return fmt.Errorf("%w: SecretRef is required for outlook adapter", ErrInvalidSource)
	}
	if source.SecretRef.Name == "" {
		return fmt.Errorf("%w: SecretRef.Name must not be empty", ErrInvalidSource)
	}
	return nil
}

// acquireToken performs the OAuth2 client_credentials grant at the given endpoint.
func (a *OutlookAdapter) acquireToken(ctx context.Context, tokenEndpoint, clientID, clientSecret string) (string, int, error) {
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"scope":         {outlookDefaultScope},
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

	var tokenResp outlookTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", 0, fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", 0, fmt.Errorf("token response missing access_token")
	}

	return tokenResp.AccessToken, tokenResp.ExpiresIn, nil
}
