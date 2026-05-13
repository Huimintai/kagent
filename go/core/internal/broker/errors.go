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

import "errors"

// Sentinel errors for the credential broker.
var (
	// ErrNoCredential is returned when no PlatformCredential matches the request.
	ErrNoCredential = errors.New("no matching PlatformCredential found")

	// ErrPolicyDenied is returned when the principal is not permitted by the AccessPolicy.
	ErrPolicyDenied = errors.New("access denied by credential policy")

	// ErrConsentRequired is returned when user delegation consent is required but not present.
	ErrConsentRequired = errors.New("user consent required for delegated credential")

	// ErrAdapterNotFound is returned when no adapter is registered for the requested platform.
	ErrAdapterNotFound = errors.New("no adapter registered for platform")

	// ErrInvalidSource is returned when the credential source configuration is invalid.
	ErrInvalidSource = errors.New("invalid credential source configuration")
)
