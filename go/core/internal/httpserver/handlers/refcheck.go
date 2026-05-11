package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/kagent-dev/kagent/go/api/v1alpha2"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReferenceSource represents a type of resource that may hold references to Agents.
type ReferenceSource interface {
	FindReferencesTo(ctx context.Context, kubeClient client.Client, agentRef types.NamespacedName) ([]Reference, error)
}

// Reference describes a single inbound reference to an agent.
type Reference struct {
	Kind      string // e.g. "ScheduledRun", "Agent"
	Name      string
	Namespace string
	Active    bool // true if the reference is "active" (non-suspended ScheduledRun, or any Agent-as-Tool)
}

func (r Reference) String() string {
	status := "active"
	if !r.Active {
		status = "suspended"
	}
	return fmt.Sprintf("%s %s/%s (%s)", r.Kind, r.Namespace, r.Name, status)
}

// ReferenceChecker aggregates multiple ReferenceSource implementations.
type ReferenceChecker struct {
	Sources []ReferenceSource
}

// CheckDeletionAllowed returns an error if any references (active or suspended) exist.
func (rc *ReferenceChecker) CheckDeletionAllowed(ctx context.Context, kubeClient client.Client, agentRef types.NamespacedName) error {
	if rc == nil {
		return nil
	}
	refs, err := rc.allReferences(ctx, kubeClient, agentRef)
	if err != nil {
		return fmt.Errorf("failed to check references: %w", err)
	}
	if len(refs) == 0 {
		return nil
	}
	return buildConflictError("delete", agentRef, refs)
}

// CheckEditAllowed returns an error if any ACTIVE (non-suspended) references exist.
func (rc *ReferenceChecker) CheckEditAllowed(ctx context.Context, kubeClient client.Client, agentRef types.NamespacedName) error {
	if rc == nil {
		return nil
	}
	refs, err := rc.allReferences(ctx, kubeClient, agentRef)
	if err != nil {
		return fmt.Errorf("failed to check references: %w", err)
	}
	active := filterActive(refs)
	if len(active) == 0 {
		return nil
	}
	return buildConflictError("edit", agentRef, active)
}

func (rc *ReferenceChecker) allReferences(ctx context.Context, kubeClient client.Client, agentRef types.NamespacedName) ([]Reference, error) {
	var all []Reference
	for _, src := range rc.Sources {
		refs, err := src.FindReferencesTo(ctx, kubeClient, agentRef)
		if err != nil {
			return nil, err
		}
		all = append(all, refs...)
	}
	return all, nil
}

func filterActive(refs []Reference) []Reference {
	var active []Reference
	for _, r := range refs {
		if r.Active {
			active = append(active, r)
		}
	}
	return active
}

func buildConflictError(action string, agentRef types.NamespacedName, refs []Reference) error {
	refStrs := make([]string, len(refs))
	for i, r := range refs {
		refStrs[i] = r.String()
	}

	var hint string
	switch action {
	case "delete":
		hint = "Delete or update these resources first."
	case "edit":
		hint = "Suspend the referencing ScheduledRuns or remove the Agent-as-Tool references first."
	}

	return fmt.Errorf("cannot %s Agent %q: referenced by %s. %s",
		action, agentRef.String(), strings.Join(refStrs, ", "), hint)
}

// ScheduledRunReferenceSource checks ScheduledRun resources for agent references.
type ScheduledRunReferenceSource struct{}

func (s *ScheduledRunReferenceSource) FindReferencesTo(ctx context.Context, kubeClient client.Client, agentRef types.NamespacedName) ([]Reference, error) {
	var srList v1alpha2.ScheduledRunList
	if err := kubeClient.List(ctx, &srList); err != nil {
		return nil, fmt.Errorf("failed to list ScheduledRuns: %w", err)
	}

	var refs []Reference
	for _, sr := range srList.Items {
		refNs := sr.Spec.AgentRef.Namespace
		if refNs == "" {
			refNs = sr.Namespace
		}
		if sr.Spec.AgentRef.Name == agentRef.Name && refNs == agentRef.Namespace {
			refs = append(refs, Reference{
				Kind:      "ScheduledRun",
				Name:      sr.Name,
				Namespace: sr.Namespace,
				Active:    !sr.Spec.Suspend,
			})
		}
	}
	return refs, nil
}

// AgentToolReferenceSource checks if other Agents reference the target agent as a Tool.
type AgentToolReferenceSource struct{}

func (a *AgentToolReferenceSource) FindReferencesTo(ctx context.Context, kubeClient client.Client, agentRef types.NamespacedName) ([]Reference, error) {
	var agentList v1alpha2.AgentList
	if err := kubeClient.List(ctx, &agentList); err != nil {
		return nil, fmt.Errorf("failed to list Agents: %w", err)
	}

	var refs []Reference
	for _, ag := range agentList.Items {
		spec := ag.GetAgentSpec()
		if spec == nil || spec.Declarative == nil {
			continue
		}
		for _, tool := range spec.Declarative.Tools {
			if tool.Type == v1alpha2.ToolProviderType_Agent && tool.Agent != nil {
				toolRef := tool.Agent.NamespacedName(ag.Namespace)
				if toolRef == agentRef {
					refs = append(refs, Reference{
						Kind:      "Agent",
						Name:      ag.Name,
						Namespace: ag.Namespace,
						Active:    true, // Agents are always considered active borrowers
					})
					break // one match per agent is enough
				}
			}
		}
	}
	return refs, nil
}
