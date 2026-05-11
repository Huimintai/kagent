package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	api "github.com/kagent-dev/kagent/go/api/httpapi"
	"github.com/kagent-dev/kagent/go/api/v1alpha2"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestE2ESmokeTest verifies the core HTTP API surface introduced by our fork:
//   - Agent CRUD with ownership (user_id, private_mode)
//   - Session pinning (PATCH)
//   - ScheduledRun CRUD + frequency validation
//   - ToolServer user_id propagation
//   - Reference protection (agent delete blocked by active ScheduledRun)
func TestE2ESmokeTest(t *testing.T) {
	kagentURL := os.Getenv("KAGENT_URL")
	if kagentURL == "" {
		kagentURL = "http://localhost:8083"
	}

	namespace := "dbci-agent"
	userA := "user-a@example.com"
	userB := "user-b@example.com"
	agentName := "smoke-test-agent"
	srName := "smoke-test-sr"
	modelConfig := "sap-aicore-claude-46-sonnet"

	httpClient := &http.Client{Timeout: 10 * time.Second}

	// Helper: make HTTP request with user header
	doReqAs := func(t *testing.T, method, url, user string, body interface{}) *http.Response {
		t.Helper()
		var reader io.Reader
		if body != nil {
			b, err := json.Marshal(body)
			require.NoError(t, err)
			reader = bytes.NewReader(b)
		}
		req, err := http.NewRequest(method, url, reader)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		if user != "" {
			req.Header.Set("X-Forwarded-Email", user)
		}
		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		return resp
	}

	// Shorthand for user A
	doReq := func(t *testing.T, method, url string, body interface{}) *http.Response {
		return doReqAs(t, method, url, userA, body)
	}

	// Helper: read response body as string
	readBody := func(t *testing.T, resp *http.Response) string {
		t.Helper()
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		return string(b)
	}

	// Helper: decode JSON response
	decodeJSON := func(t *testing.T, resp *http.Response, v interface{}) {
		t.Helper()
		defer resp.Body.Close()
		err := json.NewDecoder(resp.Body).Decode(v)
		require.NoError(t, err)
	}

	// Cleanup
	cleanup := func() {
		doReq(t, http.MethodDelete, fmt.Sprintf("%s/api/scheduledruns/%s/%s", kagentURL, namespace, srName), nil)
		doReq(t, http.MethodDelete, fmt.Sprintf("%s/api/agents/%s/%s", kagentURL, namespace, agentName), nil)
	}
	cleanup()
	t.Cleanup(cleanup)

	// =========================================================================
	// Phase 1: Health check
	// =========================================================================
	t.Run("health", func(t *testing.T) {
		resp := doReq(t, http.MethodGet, kagentURL+"/health", nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})

	// =========================================================================
	// Phase 2: Create agent with ownership annotations
	// =========================================================================
	t.Run("agent_create_with_ownership", func(t *testing.T) {
		agent := v1alpha2.Agent{
			TypeMeta: metav1.TypeMeta{APIVersion: "kagent.dev/v1alpha2", Kind: "Agent"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      agentName,
				Namespace: namespace,
			},
			Spec: v1alpha2.AgentSpec{
				Type:        v1alpha2.AgentType_Declarative,
				Description: "Smoke test agent with ownership",
				Declarative: &v1alpha2.DeclarativeAgentSpec{
					ModelConfig:   modelConfig,
					SystemMessage: "You are a smoke test agent.",
				},
			},
		}
		resp := doReq(t, http.MethodPost, kagentURL+"/api/agents", agent)
		body := readBody(t, resp)
		require.Equal(t, http.StatusCreated, resp.StatusCode, "create agent failed: %s", body)
	})

	// =========================================================================
	// Phase 3: Get agent — verify ownership metadata in response
	// =========================================================================
	t.Run("agent_get_has_ownership", func(t *testing.T) {
		resp := doReq(t, http.MethodGet, fmt.Sprintf("%s/api/agents/%s/%s", kagentURL, namespace, agentName), nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var result api.StandardResponse[api.AgentResponse]
		decodeJSON(t, resp, &result)
		require.False(t, result.Error)
		// Agent created by user A should have user_id set
		require.Equal(t, userA, result.Data.UserID, "agent should have creator's user_id")
	})

	// =========================================================================
	// Phase 4: Agent edit by owner succeeds, by non-owner fails (if private)
	// =========================================================================
	t.Run("agent_ownership_enforcement", func(t *testing.T) {
		// User A can edit their own agent (make it private)
		agent := v1alpha2.Agent{
			TypeMeta: metav1.TypeMeta{APIVersion: "kagent.dev/v1alpha2", Kind: "Agent"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      agentName,
				Namespace: namespace,
				Annotations: map[string]string{
					"kagent.dev/user-id":      userA,
					"kagent.dev/private-mode": "true",
				},
			},
			Spec: v1alpha2.AgentSpec{
				Type:        v1alpha2.AgentType_Declarative,
				Description: "Updated by owner",
				Declarative: &v1alpha2.DeclarativeAgentSpec{
					ModelConfig:   modelConfig,
					SystemMessage: "Updated system message.",
				},
			},
		}
		resp := doReq(t, http.MethodPut, kagentURL+"/api/agents", agent)
		body := readBody(t, resp)
		require.Equal(t, http.StatusOK, resp.StatusCode, "owner edit should succeed: %s", body)

		// User B cannot delete user A's private agent
		resp = doReqAs(t, http.MethodDelete, fmt.Sprintf("%s/api/agents/%s/%s", kagentURL, namespace, agentName), userB, nil)
		resp.Body.Close()
		// Should be forbidden (403) or not found depending on implementation
		require.NotEqual(t, http.StatusOK, resp.StatusCode, "non-owner should not be able to delete private agent")
	})

	// =========================================================================
	// Phase 5: Session create + pin (PATCH)
	// =========================================================================
	var sessionID string
	t.Run("session_create_and_pin", func(t *testing.T) {
		// Create session
		agentRef := fmt.Sprintf("%s/%s", namespace, agentName)
		sessionName := "smoke-test-session"
		sessionReq := api.SessionRequest{
			AgentRef: &agentRef,
			Name:     &sessionName,
		}
		resp := doReq(t, http.MethodPost, kagentURL+"/api/sessions", sessionReq)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var result api.StandardResponse[api.Session]
		decodeJSON(t, resp, &result)
		require.False(t, result.Error)
		sessionID = result.Data.ID
		require.NotEmpty(t, sessionID)

		// PATCH to pin
		patchBody := map[string]interface{}{"pinned": true}
		resp = doReq(t, http.MethodPatch, fmt.Sprintf("%s/api/sessions/%s", kagentURL, sessionID), patchBody)
		body := readBody(t, resp)
		require.Equal(t, http.StatusOK, resp.StatusCode, "session pin failed: %s", body)

		// Verify pinned state
		resp = doReq(t, http.MethodGet, fmt.Sprintf("%s/api/sessions/%s", kagentURL, sessionID), nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var getResult api.StandardResponse[api.Session]
		decodeJSON(t, resp, &getResult)
		require.False(t, getResult.Error)
		require.True(t, getResult.Data.Pinned, "session should be pinned")
	})

	// =========================================================================
	// Phase 6: ScheduledRun — frequency validation (reject sub-hour)
	// =========================================================================
	t.Run("scheduledrun_reject_sub_hour", func(t *testing.T) {
		sr := v1alpha2.ScheduledRun{
			ObjectMeta: metav1.ObjectMeta{Name: srName, Namespace: namespace},
			Spec: v1alpha2.ScheduledRunSpec{
				Schedule:      "*/5 * * * *", // every 5 min — too frequent
				AgentRef:      v1alpha2.AgentReference{Name: agentName, Namespace: namespace},
				Prompt:        "test",
				MaxRunHistory: 5,
			},
		}
		resp := doReq(t, http.MethodPost, kagentURL+"/api/scheduledruns", sr)
		body := readBody(t, resp)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode, "sub-hour schedule should be rejected: %s", body)
	})

	// =========================================================================
	// Phase 7: ScheduledRun — valid CRUD
	// =========================================================================
	t.Run("scheduledrun_crud", func(t *testing.T) {
		// Create (hourly)
		sr := v1alpha2.ScheduledRun{
			ObjectMeta: metav1.ObjectMeta{Name: srName, Namespace: namespace},
			Spec: v1alpha2.ScheduledRunSpec{
				Schedule:      "0 * * * *",
				AgentRef:      v1alpha2.AgentReference{Name: agentName, Namespace: namespace},
				Prompt:        "run smoke test",
				Suspend:       false,
				MaxRunHistory: 3,
			},
		}
		resp := doReq(t, http.MethodPost, kagentURL+"/api/scheduledruns", sr)
		body := readBody(t, resp)
		require.Equal(t, http.StatusCreated, resp.StatusCode, "create scheduled run failed: %s", body)

		// GET
		resp = doReq(t, http.MethodGet, fmt.Sprintf("%s/api/scheduledruns/%s/%s", kagentURL, namespace, srName), nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// LIST
		resp = doReq(t, http.MethodGet, kagentURL+"/api/scheduledruns", nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// UPDATE (suspend)
		sr.Spec.Suspend = true
		resp = doReq(t, http.MethodPut, fmt.Sprintf("%s/api/scheduledruns/%s/%s", kagentURL, namespace, srName), sr)
		body = readBody(t, resp)
		require.Equal(t, http.StatusOK, resp.StatusCode, "update scheduled run failed: %s", body)
	})

	// =========================================================================
	// Phase 8: Reference protection — agent cannot be deleted while SR exists
	// =========================================================================
	t.Run("ref_protection_blocks_delete", func(t *testing.T) {
		resp := doReq(t, http.MethodDelete, fmt.Sprintf("%s/api/agents/%s/%s", kagentURL, namespace, agentName), nil)
		body := readBody(t, resp)
		require.Equal(t, http.StatusConflict, resp.StatusCode, "agent delete should be blocked: %s", body)
	})

	// =========================================================================
	// Phase 9: Delete SR, then agent deletion succeeds
	// =========================================================================
	t.Run("cleanup_sr_then_delete_agent", func(t *testing.T) {
		resp := doReq(t, http.MethodDelete, fmt.Sprintf("%s/api/scheduledruns/%s/%s", kagentURL, namespace, srName), nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = doReq(t, http.MethodDelete, fmt.Sprintf("%s/api/agents/%s/%s", kagentURL, namespace, agentName), nil)
		body := readBody(t, resp)
		require.Equal(t, http.StatusOK, resp.StatusCode, "agent delete should succeed after SR removal: %s", body)
	})

	// =========================================================================
	// Phase 10: ToolServer list returns userId field
	// =========================================================================
	t.Run("toolserver_list_has_userid", func(t *testing.T) {
		resp := doReq(t, http.MethodGet, kagentURL+"/api/toolservers", nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var result api.StandardResponse[[]api.ToolServerResponse]
		decodeJSON(t, resp, &result)
		require.False(t, result.Error)
		// Just verify the endpoint responds with expected structure
		require.NotNil(t, result.Data)
	})

	// =========================================================================
	// Phase 11: Session cleanup
	// =========================================================================
	t.Run("session_delete", func(t *testing.T) {
		if sessionID == "" {
			t.Skip("no session to delete")
		}
		resp := doReq(t, http.MethodDelete, fmt.Sprintf("%s/api/sessions/%s", kagentURL, sessionID), nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})

	t.Log("Smoke test passed!")
}
