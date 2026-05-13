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

package controller

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/kagent-dev/kagent/go/api/v1alpha2"
	"github.com/kagent-dev/kagent/go/core/internal/broker"
)

const (
	// platformCredentialRequeueInterval is how often the controller re-validates credentials.
	platformCredentialRequeueInterval = 5 * time.Minute
)

// PlatformCredentialController reconciles PlatformCredential resources.
// It validates the credential source using the appropriate adapter and updates status conditions.
type PlatformCredentialController struct {
	client.Client
	Scheme *runtime.Scheme
	Broker *broker.DefaultBroker
}

// +kubebuilder:rbac:groups=kagent.dev,resources=platformcredentials,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kagent.dev,resources=platformcredentials/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kagent.dev,resources=platformcredentials/finalizers,verbs=update

func (r *PlatformCredentialController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("platformcredential-controller")

	var cred v1alpha2.PlatformCredential
	if err := r.Get(ctx, req.NamespacedName, &cred); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Reconciling PlatformCredential", "name", cred.Name, "platform", cred.Spec.Platform)

	// Validate the credential source using the broker's adapter registry.
	validationErr := r.Broker.ValidateCredential(cred.Spec.Platform, cred.Spec.Source)

	now := metav1.Now()
	condition := metav1.Condition{
		Type:               v1alpha2.PlatformCredentialConditionTypeReady,
		ObservedGeneration: cred.Generation,
		LastTransitionTime: now,
	}

	if validationErr != nil {
		condition.Status = metav1.ConditionFalse
		condition.Reason = "ValidationFailed"
		condition.Message = validationErr.Error()
		logger.Info("PlatformCredential validation failed", "name", cred.Name, "error", validationErr)
	} else {
		condition.Status = metav1.ConditionTrue
		condition.Reason = "Valid"
		condition.Message = "Credential source validated successfully"
		cred.Status.LastSync = &now
	}

	// Update condition
	setCondition(&cred.Status.Conditions, condition)

	if err := r.Status().Update(ctx, &cred); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update PlatformCredential status: %w", err)
	}

	// Requeue periodically to re-validate credential validity.
	return ctrl.Result{RequeueAfter: platformCredentialRequeueInterval}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PlatformCredentialController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			NeedLeaderElection: new(true),
		}).
		For(&v1alpha2.PlatformCredential{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Named("platformcredential").
		Complete(r)
}

// setCondition sets or updates a condition in the conditions slice.
func setCondition(conditions *[]metav1.Condition, condition metav1.Condition) {
	if conditions == nil {
		return
	}
	for i, c := range *conditions {
		if c.Type == condition.Type {
			// Only update LastTransitionTime if the status changed.
			if c.Status == condition.Status {
				condition.LastTransitionTime = c.LastTransitionTime
			}
			(*conditions)[i] = condition
			return
		}
	}
	*conditions = append(*conditions, condition)
}
