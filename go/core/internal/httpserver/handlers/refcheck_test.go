package handlers_test

import (
	"context"
	"testing"

	"github.com/kagent-dev/kagent/go/api/v1alpha2"
	"github.com/kagent-dev/kagent/go/core/internal/httpserver/handlers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newTestScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	v1alpha2.AddToScheme(s)
	return s
}

func TestScheduledRunReferenceSource(t *testing.T) {
	src := &handlers.ScheduledRunReferenceSource{}

	tests := []struct {
		name      string
		agentRef  types.NamespacedName
		objects   []v1alpha2.ScheduledRun
		wantCount int
		wantActive int
	}{
		{
			name:     "no references",
			agentRef: types.NamespacedName{Namespace: "default", Name: "my-agent"},
			objects:  nil,
			wantCount: 0,
		},
		{
			name:     "active reference matches",
			agentRef: types.NamespacedName{Namespace: "default", Name: "my-agent"},
			objects: []v1alpha2.ScheduledRun{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "sr1", Namespace: "default"},
					Spec: v1alpha2.ScheduledRunSpec{
						AgentRef: v1alpha2.AgentReference{Name: "my-agent"},
						Suspend:  false,
					},
				},
			},
			wantCount:  1,
			wantActive: 1,
		},
		{
			name:     "suspended reference matches",
			agentRef: types.NamespacedName{Namespace: "default", Name: "my-agent"},
			objects: []v1alpha2.ScheduledRun{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "sr1", Namespace: "default"},
					Spec: v1alpha2.ScheduledRunSpec{
						AgentRef: v1alpha2.AgentReference{Name: "my-agent"},
						Suspend:  true,
					},
				},
			},
			wantCount:  1,
			wantActive: 0,
		},
		{
			name:     "namespace mismatch does not match",
			agentRef: types.NamespacedName{Namespace: "production", Name: "my-agent"},
			objects: []v1alpha2.ScheduledRun{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "sr1", Namespace: "default"},
					Spec: v1alpha2.ScheduledRunSpec{
						AgentRef: v1alpha2.AgentReference{Name: "my-agent"},
					},
				},
			},
			wantCount: 0,
		},
		{
			name:     "explicit namespace in AgentRef matches",
			agentRef: types.NamespacedName{Namespace: "production", Name: "my-agent"},
			objects: []v1alpha2.ScheduledRun{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "sr1", Namespace: "default"},
					Spec: v1alpha2.ScheduledRunSpec{
						AgentRef: v1alpha2.AgentReference{Name: "my-agent", Namespace: "production"},
					},
				},
			},
			wantCount:  1,
			wantActive: 1,
		},
		{
			name:     "multiple references",
			agentRef: types.NamespacedName{Namespace: "default", Name: "my-agent"},
			objects: []v1alpha2.ScheduledRun{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "sr1", Namespace: "default"},
					Spec: v1alpha2.ScheduledRunSpec{
						AgentRef: v1alpha2.AgentReference{Name: "my-agent"},
						Suspend:  false,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "sr2", Namespace: "default"},
					Spec: v1alpha2.ScheduledRunSpec{
						AgentRef: v1alpha2.AgentReference{Name: "my-agent"},
						Suspend:  true,
					},
				},
			},
			wantCount:  2,
			wantActive: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objs := make([]runtime.Object, len(tt.objects))
			for i := range tt.objects {
				objs[i] = &tt.objects[i]
			}
			kubeClient := fake.NewClientBuilder().WithScheme(newTestScheme()).WithRuntimeObjects(objs...).Build()

			refs, err := src.FindReferencesTo(context.Background(), kubeClient, tt.agentRef)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(refs) != tt.wantCount {
				t.Errorf("got %d references, want %d", len(refs), tt.wantCount)
			}
			activeCount := 0
			for _, r := range refs {
				if r.Active {
					activeCount++
				}
			}
			if activeCount != tt.wantActive {
				t.Errorf("got %d active references, want %d", activeCount, tt.wantActive)
			}
		})
	}
}

