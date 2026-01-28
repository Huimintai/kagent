package mocks

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
)

// MockSAPAICoreServer provides a mock SAP AI Core server for testing
type MockSAPAICoreServer struct {
	server       *httptest.Server
	k8sURL       string
	deploymentID string
	modelName    string
}

// OAuth2TokenResponse represents the OAuth2 token endpoint response
type OAuth2TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// DeploymentListResponse represents the deployment list endpoint response
type DeploymentListResponse struct {
	Count     int          `json:"count"`
	Resources []Deployment `json:"resources"`
}

// Deployment represents a single deployment resource
type Deployment struct {
	ID                string            `json:"id"`
	ConfigurationID   string            `json:"configurationId"`
	ConfigurationName string            `json:"configurationName"`
	ScenarioID        string            `json:"scenarioId"`
	Status            string            `json:"status"`
	TargetStatus      string            `json:"targetStatus"`
	Details           DeploymentDetails `json:"details"`
}

// DeploymentDetails contains deployment configuration details
type DeploymentDetails struct {
	Resources DeploymentResources `json:"resources"`
}

// DeploymentResources contains backend details
type DeploymentResources struct {
	BackendDetails BackendDetails `json:"backend_details"`
}

// BackendDetails contains model information
type BackendDetails struct {
	Model ModelInfo `json:"model"`
}

// ModelInfo contains model name and version
type ModelInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ChatCompletionRequest represents OpenAI-compatible chat completion request
type ChatCompletionRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionResponse represents OpenAI-compatible chat completion response
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// NewMockSAPAICoreServer creates a new mock SAP AI Core server
func NewMockSAPAICoreServer(deploymentID, modelName string, port uint16) *MockSAPAICoreServer {
	mock := &MockSAPAICoreServer{
		deploymentID: deploymentID,
		modelName:    modelName,
	}

	// Use httptest.NewUnstartedServer to get more control
	mock.server = httptest.NewUnstartedServer(http.HandlerFunc(mock.handleRequest))

	// Configure the server to listen on all interfaces
	mock.server.Listener, _ = net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))

	// Start the server
	mock.server.Start()

	return mock
}

// SetK8sURL sets the Kubernetes-accessible URL for the server
func (m *MockSAPAICoreServer) SetK8sURL(k8sURL string) {
	m.k8sURL = k8sURL
}

// URL returns the base URL of the mock server
func (m *MockSAPAICoreServer) URL() string {
	return m.server.URL
}

// Close stops the mock server
func (m *MockSAPAICoreServer) Close() {
	m.server.Close()
}

func (m *MockSAPAICoreServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/oauth/token":
		m.handleOAuth2Token(w, r)
	case r.URL.Path == "/v2/lm/deployments":
		m.handleDeploymentList(w, r)
	case strings.HasPrefix(r.URL.Path, "/v2/inference/deployments/"):
		m.handleChatCompletion(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (m *MockSAPAICoreServer) handleOAuth2Token(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	grantType := r.FormValue("grant_type")
	if grantType != "client_credentials" {
		http.Error(w, "Unsupported grant_type", http.StatusBadRequest)
		return
	}

	// Return mock OAuth2 token
	response := OAuth2TokenResponse{
		AccessToken: "test-access-token-12345",
		TokenType:   "bearer",
		ExpiresIn:   3600,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (m *MockSAPAICoreServer) handleDeploymentList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Return mock deployment list with one RUNNING deployment
	response := DeploymentListResponse{
		Count: 1,
		Resources: []Deployment{
			{
				ID:                m.deploymentID,
				ConfigurationID:   "test-config",
				ConfigurationName: "test-model-config",
				ScenarioID:        "foundation-models",
				Status:            "RUNNING",
				TargetStatus:      "RUNNING",
				Details: DeploymentDetails{
					Resources: DeploymentResources{
						BackendDetails: BackendDetails{
							Model: ModelInfo{
								Name:    m.modelName,
								Version: "2024-01-01",
							},
						},
					},
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (m *MockSAPAICoreServer) handleChatCompletion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	var req ChatCompletionRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Find the user message to respond to
	var userMessage string
	for _, msg := range req.Messages {
		if msg.Role == "user" {
			userMessage = msg.Content
			break
		}
	}

	// Generate mock response based on user message
	responseContent := "Hello! I am responding from SAP AI Core."
	if strings.Contains(strings.ToLower(userMessage), "hello") {
		responseContent = "Hello! I am responding from SAP AI Core."
	}

	response := ChatCompletionResponse{
		ID:      "chatcmpl-sap-1",
		Object:  "chat.completion",
		Created: 1677652288,
		Model:   m.modelName,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: responseContent,
				},
				FinishReason: "stop",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
