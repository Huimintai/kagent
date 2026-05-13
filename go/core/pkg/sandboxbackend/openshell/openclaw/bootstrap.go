package openclaw

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/kagent-dev/kagent/go/api/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BuildBootstrapJSON builds ~/.openclaw/openclaw.json contents plus environment variables that must be present when
// OpenClaw resolves openshell:resolve:env:<VAR> (API key + channel tokens).
func BuildBootstrapJSON(ctx context.Context, kube client.Client, namespace string, sbx *v1alpha2.AgentHarness, mc *v1alpha2.ModelConfig, gwPort int) ([]byte, map[string]string, error) {
	if mc == nil {
		return nil, nil, fmt.Errorf("ModelConfig is required")
	}
	apiKey, err := ResolveModelConfigAPIKey(ctx, kube, mc)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve model API key: %w", err)
	}
	apiAdapter, err := providerAPI(mc)
	if err != nil {
		return nil, nil, err
	}

	apiKeyEnv := DefaultAPIKeyEnvVar(mc.Spec.Provider)
	// secretEnv holds only env vars that openclaw's secrets provider must expose
	// (API keys, channel tokens). These go into the openclaw.json secrets allowlist
	// and must match the pattern /^[A-Z][A-Z0-9_]{0,127}$/.
	secretEnv := map[string]string{
		apiKeyEnv: apiKey,
	}

	modelID := strings.TrimSpace(mc.Spec.Model)
	if modelID == "" {
		return nil, nil, fmt.Errorf("ModelConfig.spec.model is required for OpenClaw bootstrap JSON")
	}

	providerRecord := GatewayProviderRecordName(mc.Spec.Provider)
	doc := buildCoreBootstrapDocument(mc, gwPort, apiKeyEnv, providerRecord, modelID, apiAdapter)

	chState, err := accumulateHarnessChannels(ctx, kube, namespace, sbx.Spec.Channels, secretEnv)
	if err != nil {
		return nil, nil, err
	}
	doc.Channels = chState.channelsJSON()

	applySecretsAllowlist(&doc, secretEnv)

	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal openclaw json: %w", err)
	}

	// processEnv contains all env vars to inject into the openclaw gateway process,
	// including proxy bypass settings that are not secrets and must not be in the allowlist.
	processEnv := make(map[string]string, len(secretEnv)+2)
	for k, v := range secretEnv {
		processEnv[k] = v
	}
	// If the model endpoint is a cluster-internal URL (http scheme), add its host to no_proxy
	// so the openclaw node process bypasses any HTTPS_PROXY set in the sandbox environment.
	baseURL := bootstrapProviderBaseURL(mc)
	if u, parseErr := url.Parse(baseURL); parseErr == nil && u.Scheme == "http" && u.Host != "" {
		noProxy := "127.0.0.1,localhost,::1," + u.Hostname()
		processEnv["no_proxy"] = noProxy
		processEnv["NO_PROXY"] = noProxy
	}

	return raw, processEnv, nil
}

func buildCoreBootstrapDocument(mc *v1alpha2.ModelConfig, gwPort int, apiKeyEnv, providerRecord, modelID, apiAdapter string) bootstrapDocument {
	baseURL := bootstrapProviderBaseURL(mc)
	return bootstrapDocument{
		Gateway: gatewaySection{
			Mode: "local",
			Auth: gatewayAuth{Mode: "none"},
			Port: gwPort,
		},
		Models: modelsSection{
			Mode: "merge",
			Providers: map[string]providerSettings{
				providerRecord: {
					BaseURL: baseURL,
					APIKey:  openshellResolveEnv(apiKeyEnv),
					Auth:    providerAuth(mc),
					API:     apiAdapter,
					Models: []modelSlot{
						{ID: modelID, Name: modelID},
					},
				},
			},
		},
		Agents: agentsSection{
			Defaults: agentDefaults{
				Model: defaultModelPick{
					Primary: fmt.Sprintf("%s/%s", providerRecord, modelID),
				},
			},
		},
	}
}

func applySecretsAllowlist(doc *bootstrapDocument, env map[string]string) {
	secretAllow := make([]string, 0, len(env))
	for k := range env {
		secretAllow = append(secretAllow, k)
	}
	slices.Sort(secretAllow)
	doc.Secrets = secretsSection{
		Providers: map[string]secretProvider{
			bootstrapSecretProviderID: {
				Source:    "env",
				Allowlist: secretAllow,
			},
		},
	}
}