func TestAgentToolReferenceSource(t *testing.T) {
	src := &handlers.AgentToolReferenceSource{}

	tests := []struct {
		name      string
		agentRef  types.NamespacedName
		agents    []v1alpha2.Agent
		wantCount int
	}{
		{
			name:      "no agents reference target",
			agentRef:  types.NamespacedName{Namespace: "default", Name: "target-agent"},
			agents:    nil,
			wantCount: 0,
		},
		{
			name:     "agent references target as tool",
			agentRef: types.NamespacedName{Namespace: "default", Name: "target-agent"},
			agents: []v1alpha2.Agent{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "caller-agent", Namespace: "default"},
					Spec: v1alpha2.AgentSpec{
						Declarative: &v1alpha2.DeclarativeAgentSpec{
							Tools: []*v1alpha2.Tool{
								{
									Type: v1alpha2.ToolProviderType_Agent,
									Agent: &v1alpha2.TypedReference{
										Name: "target-agent",
									},
								},
							},
						},
					},
				},
			},
			wantCount: 1,
		},
		{
			name:     "agent does not reference target",
			agentRef: types.NamespacedName{Namespace: "default", Name: "target-agent"},
			agents: []v1alpha2.Agent{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "caller-agent", Namespace: "default"},
					Spec: v1alpha2.AgentSpec{
						Declarative: &v1alpha2.DeclarativeAgentSpec{
							Tools: []*v1alpha2.Tool{
								{
									Type: v1alpha2.ToolProviderType_Agent,
									Agent: &v1alpha2.TypedReference{
										Name: "other-agent",
									},
								},
							},
						},
					},
				},
			},
			wantCount: 0,
		},
		{
			name:     "self-reference is detected",
			agentRef: types.NamespacedName{Namespace: "default", Name: "caller-agent"},
			agents: []v1alpha2.Agent{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "other-agent", Namespace: "default"},
					Spec: v1alpha2.AgentSpec{
						Declarative: &v1alpha2.DeclarativeAgentSpec{
							Tools: []*v1alpha2.Tool{
								{
									Type: v1alpha2.ToolProviderType_Agent,
									Agent: &v1alpha2.TypedReference{
										Name: "caller-agent",
									},
								},
							},
						},
					},
				},
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objs := make([]runtime.Object, len(tt.agents))
			for i := range tt.agents {
				objs[i] = &tt.agents[i]
			}
			kubeClient := fake.NewClientBuilder().WithScheme(newTestScheme()).WithRuntimeObjects(objs...).Build()

			refs, err := src.FindReferencesTo(context.Background(), kubeClient, tt.agentRef)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(refs) != tt.wantCount {
				t.Errorf("got %d references, want %d", len(refs), tt.wantCount)
			}
			// Agent-as-Tool references are always active
			for _, r := range refs {
				if !r.Active {
					t.Error("expected Agent-as-Tool reference to be active")
				}
			}
		})
	}
}

