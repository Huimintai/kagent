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

package auth

import (
	"context"
	"time"

	v1alpha2 "github.com/kagent-dev/kagent/go/api/v1alpha2"
)

// Token represents a short-lived credential for a platform.
type Token struct {
	// Value is the raw token string (e.g., Bearer token, API key).
	Value string

	// ExpiresAt is when this token becomes invalid.
	ExpiresAt time.Time

	// Platform identifies which external platform this token is for.
	Platform string

	// Scopes lists the platform-specific permissions granted.
	Scopes []string

	// Principal identifies who this token was issued for.
	Principal string
}

// Grant represents an active credential grant.
type Grant struct {
	// Platform identifies the external platform.
	Platform string

	// Scopes lists the granted permissions.
	Scopes []string

	// ExpiresAt is when this grant expires.
	ExpiresAt time.Time
}

// CredentialBroker manages platform credential lifecycle.
// It is the central interface for acquiring scoped, short-lived tokens
// from PlatformCredential resources on behalf of principals.
type CredentialBroker interface {
	// Acquire mints a short-lived token for the given principal and platform.
	// The broker locates the matching PlatformCredential, evaluates access policy,
	// and delegates to the appropriate PlatformAdapter.
	Acquire(ctx context.Context, principal Principal, platform string, scopes []string) (*Token, error)

	// Revoke invalidates any cached tokens for the principal+platform combination.
	Revoke(ctx context.Context, principal Principal, platform string) error

	// ListGrants returns active grants for a principal.
	ListGrants(ctx context.Context, principal Principal) ([]Grant, error)
}

// PlatformAdapter implements credential minting for a specific platform.
// Each supported platform (GitHub, Jira, Slack, K8s) provides an adapter
// that knows how to exchange stored credential material for short-lived tokens.
type PlatformAdapter interface {
	// Platform returns the identifier for this adapter (e.g., "github", "jira").
	Platform() string

	// Mint creates a short-lived token using the provided credential source.
	Mint(ctx context.Context, source v1alpha2.CredentialSource, principal Principal, scopes []string) (*Token, error)

	// Validate checks that the credential source configuration is valid
	// for this platform (e.g., required secret keys are present).
	Validate(source v1alpha2.CredentialSource) error
}
