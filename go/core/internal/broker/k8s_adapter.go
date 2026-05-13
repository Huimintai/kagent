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
	"time"

	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha2 "github.com/kagent-dev/kagent/go/api/v1alpha2"
	"github.com/kagent-dev/kagent/go/core/pkg/auth"
)

const (
	// k8sTokenDefaultExpiration is the default lifetime for minted ServiceAccount tokens.
	k8sTokenDefaultExpiration = 1 * time.Hour

	// k8sDefaultNamespace is used when no namespace context is available.
	k8sDefaultNamespace = "default"
)

// K8sAdapter implements auth.PlatformAdapter for Kubernetes.
// It uses the TokenRequest API to create audience-scoped ServiceAccount tokens.
type K8sAdapter struct {
	client client.Client
}

// NewK8sAdapter creates a K8sAdapter with the given controller-runtime client.
func NewK8sAdapter(c client.Client) *K8sAdapter {
	return &K8sAdapter{client: c}
}

// Platform returns the platform identifier.
func (a *K8sAdapter) Platform() string {
	return "k8s"
}

// Mint creates a short-lived ServiceAccount token using the TokenRequest API.
// The scopes parameter is used as audiences for the generated token.
// The SecretRef in the source must reference a Secret whose name corresponds
// to the ServiceAccount to use for token creation.
func (a *K8sAdapter) Mint(ctx context.Context, source v1alpha2.CredentialSource, principal auth.Principal, scopes []string) (*auth.Token, error) {
	if source.SecretRef == nil {
		return nil, fmt.Errorf("%w: SecretRef is required for k8s adapter", ErrInvalidSource)
	}

	saName := source.SecretRef.Name
	namespace := k8sDefaultNamespace

	audiences := scopes
	if len(audiences) == 0 {
		audiences = []string{"https://kubernetes.default.svc"}
	}

	// Look up the ServiceAccount to use as the token subject.
	sa := &corev1.ServiceAccount{}
	if err := a.client.Get(ctx, types.NamespacedName{Name: saName, Namespace: namespace}, sa); err != nil {
		return nil, fmt.Errorf("failed to get ServiceAccount %s/%s: %w", namespace, saName, err)
	}

	expirationSeconds := int64(k8sTokenDefaultExpiration / time.Second)
	tokenRequest := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         audiences,
			ExpirationSeconds: &expirationSeconds,
		},
	}

	// Create the TokenRequest sub-resource on the ServiceAccount.
	if err := a.client.SubResource("token").Create(ctx, sa, tokenRequest); err != nil {
		return nil, fmt.Errorf("failed to create token for ServiceAccount %s/%s: %w", namespace, saName, err)
	}

	return &auth.Token{
		Value:     tokenRequest.Status.Token,
		ExpiresAt: tokenRequest.Status.ExpirationTimestamp.Time,
		Platform:  "k8s",
		Scopes:    audiences,
		Principal: principal.User.ID,
	}, nil
}

// Validate checks that the credential source has the required configuration for Kubernetes.
func (a *K8sAdapter) Validate(source v1alpha2.CredentialSource) error {
	if source.Type != v1alpha2.CredentialSourceTypeServiceAccount {
		return fmt.Errorf("%w: expected source type ServiceAccount, got %s", ErrInvalidSource, source.Type)
	}
	if source.SecretRef == nil {
		return fmt.Errorf("%w: SecretRef is required for k8s adapter", ErrInvalidSource)
	}
	if source.SecretRef.Name == "" {
		return fmt.Errorf("%w: SecretRef.Name must not be empty", ErrInvalidSource)
	}
	return nil
}
