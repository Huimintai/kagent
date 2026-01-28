//go:build e2e

package e2e_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kagent-dev/kagent/go/api/v1alpha2"
	e2emocks "github.com/kagent-dev/kagent/go/test/e2e/mocks"
)

// TestE2E_SAPAICoreInvoke validates the complete SAP AI Core integration chain:
// Secret (credentials) → ModelConfig (SAP provider) → Agent → A2A invocation
func TestE2E_SAPAICoreInvoke(t *testing.T) {
	// 1. Setup mock SAP AI Core server that simulates OAuth2, deployment query, and inference endpoints
	deploymentID := "test-deployment-id"
	modelName := "gpt-4.1-mini"
	mockServer := e2emocks.NewMockSAPAICoreServer(deploymentID, modelName, 0)
	defer mockServer.Close()

	// Convert to K8s-accessible URL
	baseURL := buildK8sURL(mockServer.URL())
	mockServer.SetK8sURL(baseURL)

	// 2. Setup Kubernetes client
	cli := setupK8sClient(t, false)

	// 3. Create Secret with SAP AI Core credentials (client_id and client_secret keys)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "sap-credentials-",
			Namespace:    "kagent",
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"client_id":     "test-client-id",
			"client_secret": "test-client-secret",
		},
	}
	err := cli.Create(t.Context(), secret)
	if err != nil {
		t.Fatalf("failed to create secret: %v", err)
	}
	cleanup(t, cli, secret)

	// 4. Create ModelConfig with SAP AI Core provider
	modelCfg := &v1alpha2.ModelConfig{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "sap-model-config-",
			Namespace:    "kagent",
		},
		Spec: v1alpha2.ModelConfigSpec{
			Model:           modelName,
			Provider:        v1alpha2.ModelProviderSAPAICore,
			APIKeySecret:    secret.Name,
			APIKeySecretKey: "client_id", // Primary key for reference
			SAPAICore: &v1alpha2.SAPAICoreConfig{
				BaseUrl:          baseURL,
				TokenUrl:         baseURL + "/oauth/token",
				ResourceGroup:    "default",
				DeploymentId:     deploymentID,
				ModelVersion:     "latest",
				ClientIdentifier: "kagent",
			},
		},
	}
	err = cli.Create(t.Context(), modelCfg)
	if err != nil {
		t.Fatalf("failed to create model config: %v", err)
	}
	cleanup(t, cli, modelCfg)

	// 5. Create Agent using the SAP AI Core ModelConfig
	agent := setupAgent(t, cli, modelCfg.Name, nil)

	// 6. Setup A2A client for agent invocation
	a2aClient := setupA2AClient(t, agent)

	// 7. Run synchronous invocation test
	t.Run("sync_invocation", func(t *testing.T) {
		runSyncTest(t, a2aClient, "Hello SAP AI Core", "Hello! I am responding from SAP AI Core.", nil)
	})

	// 8. Run streaming invocation test
	t.Run("streaming_invocation", func(t *testing.T) {
		runStreamingTest(t, a2aClient, "Hello SAP AI Core", "Hello! I am responding from SAP AI Core.")
	})
}
