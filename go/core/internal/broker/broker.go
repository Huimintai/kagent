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
	"fmt"
	"strings"

	v1alpha2 "github.com/kagent-dev/kagent/go/api/v1alpha2"
	"github.com/kagent-dev/kagent/go/core/pkg/auth"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Compile-time interface assertions.
var (
	_ auth.CredentialBroker = (*DefaultBroker)(nil)
	_ auth.PlatformAdapter  = (*K8sAdapter)(nil)
	_ auth.PlatformAdapter  = (*GitHubAdapter)(nil)
)

// DefaultBroker implements auth.CredentialBroker by looking up PlatformCredential
// resources, evaluating access policy, and delegating to platform adapters.
type DefaultBroker struct {
	adapters map[string]auth.PlatformAdapter
	client   client.Client
}

// New creates a DefaultBroker with the given controller-runtime client and adapters.
func New(c client.Client, adapters ...auth.PlatformAdapter) *DefaultBroker {
	m := make(map[string]auth.PlatformAdapter, len(adapters))
	for _, a := range adapters {
		m[a.Platform()] = a
	}
	return &DefaultBroker{
		adapters: m,
		client:   c,
	}
}

// Acquire mints a short-lived token for the given principal and platform.
func (b *DefaultBroker) Acquire(ctx context.Context, principal auth.Principal, platform string, scopes []string) (*auth.Token, error) {
	adapter, ok := b.adapters[platform]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrAdapterNotFound, platform)
	}

	cred, err := b.findCredential(ctx, platform)
	if err != nil {
		return nil, err
	}

	if err := b.evaluatePolicy(cred.Spec.AccessPolicy, principal, scopes); err != nil {
		return nil, err
	}

	token, err := adapter.Mint(ctx, cred.Spec.Source, principal, scopes)
	if err != nil {
		return nil, fmt.Errorf("adapter mint failed for platform %s: %w", platform, err)
	}
	return token, nil
}

// Revoke invalidates cached tokens for the principal+platform combination.
// Currently a no-op as tokens are short-lived and not cached.
func (b *DefaultBroker) Revoke(_ context.Context, _ auth.Principal, _ string) error {
	// Short-lived tokens are not cached in this implementation.
	// Future: invalidate token cache entries here.
	return nil
}

// ListGrants returns active grants for a principal by scanning all PlatformCredentials
// whose access policy permits the principal.
func (b *DefaultBroker) ListGrants(ctx context.Context, principal auth.Principal) ([]auth.Grant, error) {
	var credList v1alpha2.PlatformCredentialList
	if err := b.client.List(ctx, &credList); err != nil {
		return nil, fmt.Errorf("failed to list PlatformCredentials: %w", err)
	}

	var grants []auth.Grant
	for _, cred := range credList.Items {
		allowedScopes := b.allowedScopes(cred.Spec.AccessPolicy, principal)
		if len(allowedScopes) > 0 {
			grants = append(grants, auth.Grant{
				Platform: cred.Spec.Platform,
				Scopes:   allowedScopes,
			})
		}
	}
	return grants, nil
}

// ValidateCredential checks if the credential source configuration is valid
// for the specified platform using the registered adapter.
func (b *DefaultBroker) ValidateCredential(platform string, source v1alpha2.CredentialSource) error {
	adapter, ok := b.adapters[platform]
	if !ok {
		return fmt.Errorf("%w: %s", ErrAdapterNotFound, platform)
	}
	return adapter.Validate(source)
}

// findCredential locates the first PlatformCredential for the given platform.
func (b *DefaultBroker) findCredential(ctx context.Context, platform string) (*v1alpha2.PlatformCredential, error) {
	var credList v1alpha2.PlatformCredentialList
	if err := b.client.List(ctx, &credList); err != nil {
		return nil, fmt.Errorf("failed to list PlatformCredentials: %w", err)
	}

	for i := range credList.Items {
		if credList.Items[i].Spec.Platform == platform {
			return &credList.Items[i], nil
		}
	}
	return nil, fmt.Errorf("%w: platform=%s", ErrNoCredential, platform)
}

// evaluatePolicy checks that at least one AccessPolicyRule permits the principal
// with the requested scopes. If no rules are defined, access is denied by default.
func (b *DefaultBroker) evaluatePolicy(rules []v1alpha2.AccessPolicyRule, principal auth.Principal, scopes []string) error {
	if len(rules) == 0 {
		return fmt.Errorf("%w: no access policy rules defined", ErrPolicyDenied)
	}

	for _, rule := range rules {
		if !b.principalMatches(rule.Principals, principal) {
			continue
		}
		if rule.Delegation == "required" && principal.User.ID == "" {
			return fmt.Errorf("%w: rule requires user delegation", ErrConsentRequired)
		}
		if b.scopesCovered(rule.Scopes, scopes) {
			return nil
		}
	}
	return fmt.Errorf("%w: principal not permitted for requested scopes", ErrPolicyDenied)
}

// principalMatches checks if the principal matches any of the rule's principal patterns.
// Patterns are of the form "agent:<name>", "user:<name>", "skill:<name>", or wildcards like "agent:*".
func (b *DefaultBroker) principalMatches(patterns []string, principal auth.Principal) bool {
	for _, pattern := range patterns {
		if pattern == "*" {
			return true
		}
		parts := strings.SplitN(pattern, ":", 2)
		if len(parts) != 2 {
			continue
		}
		kind, name := parts[0], parts[1]
		switch kind {
		case "agent":
			if name == "*" || name == principal.Agent.ID {
				return true
			}
		case "user":
			if name == "*" || name == principal.User.ID {
				return true
			}
		}
	}
	return false
}

// scopesCovered checks that all requested scopes are present in the allowed set.
func (b *DefaultBroker) scopesCovered(allowed, requested []string) bool {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, s := range allowed {
		allowedSet[s] = struct{}{}
	}
	// Wildcard: if allowed contains "*", all scopes are permitted.
	if _, ok := allowedSet["*"]; ok {
		return true
	}
	for _, s := range requested {
		if _, ok := allowedSet[s]; !ok {
			return false
		}
	}
	return true
}

// allowedScopes returns the union of scopes from all rules that match the principal.
func (b *DefaultBroker) allowedScopes(rules []v1alpha2.AccessPolicyRule, principal auth.Principal) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, rule := range rules {
		if !b.principalMatches(rule.Principals, principal) {
			continue
		}
		for _, s := range rule.Scopes {
			if _, ok := seen[s]; !ok {
				seen[s] = struct{}{}
				result = append(result, s)
			}
		}
	}
	return result
}
