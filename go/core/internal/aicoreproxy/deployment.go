package aicoreproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// DeploymentResolver resolves SAP AI Core deployment URLs for a given model.
// It caches the result for 1 hour.
type DeploymentResolver struct {
	baseURL       string
	resourceGroup string
	token         *TokenProvider
	log           *slog.Logger

	mu            sync.Mutex
	cache         map[string]deploymentEntry
}

type deploymentEntry struct {
	url       string
	expiresAt time.Time
}

const deploymentCacheDuration = 1 * time.Hour

// NewDeploymentResolver creates a new DeploymentResolver.
func NewDeploymentResolver(baseURL, resourceGroup string, token *TokenProvider, log *slog.Logger) *DeploymentResolver {
	return &DeploymentResolver{
		baseURL:       strings.TrimRight(baseURL, "/"),
		resourceGroup: resourceGroup,
		token:         token,
		log:           log,
		cache:         make(map[string]deploymentEntry),
	}
}

// Resolve returns the deployment URL for the given model name.
func (dr *DeploymentResolver) Resolve(ctx context.Context, model string) (string, error) {
	dr.mu.Lock()
	if entry, ok := dr.cache[model]; ok && time.Now().Before(entry.expiresAt) {
		dr.mu.Unlock()
		return entry.url, nil
	}
	dr.mu.Unlock()

	// Fetch deployments from AI Core
	deploymentURL, err := dr.fetchDeploymentURL(ctx, model)
	if err != nil {
		return "", err
	}

	dr.mu.Lock()
	dr.cache[model] = deploymentEntry{url: deploymentURL, expiresAt: time.Now().Add(deploymentCacheDuration)}
	dr.mu.Unlock()

	return deploymentURL, nil
}

func (dr *DeploymentResolver) fetchDeploymentURL(ctx context.Context, model string) (string, error) {
	token, err := dr.token.Token(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get token for deployment lookup: %w", err)
	}

	endpoint := dr.baseURL + "/v2/lm/deployments?$top=100&scenarioId=foundation-models&status=RUNNING"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create deployment request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("AI-Resource-Group", dr.resourceGroup)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("deployment request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("deployment request returned status %d", resp.StatusCode)
	}

	var result struct {
		Resources []struct {
			ID      string `json:"id"`
			Status  string `json:"status"`
			Details struct {
				Resources struct {
					BackendDetails struct {
						Model struct {
							Name string `json:"name"`
						} `json:"model"`
					} `json:"backendDetails"`
				} `json:"resources"`
			} `json:"details"`
			DeploymentURL string `json:"deploymentUrl"`
		} `json:"resources"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode deployments response: %w", err)
	}

	// Find matching deployment by model name
	for _, d := range result.Resources {
		modelName := d.Details.Resources.BackendDetails.Model.Name
		if d.Status == "RUNNING" && modelName == model {
			if d.DeploymentURL != "" {
				dr.log.Info("resolved deployment", "model", model, "url", d.DeploymentURL)
				return d.DeploymentURL, nil
			}
			// Construct URL from ID
			deploymentURL := fmt.Sprintf("%s/v2/inference/deployments/%s", dr.baseURL, d.ID)
			dr.log.Info("resolved deployment", "model", model, "url", deploymentURL)
			return deploymentURL, nil
		}
	}

	return "", fmt.Errorf("no running deployment found for model %q", model)
}
