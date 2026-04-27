package handlers

import (
	"net/http"
	"strconv"

	api "github.com/kagent-dev/kagent/go/api/httpapi"
	"github.com/kagent-dev/kagent/go/api/v1alpha2"
	"github.com/kagent-dev/kagent/go/core/internal/httpserver/errors"
	common "github.com/kagent-dev/kagent/go/core/internal/utils"
	"github.com/kagent-dev/kagent/go/core/pkg/auth"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// ScheduledRunTrigger is the interface for triggering scheduled runs manually
type ScheduledRunTrigger interface {
	TriggerManualRun(key types.NamespacedName)
}

// ScheduledRunsHandler handles ScheduledRun-related requests
type ScheduledRunsHandler struct {
	*Base
	Scheduler ScheduledRunTrigger
}

// NewScheduledRunsHandler creates a new ScheduledRunsHandler
func NewScheduledRunsHandler(base *Base, scheduler ScheduledRunTrigger) *ScheduledRunsHandler {
	return &ScheduledRunsHandler{Base: base, Scheduler: scheduler}
}

// hasAccess returns true if the given userID has access to the resource described by its annotations.
// Legacy resources without annotations are considered accessible by all users.
func hasAccess(annotations map[string]string, userID string) bool {
	if annotations == nil {
		return true
	}
	ownerID := annotations[common.AgentUserIDAnnotation]
	if ownerID == "" || ownerID == userID {
		return true
	}
	privateMode := annotations[common.AgentPrivateModeAnnotation]
	return privateMode == "false" // public items are accessible to all
}

// HandleListScheduledRuns handles GET /api/scheduledruns requests
func (h *ScheduledRunsHandler) HandleListScheduledRuns(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("scheduledruns-handler").WithValues("operation", "list")

	if err := Check(h.Authorizer, r, auth.Resource{Type: "ScheduledRun"}); err != nil {
		w.RespondWithError(err)
		return
	}

	userID, err := GetUserID(r)
	if err != nil {
		w.RespondWithError(errors.NewBadRequestError("Failed to get user ID", err))
		return
	}

	scheduledRunList := &v1alpha2.ScheduledRunList{}
	if err := h.KubeClient.List(r.Context(), scheduledRunList); err != nil {
		w.RespondWithError(errors.NewInternalServerError("Failed to list ScheduledRuns", err))
		return
	}

	filtered := make([]v1alpha2.ScheduledRun, 0, len(scheduledRunList.Items))
	for _, sr := range scheduledRunList.Items {
		if hasAccess(sr.GetAnnotations(), userID) {
			filtered = append(filtered, sr)
		}
	}

	log.Info("Successfully listed ScheduledRuns", "count", len(filtered))
	data := api.NewResponse(filtered, "Successfully listed ScheduledRuns", false)
	RespondWithJSON(w, http.StatusOK, data)
}

// HandleGetScheduledRun handles GET /api/scheduledruns/{namespace}/{name} requests
func (h *ScheduledRunsHandler) HandleGetScheduledRun(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("scheduledruns-handler").WithValues("operation", "get")

	namespace, err := GetPathParam(r, "namespace")
	if err != nil {
		w.RespondWithError(errors.NewBadRequestError("Failed to get namespace from path", err))
		return
	}

	name, err := GetPathParam(r, "name")
	if err != nil {
		w.RespondWithError(errors.NewBadRequestError("Failed to get name from path", err))
		return
	}

	log = log.WithValues("namespace", namespace, "name", name)

	if err := Check(h.Authorizer, r, auth.Resource{Type: "ScheduledRun", Name: namespace + "/" + name}); err != nil {
		w.RespondWithError(err)
		return
	}

	userID, err := GetUserID(r)
	if err != nil {
		w.RespondWithError(errors.NewBadRequestError("Failed to get user ID", err))
		return
	}

	sr := &v1alpha2.ScheduledRun{}
	if err := h.KubeClient.Get(r.Context(), client.ObjectKey{Namespace: namespace, Name: name}, sr); err != nil {
		if apierrors.IsNotFound(err) {
			w.RespondWithError(errors.NewNotFoundError("ScheduledRun not found", err))
			return
		}
		w.RespondWithError(errors.NewInternalServerError("Failed to get ScheduledRun", err))
		return
	}

	if !hasAccess(sr.GetAnnotations(), userID) {
		w.RespondWithError(errors.NewForbiddenError("Not authorized to access this ScheduledRun", nil))
		return
	}

	log.Info("Successfully retrieved ScheduledRun")
	data := api.NewResponse(sr, "Successfully retrieved ScheduledRun", false)
	RespondWithJSON(w, http.StatusOK, data)
}

// HandleCreateScheduledRun handles POST /api/scheduledruns requests
func (h *ScheduledRunsHandler) HandleCreateScheduledRun(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("scheduledruns-handler").WithValues("operation", "create")

	if err := Check(h.Authorizer, r, auth.Resource{Type: "ScheduledRun"}); err != nil {
		w.RespondWithError(err)
		return
	}

	userID, err := GetUserID(r)
	if err != nil {
		w.RespondWithError(errors.NewBadRequestError("Failed to get user ID", err))
		return
	}

	var sr v1alpha2.ScheduledRun
	if err := DecodeJSONBody(r, &sr); err != nil {
		w.RespondWithError(errors.NewBadRequestError("Invalid request body", err))
		return
	}

	if sr.Namespace == "" {
		sr.Namespace = common.GetResourceNamespace()
	}

	log = log.WithValues("namespace", sr.Namespace, "name", sr.Name)

	annotations := sr.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[common.AgentUserIDAnnotation] = userID
	// Respect an explicit private-mode from the request body; default to true (private).
	if _, ok := annotations[common.AgentPrivateModeAnnotation]; !ok {
		annotations[common.AgentPrivateModeAnnotation] = strconv.FormatBool(common.DefaultAgentPrivateMode)
	}
	sr.SetAnnotations(annotations)

	if err := h.KubeClient.Create(r.Context(), &sr); err != nil {
		w.RespondWithError(errors.NewInternalServerError("Failed to create ScheduledRun", err))
		return
	}

	log.Info("Successfully created ScheduledRun")
	data := api.NewResponse(sr, "Successfully created ScheduledRun", false)
	RespondWithJSON(w, http.StatusCreated, data)
}

// HandleUpdateScheduledRun handles PUT /api/scheduledruns/{namespace}/{name} requests
func (h *ScheduledRunsHandler) HandleUpdateScheduledRun(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("scheduledruns-handler").WithValues("operation", "update")

	namespace, err := GetPathParam(r, "namespace")
	if err != nil {
		w.RespondWithError(errors.NewBadRequestError("Failed to get namespace from path", err))
		return
	}

	name, err := GetPathParam(r, "name")
	if err != nil {
		w.RespondWithError(errors.NewBadRequestError("Failed to get name from path", err))
		return
	}

	log = log.WithValues("namespace", namespace, "name", name)

	if err := Check(h.Authorizer, r, auth.Resource{Type: "ScheduledRun", Name: namespace + "/" + name}); err != nil {
		w.RespondWithError(err)
		return
	}

	userID, err := GetUserID(r)
	if err != nil {
		w.RespondWithError(errors.NewBadRequestError("Failed to get user ID", err))
		return
	}

	var incoming v1alpha2.ScheduledRun
	if err := DecodeJSONBody(r, &incoming); err != nil {
		w.RespondWithError(errors.NewBadRequestError("Invalid request body", err))
		return
	}

	existing := &v1alpha2.ScheduledRun{}
	if err := h.KubeClient.Get(r.Context(), client.ObjectKey{Namespace: namespace, Name: name}, existing); err != nil {
		if apierrors.IsNotFound(err) {
			w.RespondWithError(errors.NewNotFoundError("ScheduledRun not found", err))
			return
		}
		w.RespondWithError(errors.NewInternalServerError("Failed to get ScheduledRun", err))
		return
	}

	ownerID := ""
	if ann := existing.GetAnnotations(); ann != nil {
		ownerID = ann[common.AgentUserIDAnnotation]
	}
	if ownerID != "" && ownerID != userID {
		w.RespondWithError(errors.NewForbiddenError("Not authorized to update this ScheduledRun", nil))
		return
	}

	existing.Spec = incoming.Spec

	// Preserve the owner user-id annotation; allow updating private-mode if provided.
	existingAnnotations := existing.GetAnnotations()
	if existingAnnotations == nil {
		existingAnnotations = make(map[string]string)
	}
	existingAnnotations[common.AgentUserIDAnnotation] = userID
	if incomingAnnotations := incoming.GetAnnotations(); incomingAnnotations != nil {
		if rawPrivate, ok := incomingAnnotations[common.AgentPrivateModeAnnotation]; ok {
			existingAnnotations[common.AgentPrivateModeAnnotation] = rawPrivate
		}
	}
	existing.SetAnnotations(existingAnnotations)

	if err := h.KubeClient.Update(r.Context(), existing); err != nil {
		w.RespondWithError(errors.NewInternalServerError("Failed to update ScheduledRun", err))
		return
	}

	log.Info("Successfully updated ScheduledRun")
	data := api.NewResponse(existing, "Successfully updated ScheduledRun", false)
	RespondWithJSON(w, http.StatusOK, data)
}

// HandleDeleteScheduledRun handles DELETE /api/scheduledruns/{namespace}/{name} requests
func (h *ScheduledRunsHandler) HandleDeleteScheduledRun(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("scheduledruns-handler").WithValues("operation", "delete")

	namespace, err := GetPathParam(r, "namespace")
	if err != nil {
		w.RespondWithError(errors.NewBadRequestError("Failed to get namespace from path", err))
		return
	}

	name, err := GetPathParam(r, "name")
	if err != nil {
		w.RespondWithError(errors.NewBadRequestError("Failed to get name from path", err))
		return
	}

	log = log.WithValues("namespace", namespace, "name", name)

	if err := Check(h.Authorizer, r, auth.Resource{Type: "ScheduledRun", Name: namespace + "/" + name}); err != nil {
		w.RespondWithError(err)
		return
	}

	userID, err := GetUserID(r)
	if err != nil {
		w.RespondWithError(errors.NewBadRequestError("Failed to get user ID", err))
		return
	}

	sr := &v1alpha2.ScheduledRun{}
	if err := h.KubeClient.Get(r.Context(), client.ObjectKey{Namespace: namespace, Name: name}, sr); err != nil {
		if apierrors.IsNotFound(err) {
			w.RespondWithError(errors.NewNotFoundError("ScheduledRun not found", err))
			return
		}
		w.RespondWithError(errors.NewInternalServerError("Failed to get ScheduledRun", err))
		return
	}

	ownerID := ""
	if ann := sr.GetAnnotations(); ann != nil {
		ownerID = ann[common.AgentUserIDAnnotation]
	}
	if ownerID != "" && ownerID != userID {
		w.RespondWithError(errors.NewForbiddenError("Not authorized to delete this ScheduledRun", nil))
		return
	}

	if err := h.KubeClient.Delete(r.Context(), sr); err != nil {
		w.RespondWithError(errors.NewInternalServerError("Failed to delete ScheduledRun", err))
		return
	}

	log.Info("Successfully deleted ScheduledRun")
	data := api.NewResponse(struct{}{}, "Successfully deleted ScheduledRun", false)
	RespondWithJSON(w, http.StatusOK, data)
}

// HandleTriggerScheduledRun handles POST /api/scheduledruns/{namespace}/{name}/trigger requests
func (h *ScheduledRunsHandler) HandleTriggerScheduledRun(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("scheduledruns-handler").WithValues("operation", "trigger")

	namespace, err := GetPathParam(r, "namespace")
	if err != nil {
		w.RespondWithError(errors.NewBadRequestError("Failed to get namespace from path", err))
		return
	}

	name, err := GetPathParam(r, "name")
	if err != nil {
		w.RespondWithError(errors.NewBadRequestError("Failed to get name from path", err))
		return
	}

	log = log.WithValues("namespace", namespace, "name", name)

	if err := Check(h.Authorizer, r, auth.Resource{Type: "ScheduledRun", Name: namespace + "/" + name}); err != nil {
		w.RespondWithError(err)
		return
	}

	userID, err := GetUserID(r)
	if err != nil {
		w.RespondWithError(errors.NewBadRequestError("Failed to get user ID", err))
		return
	}

	sr := &v1alpha2.ScheduledRun{}
	if err := h.KubeClient.Get(r.Context(), client.ObjectKey{Namespace: namespace, Name: name}, sr); err != nil {
		if apierrors.IsNotFound(err) {
			w.RespondWithError(errors.NewNotFoundError("ScheduledRun not found", err))
			return
		}
		w.RespondWithError(errors.NewInternalServerError("Failed to get ScheduledRun", err))
		return
	}

	if !hasAccess(sr.GetAnnotations(), userID) {
		w.RespondWithError(errors.NewForbiddenError("Not authorized to trigger this ScheduledRun", nil))
		return
	}

	log.Info("Manually triggering ScheduledRun")
	h.Scheduler.TriggerManualRun(types.NamespacedName{Namespace: namespace, Name: name})
	data := api.NewResponse(struct{}{}, "ScheduledRun triggered successfully", false)
	RespondWithJSON(w, http.StatusAccepted, data)
}
