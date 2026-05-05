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

	"github.com/kagent-dev/kagent/go/api/v1alpha2"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestE2ERefProtection verifies that:
// 1. An agent referenced by an active ScheduledRun cannot be deleted (409)
// 2. An agent referenced by an active ScheduledRun cannot be edited (409)
// 3. Suspending the ScheduledRun allows editing (200)
// 4. Deleting the ScheduledRun allows agent deletion (200)
// 5. ScheduledRuns with sub-hour frequency are rejected (400)
func TestE2ERefProtection(t *testing.T) {
	kagentURL := os.Getenv("KAGENT_URL")
	if kagentURL == "" {
		kagentURL = "http://localhost:8083"
	}

	namespace := "dbci-agent"
	agentName := "refprot-test-agent"
	srName := "refprot-test-sr"
	modelConfig := "sap-aicore-claude-46-sonnet" // must exist in the namespace

	httpClient := &http.Client{Timeout: 10 * time.Second}

	// Helper: make HTTP request
	doReq := func(t *testing.T, method, url string, body interface{}) *http.Response {
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
		req.Header.Set("X-Forwarded-Email", "test-refprot@example.com")
		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		return resp
	}

	// Cleanup function
	cleanup := func() {
		doReq(t, http.MethodDelete, fmt.Sprintf("%s/api/scheduledruns/%s/%s", kagentURL, namespace, srName), nil)
		doReq(t, http.MethodDelete, fmt.Sprintf("%s/api/agents/%s/%s", kagentURL, namespace, agentName), nil)
	}

	// Clean up before and after
	cleanup()
	t.Cleanup(cleanup)

	// --- Phase 1: Create test agent ---
	t.Log("Phase 1: Creating test agent")
	agent := v1alpha2.Agent{
		TypeMeta: metav1.TypeMeta{APIVersion: "kagent.dev/v1alpha2", Kind: "Agent"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      agentName,
			Namespace: namespace,
		},
		Spec: v1alpha2.AgentSpec{
			Type:        v1alpha2.AgentType_Declarative,
			Description: "Ref protection smoke test agent",
			Declarative: &v1alpha2.DeclarativeAgentSpec{
				ModelConfig:   modelConfig,
				SystemMessage: "You are a test agent.",
			},
		},
	}
	resp := doReq(t, http.MethodPost, kagentURL+"/api/agents", agent)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "failed to create agent")
	resp.Body.Close()

	// --- Phase 2: Reject sub-hour schedule ---
	t.Log("Phase 2: Verifying sub-hour schedule is rejected")
	srBad := v1alpha2.ScheduledRun{
		ObjectMeta: metav1.ObjectMeta{Name: srName, Namespace: namespace},
		Spec: v1alpha2.ScheduledRunSpec{
			Schedule:      "*/5 * * * *", // every 5 min — should be rejected
			AgentRef:      v1alpha2.AgentReference{Name: agentName, Namespace: namespace},
			Prompt:        "test prompt",
			MaxRunHistory: 5,
		},
	}
	resp = doReq(t, http.MethodPost, kagentURL+"/api/scheduledruns", srBad)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "sub-hour schedule should be rejected")
	resp.Body.Close()

	// --- Phase 3: Create valid ScheduledRun (hourly) ---
	t.Log("Phase 3: Creating valid ScheduledRun (hourly, active)")
	sr := v1alpha2.ScheduledRun{
		ObjectMeta: metav1.ObjectMeta{Name: srName, Namespace: namespace},
		Spec: v1alpha2.ScheduledRunSpec{
			Schedule:      "0 * * * *", // every hour
			AgentRef:      v1alpha2.AgentReference{Name: agentName, Namespace: namespace},
			Prompt:        "test prompt",
			Suspend:       false,
			MaxRunHistory: 5,
		},
	}
	resp = doReq(t, http.MethodPost, kagentURL+"/api/scheduledruns", sr)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "failed to create scheduled run")
	resp.Body.Close()

	// --- Phase 4: Attempt to delete agent — should be blocked (409) ---
	t.Log("Phase 4: Verifying agent deletion is blocked")
	resp = doReq(t, http.MethodDelete, fmt.Sprintf("%s/api/agents/%s/%s", kagentURL, namespace, agentName), nil)
	require.Equal(t, http.StatusConflict, resp.StatusCode, "agent delete should be blocked by active ScheduledRun")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	t.Logf("Delete blocked: %s", string(body))

	// --- Phase 5: Attempt to edit agent — should be blocked (409) ---
	t.Log("Phase 5: Verifying agent edit is blocked")
	agent.Spec.Description = "Updated description"
	resp = doReq(t, http.MethodPut, kagentURL+"/api/agents", agent)
	require.Equal(t, http.StatusConflict, resp.StatusCode, "agent edit should be blocked by active ScheduledRun")
	resp.Body.Close()

	// --- Phase 6: Suspend ScheduledRun → edit should succeed ---
	t.Log("Phase 6: Suspending ScheduledRun, then editing agent")
	sr.Spec.Suspend = true
	resp = doReq(t, http.MethodPut, fmt.Sprintf("%s/api/scheduledruns/%s/%s", kagentURL, namespace, srName), sr)
	require.Equal(t, http.StatusOK, resp.StatusCode, "failed to suspend scheduled run")
	resp.Body.Close()

	agent.Spec.Description = "Updated after suspend"
	resp = doReq(t, http.MethodPut, kagentURL+"/api/agents", agent)
	require.Equal(t, http.StatusOK, resp.StatusCode, "agent edit should succeed after suspending ScheduledRun")
	resp.Body.Close()

	// --- Phase 7: Delete still blocked (suspended SR still references) ---
	t.Log("Phase 7: Verifying agent deletion still blocked (suspended ref exists)")
	resp = doReq(t, http.MethodDelete, fmt.Sprintf("%s/api/agents/%s/%s", kagentURL, namespace, agentName), nil)
	require.Equal(t, http.StatusConflict, resp.StatusCode, "agent delete should still be blocked by suspended ScheduledRun")
	resp.Body.Close()

	// --- Phase 8: Delete ScheduledRun → agent deletion succeeds ---
	t.Log("Phase 8: Deleting ScheduledRun, then deleting agent")
	resp = doReq(t, http.MethodDelete, fmt.Sprintf("%s/api/scheduledruns/%s/%s", kagentURL, namespace, srName), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "failed to delete scheduled run")
	resp.Body.Close()

	resp = doReq(t, http.MethodDelete, fmt.Sprintf("%s/api/agents/%s/%s", kagentURL, namespace, agentName), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "agent delete should succeed after removing ScheduledRun")
	resp.Body.Close()

	t.Log("All reference protection checks passed!")
}
