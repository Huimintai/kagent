package handlers

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	api "github.com/kagent-dev/kagent/go/api/httpapi"
	"github.com/kagent-dev/kagent/go/core/internal/httpserver/errors"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// StatsHandler handles stats-related requests
type StatsHandler struct {
	*Base
}

// NewStatsHandler creates a new StatsHandler
func NewStatsHandler(base *Base) *StatsHandler {
	return &StatsHandler{Base: base}
}

// HandleGetStats handles GET /api/stats requests
func (h *StatsHandler) HandleGetStats(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("stats-handler").WithValues("operation", "get-stats")

	// Parse optional ?limit=10 query param (default 10, max 50)
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil || parsed < 1 {
			w.RespondWithError(errors.NewBadRequestError("Invalid limit parameter: must be a positive integer", nil))
			return
		}
		limit = parsed
	}
	if limit > 50 {
		limit = 50
	}

	excludePattern := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("excludePattern")))

	stats, err := h.DatabaseService.GetStats(r.Context(), limit)
	if err != nil {
		log.Error(err, "Failed to get stats")
		w.RespondWithError(errors.NewInternalServerError("Failed to get platform stats", err))
		return
	}

	// Map DB results to API types, filtering out excluded agents and computing scores
	topAgents := make([]api.AgentStat, 0, len(stats.TopAgents))
	for _, a := range stats.TopAgents {
		if excludePattern != "" && strings.Contains(strings.ToLower(a.AgentID), excludePattern) {
			continue
		}
		score := computePopularityScore(a.UserCount, a.SessionCount, a.MessageCount, a.LastActiveAt)
		topAgents = append(topAgents, api.AgentStat{
			AgentID:      a.AgentID,
			UserCount:    a.UserCount,
			SessionCount: a.SessionCount,
			MessageCount: a.MessageCount,
			Score:        score,
			LastActiveAt: formatTimePtr(a.LastActiveAt),
		})
	}

	// Sort by score descending (already ordered by user_count from DB, but score reranks)
	sortAgentsByScore(topAgents)

	topMCPs := make([]api.ToolServerStat, len(stats.TopMCPs))
	for i, ts := range stats.TopMCPs {
		topMCPs[i] = api.ToolServerStat{
			Name:          ts.Name,
			GroupKind:     ts.GroupKind,
			AgentCount:    ts.AgentCount,
			LastConnected: formatTimePtr(ts.LastConnected),
		}
	}

	response := api.StatsResponse{
		Summary: api.PlatformSummary{
			TotalAgents:      stats.Summary.TotalAgents,
			TotalSessions:    stats.Summary.TotalSessions,
			TotalToolServers: stats.Summary.TotalToolServers,
			SessionsToday:    stats.Summary.SessionsToday,
		},
		TopAgents: topAgents,
		TopMCPs:   topMCPs,
	}

	log.Info("Successfully retrieved platform stats")
	RespondWithJSON(w, http.StatusOK, api.NewResponse(response, "Successfully retrieved platform stats", false))
}

// computePopularityScore calculates a composite popularity score.
// Weights: users (60%), engagement depth (25%), recency (15%).
// This ensures agents with more diverse users rank higher, while
// repeat usage and fresh activity serve as tiebreakers.
func computePopularityScore(userCount, sessionCount, messageCount int64, lastActiveAt *time.Time) float64 {
	// User reach: primary signal — log-scaled to avoid runaway dominance
	userScore := math.Log2(float64(userCount)+1) * 10.0

	// Engagement: avg messages per session indicates conversation depth
	var engagementScore float64
	if sessionCount > 0 {
		avgMsgsPerSession := float64(messageCount) / float64(sessionCount)
		engagementScore = math.Log2(avgMsgsPerSession+1) * 4.0
	}

	// Repeat usage: sessions per user indicates stickiness
	var stickinessScore float64
	if userCount > 0 {
		sessionsPerUser := float64(sessionCount) / float64(userCount)
		stickinessScore = math.Log2(sessionsPerUser+1) * 3.0
	}

	// Recency: bonus for agents active in the last 7 days, decays linearly
	var recencyScore float64
	if lastActiveAt != nil {
		hoursSince := time.Since(*lastActiveAt).Hours()
		if hoursSince < 168 { // 7 days
			recencyScore = (1.0 - hoursSince/168.0) * 5.0
		}
	}

	return userScore + engagementScore + stickinessScore + recencyScore
}

// sortAgentsByScore sorts agents by score descending.
func sortAgentsByScore(agents []api.AgentStat) {
	for i := 1; i < len(agents); i++ {
		for j := i; j > 0 && agents[j].Score > agents[j-1].Score; j-- {
			agents[j], agents[j-1] = agents[j-1], agents[j]
		}
	}
}

// formatTimePtr formats a *time.Time to an RFC3339 string pointer.
func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}