func TestReferenceChecker_DeletionAllowed(t *testing.T) {
	checker := &handlers.ReferenceChecker{
		Sources: []handlers.ReferenceSource{
			&handlers.ScheduledRunReferenceSource{},
			&handlers.AgentToolReferenceSource{},
		},
	}

	tests := []struct {
		name     string
		agentRef types.NamespacedName
		objects  []runtime.Object
		wantErr  bool
	}{
		{
			name:     "no references - deletion allowed",
			agentRef: types.NamespacedName{Namespace: "default", Name: "my-agent"},
			objects:  nil,
			wantErr:  false,
		},
		{
			name:     "active ScheduledRun - deletion blocked",
			agentRef: types.NamespacedName{Namespace: "default", Name: "my-agent"},
			objects: []runtime.Object{
				&v1alpha2.ScheduledRun{
					ObjectMeta: metav1.ObjectMeta{Name: "sr1", Namespace: "default"},
					Spec:       v1alpha2.ScheduledRunSpec{AgentRef: v1alpha2.AgentReference{Name: "my-agent"}},
				},
			},
			wantErr: true,
		},
		{
			name:     "suspended ScheduledRun - deletion still blocked",
			agentRef: types.NamespacedName{Namespace: "default", Name: "my-agent"},
			objects: []runtime.Object{
				&v1alpha2.ScheduledRun{
					ObjectMeta: metav1.ObjectMeta{Name: "sr1", Namespace: "default"},
					Spec:       v1alpha2.ScheduledRunSpec{AgentRef: v1alpha2.AgentReference{Name: "my-agent"}, Suspend: true},
				},
			},
			wantErr: true,
		},
		{
			name:     "Agent-as-Tool - deletion blocked",
			agentRef: types.NamespacedName{Namespace: "default", Name: "target-agent"},
			objects: []runtime.Object{
				&v1alpha2.Agent{
					ObjectMeta: metav1.ObjectMeta{Name: "caller", Namespace: "default"},
					Spec: v1alpha2.AgentSpec{
						Declarative: &v1alpha2.DeclarativeAgentSpec{
							Tools: []*v1alpha2.Tool{{Type: v1alpha2.ToolProviderType_Agent, Agent: &v1alpha2.TypedReference{Name: "target-agent"}}},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kubeClient := fake.NewClientBuilder().WithScheme(newTestScheme()).WithRuntimeObjects(tt.objects...).Build()
			err := checker.CheckDeletionAllowed(context.Background(), kubeClient, tt.agentRef)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckDeletionAllowed() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReferenceChecker_EditAllowed(t *testing.T) {
	checker := &handlers.ReferenceChecker{
		Sources: []handlers.ReferenceSource{
			&handlers.ScheduledRunReferenceSource{},
			&handlers.AgentToolReferenceSource{},
		},
	}

	tests := []struct {
		name     string
		agentRef types.NamespacedName
		objects  []runtime.Object
		wantErr  bool
	}{
		{
			name:     "no references - edit allowed",
			agentRef: types.NamespacedName{Namespace: "default", Name: "my-agent"},
			objects:  nil,
			wantErr:  false,
		},
		{
			name:     "active ScheduledRun - edit blocked",
			agentRef: types.NamespacedName{Namespace: "default", Name: "my-agent"},
			objects: []runtime.Object{
				&v1alpha2.ScheduledRun{
					ObjectMeta: metav1.ObjectMeta{Name: "sr1", Namespace: "default"},
					Spec:       v1alpha2.ScheduledRunSpec{AgentRef: v1alpha2.AgentReference{Name: "my-agent"}},
				},
			},
			wantErr: true,
		},
		{
			name:     "suspended ScheduledRun - edit allowed",
			agentRef: types.NamespacedName{Namespace: "default", Name: "my-agent"},
			objects: []runtime.Object{
				&v1alpha2.ScheduledRun{
					ObjectMeta: metav1.ObjectMeta{Name: "sr1", Namespace: "default"},
					Spec:       v1alpha2.ScheduledRunSpec{AgentRef: v1alpha2.AgentReference{Name: "my-agent"}, Suspend: true},
				},
			},
			wantErr: false,
		},
		{
			name:     "Agent-as-Tool - edit blocked",
			agentRef: types.NamespacedName{Namespace: "default", Name: "target-agent"},
			objects: []runtime.Object{
				&v1alpha2.Agent{
					ObjectMeta: metav1.ObjectMeta{Name: "caller", Namespace: "default"},
					Spec: v1alpha2.AgentSpec{
						Declarative: &v1alpha2.DeclarativeAgentSpec{
							Tools: []*v1alpha2.Tool{{Type: v1alpha2.ToolProviderType_Agent, Agent: &v1alpha2.TypedReference{Name: "target-agent"}}},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kubeClient := fake.NewClientBuilder().WithScheme(newTestScheme()).WithRuntimeObjects(tt.objects...).Build()
			err := checker.CheckEditAllowed(context.Background(), kubeClient, tt.agentRef)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckEditAllowed() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateScheduleFrequency(t *testing.T) {
	tests := []struct {
		name     string
		schedule string
		wantErr  bool
	}{
		{name: "every minute - rejected", schedule: "* * * * *", wantErr: true},
		{name: "every 5 minutes - rejected", schedule: "*/5 * * * *", wantErr: true},
		{name: "every 30 minutes - rejected", schedule: "*/30 * * * *", wantErr: true},
		{name: "every hour - accepted", schedule: "0 * * * *", wantErr: false},
		{name: "every 2 hours - accepted", schedule: "0 */2 * * *", wantErr: false},
		{name: "daily at 9am - accepted", schedule: "0 9 * * *", wantErr: false},
		{name: "weekdays at 9am - accepted", schedule: "0 9 * * 1-5", wantErr: false},
		{name: "invalid expression - rejected", schedule: "invalid", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handlers.ValidateScheduleFrequency(tt.schedule)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateScheduleFrequency(%q) error = %v, wantErr %v", tt.schedule, err, tt.wantErr)
			}
		})
	}
}
