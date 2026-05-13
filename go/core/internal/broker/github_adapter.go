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
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha2 "github.com/kagent-dev/kagent/go/api/v1alpha2"
	"github.com/kagent-dev/kagent/go/core/pkg/auth"
)

const (
	// GitHub App Secret keys.
	githubKeyAppID          = "app-id"
	githubKeyInstallationID = "installation-id"
	githubKeyPrivateKey     = "private-key"

	// githubTokenExpiration is the maximum lifetime of a GitHub installation token.
	githubTokenExpiration = 1 * time.Hour

	// githubAPIBaseURL is the GitHub API base URL.
	githubAPIBaseURL = "https://api.github.com"
)

// GitHubAdapter implements auth.PlatformAdapter for GitHub.
// It reads a GitHub App private key from a Secret, creates a JWT,
// and exchanges it for an installation access token.
type GitHubAdapter struct {
	client     client.Client
	httpClient *http.Client
}

// NewGitHubAdapter creates a GitHubAdapter with the given controller-runtime client.
func NewGitHubAdapter(c client.Client) *GitHubAdapter {
	return &GitHubAdapter{
		client:     c,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Platform returns the platform identifier.
func (a *GitHubAdapter) Platform() string {
	return "github"
}

// Mint creates a GitHub installation access token.
// It reads the App private key from the referenced Secret, generates a JWT,
// and exchanges it for an installation token with the requested permissions.
func (a *GitHubAdapter) Mint(ctx context.Context, source v1alpha2.CredentialSource, principal auth.Principal, scopes []string) (*auth.Token, error) {
	if source.SecretRef == nil {
		return nil, fmt.Errorf("%w: SecretRef is required for github adapter", ErrInvalidSource)
	}

	secret := &corev1.Secret{}
	if err := a.client.Get(ctx, types.NamespacedName{Name: source.SecretRef.Name, Namespace: "default"}, secret); err != nil {
		return nil, fmt.Errorf("failed to get Secret %s: %w", source.SecretRef.Name, err)
	}

	appID, err := getSecretKey(secret, githubKeyAppID)
	if err != nil {
		return nil, err
	}
	installationID, err := getSecretKey(secret, githubKeyInstallationID)
	if err != nil {
		return nil, err
	}
	privateKeyPEM, err := getSecretKey(secret, githubKeyPrivateKey)
	if err != nil {
		return nil, err
	}

	privateKey, err := parseRSAPrivateKey(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	jwt, err := createGitHubJWT(appID, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub JWT: %w", err)
	}

	token, expiresAt, err := a.exchangeForInstallationToken(ctx, jwt, installationID, scopes)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange JWT for installation token: %w", err)
	}

	return &auth.Token{
		Value:     token,
		ExpiresAt: expiresAt,
		Platform:  "github",
		Scopes:    scopes,
		Principal: principal.User.ID,
	}, nil
}

// Validate checks that the credential source has the required Secret keys for GitHub.
func (a *GitHubAdapter) Validate(source v1alpha2.CredentialSource) error {
	if source.Type != v1alpha2.CredentialSourceTypeGitHubApp {
		return fmt.Errorf("%w: expected source type GitHubApp, got %s", ErrInvalidSource, source.Type)
	}
	if source.SecretRef == nil {
		return fmt.Errorf("%w: SecretRef is required for github adapter", ErrInvalidSource)
	}
	if source.SecretRef.Name == "" {
		return fmt.Errorf("%w: SecretRef.Name must not be empty", ErrInvalidSource)
	}
	return nil
}

// getSecretKey retrieves a key from a Secret's data, returning an error if missing.
func getSecretKey(secret *corev1.Secret, key string) (string, error) {
	data, ok := secret.Data[key]
	if !ok {
		return "", fmt.Errorf("%w: Secret %s missing required key %q", ErrInvalidSource, secret.Name, key)
	}
	return strings.TrimSpace(string(data)), nil
}

// parseRSAPrivateKey parses a PEM-encoded RSA private key.
func parseRSAPrivateKey(pemData string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8 format as fallback.
		parsed, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("failed to parse private key (PKCS1: %v, PKCS8: %v)", err, err2)
		}
		rsaKey, ok := parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("PKCS8 key is not RSA")
		}
		return rsaKey, nil
	}
	return key, nil
}

// createGitHubJWT creates a JWT for GitHub App authentication.
// The JWT is signed with RS256 and is valid for 10 minutes.
func createGitHubJWT(appID string, key *rsa.PrivateKey) (string, error) {
	now := time.Now()
	// GitHub recommends setting iat to 60 seconds in the past.
	iat := now.Add(-60 * time.Second)
	exp := now.Add(10 * time.Minute)

	header := map[string]string{
		"alg": "RS256",
		"typ": "JWT",
	}
	payload := map[string]interface{}{
		"iat": iat.Unix(),
		"exp": exp.Unix(),
		"iss": appID,
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JWT header: %w", err)
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JWT payload: %w", err)
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := headerB64 + "." + payloadB64

	hash := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(nil, key, crypto.SHA256, hash[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}
	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)

	return signingInput + "." + signatureB64, nil
}

// installationTokenResponse represents the GitHub API response for installation token creation.
type installationTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// exchangeForInstallationToken exchanges a JWT for a GitHub installation access token.
func (a *GitHubAdapter) exchangeForInstallationToken(ctx context.Context, jwt, installationID string, scopes []string) (string, time.Time, error) {
	url := fmt.Sprintf("%s/app/installations/%s/access_tokens", githubAPIBaseURL, installationID)

	// Build the request body with permissions from scopes.
	body := a.buildPermissionsBody(scopes)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to call GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", time.Time{}, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var tokenResp installationTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to decode GitHub token response: %w", err)
	}

	return tokenResp.Token, tokenResp.ExpiresAt, nil
}

// buildPermissionsBody constructs the JSON body for the installation token request.
// Scopes are expected in the format "permission:level" (e.g., "contents:read", "issues:write").
func (a *GitHubAdapter) buildPermissionsBody(scopes []string) string {
	if len(scopes) == 0 {
		return "{}"
	}

	permissions := make(map[string]string)
	for _, scope := range scopes {
		parts := strings.SplitN(scope, ":", 2)
		if len(parts) == 2 {
			permissions[parts[0]] = parts[1]
		}
	}

	if len(permissions) == 0 {
		return "{}"
	}

	body := map[string]interface{}{
		"permissions": permissions,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return "{}"
	}
	return string(data)
}
