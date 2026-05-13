package agent

import (
	"context"
	"fmt"
	"slices"

	"github.com/kagent-dev/kagent/go/api/adk"
	"github.com/kagent-dev/kagent/go/api/v1alpha2"
	"github.com/kagent-dev/kagent/go/core/internal/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"trpc.group/trpc-go/trpc-a2a-go/server"
)

// AgentManifestInputs holds the translated data needed to emit Kubernetes resources.
type AgentManifestInputs struct {
	Config          *adk.AgentConfig
	Sandbox         *v1alpha2.SandboxConfig
	Deployment      *resolvedDeployment
	AgentCard       *server.AgentCard
	SecretHashBytes []byte
}

const MAX_DEPTH = 10

type tState struct {
	// used to prevent infinite loops
	// The recursion limit is 10
	depth uint8
	// used to enforce DAG
	// The final member of the list will be the "parent" agent
	visitedAgents []string
}

func (s *tState) with(agent v1alpha2.AgentObject) *tState {
	visited := make([]string, len(s.visitedAgents), len(s.visitedAgents)+1)
	copy(visited, s.visitedAgents)
	visited = append(visited, utils.GetObjectRef(agent))
	return &tState{
		depth:         s.depth + 1,
		visitedAgents: visited,
	}
}

func (t *tState) isVisited(agentName string) bool {
	return slices.Contains(t.visitedAgents, agentName)
}

func TranslateAgent(
	ctx context.Context,
	translator AdkApiTranslator,
	agent v1alpha2.AgentObject,
) (*AgentOutputs, error) {
	inputs, err := translator.CompileAgent(ctx, agent)
	if err != nil {
		return nil, err
	}
	return translator.BuildManifest(ctx, agent, inputs)
}

func (a *adkApiTranslator) CompileAgent(
	ctx context.Context,
	agent v1alpha2.AgentObject,
) (*AgentManifestInputs, error) {
	spec := agent.GetAgentSpec()
	err := a.validateAgent(ctx, agent, &tState{})
	if err != nil {
		return nil, err
	}

	var cfg *adk.AgentConfig
	var dep *resolvedDeployment
	var secretHashBytes []byte

	switch spec.Type {
	case v1alpha2.AgentType_Declarative:
		var mdd *modelDeploymentData
		cfg, mdd, secretHashBytes, err = a.translateInlineAgent(ctx, agent)
		if err != nil {
			return nil, err
		}
		dep, err = resolveInlineDeployment(agent, mdd)
		if err != nil {
			return nil, err
		}

		// Auto-inject AI Core proxy sidecar when CLI runtimes reference a ModelConfig.
		if err := a.injectAICoreProxySidecar(ctx, agent, dep); err != nil {
			return nil, err
		}

	case v1alpha2.AgentType_BYO:
		dep, err = resolveByoDeployment(agent)
		if err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unknown agent type: %s", spec.Type)
	}

	runInSandbox := agent.GetWorkloadMode() == v1alpha2.WorkloadModeSandbox
	if runInSandbox && a.sandboxBackend == nil {
		return nil, fmt.Errorf("sandbox backend is not configured")
	}

	card := GetA2AAgentCard(agent)

	return &AgentManifestInputs{
		Config:          cfg,
		Sandbox:         spec.Sandbox,
		Deployment:      dep,
		AgentCard:       card,
		SecretHashBytes: secretHashBytes,
	}, nil
}

func (a *adkApiTranslator) validateAgent(ctx context.Context, agent v1alpha2.AgentObject, state *tState) error {
	agentRef := utils.GetObjectRef(agent)
	spec := agent.GetAgentSpec()

	if state.isVisited(agentRef) {
		return fmt.Errorf("cycle detected in agent tool chain: %s -> %s", agentRef, agentRef)
	}

	if state.depth > MAX_DEPTH {
		return fmt.Errorf("recursion limit reached in agent tool chain: %s -> %s", agentRef, agentRef)
	}

	if spec.Type != v1alpha2.AgentType_Declarative || spec.Declarative == nil {
		// We only need to validate loops in declarative agents
		return nil
	}

	for _, tool := range spec.Declarative.Tools {
		switch tool.Type {
		case v1alpha2.ToolProviderType_Agent:
			if tool.Agent == nil {
				return fmt.Errorf("tool must have an agent reference")
			}

			agentRef := tool.Agent.NamespacedName(agent.GetNamespace())

			if agentRef.Namespace == agent.GetNamespace() && agentRef.Name == agent.GetName() {
				return fmt.Errorf("agent tool cannot be used to reference itself, %s", agentRef)
			}

			toolAgent := &v1alpha2.Agent{}
			err := a.kube.Get(ctx, agentRef, toolAgent)
			if err != nil {
				return err
			}

			err = a.validateAgent(ctx, toolAgent, state.with(agent))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (a *adkApiTranslator) translateInlineAgent(ctx context.Context, agent v1alpha2.AgentObject) (*adk.AgentConfig, *modelDeploymentData, []byte, error) {
	spec := agent.GetAgentSpec()

	// Determine the effective runtime
	runtime := spec.Declarative.Runtime
	if runtime == "" {
		runtime = v1alpha2.DeclarativeRuntime_Python
	}

	// ClaudeCode and Codex runtimes manage their own model configuration via claudeCodeConfig/codexConfig
	// and do not require a ModelConfig resource.
	var model adk.Model
	var mdd *modelDeploymentData
	var secretHashBytes []byte
	if runtime == v1alpha2.DeclarativeRuntime_ClaudeCode || runtime == v1alpha2.DeclarativeRuntime_Codex {
		mdd = &modelDeploymentData{}
	} else {
		var err error
		model, mdd, secretHashBytes, err = a.translateModel(ctx, agent.GetNamespace(), spec.Declarative.ModelConfig)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	// Resolve the raw system message (template processing happens after tools are translated).
	rawSystemMessage, err := a.resolveRawSystemMessage(ctx, agent)
	if err != nil {
		return nil, nil, nil, err
	}

	cfg := &adk.AgentConfig{
		Description: spec.Description,
		Instruction: rawSystemMessage,
		Model:       model,
		ExecuteCode: spec.Declarative.ExecuteCodeBlocks,
		Stream:      new(spec.Declarative.Stream),
	}

	if spec.Sandbox != nil && spec.Sandbox.Network != nil {
		cfg.Network = &adk.NetworkConfig{
			AllowedDomains: append([]string(nil), spec.Sandbox.Network.AllowedDomains...),
		}
	}

	// Translate context management configuration
	if spec.Declarative.Context != nil {
		contextCfg := &adk.AgentContextConfig{}

		if spec.Declarative.Context.Compaction != nil {
			comp := spec.Declarative.Context.Compaction
			compCfg := &adk.AgentCompressionConfig{
				CompactionInterval: comp.CompactionInterval,
				OverlapSize:        comp.OverlapSize,
				TokenThreshold:     comp.TokenThreshold,
				EventRetentionSize: comp.EventRetentionSize,
			}

			if comp.Summarizer != nil {
				if comp.Summarizer.PromptTemplate != nil {
					compCfg.PromptTemplate = *comp.Summarizer.PromptTemplate
				}

				summarizerModelName := ""
				if comp.Summarizer.ModelConfig != nil {
					summarizerModelName = *comp.Summarizer.ModelConfig
				}

				if summarizerModelName == "" || summarizerModelName == spec.Declarative.ModelConfig {
					compCfg.SummarizerModel = model
				} else {
					summarizerModel, summarizerMdd, summarizerSecretHash, err := a.translateModel(ctx, agent.GetNamespace(), summarizerModelName)
					if err != nil {
						return nil, nil, nil, fmt.Errorf("failed to translate summarizer model config %q: %w", summarizerModelName, err)
					}
					compCfg.SummarizerModel = summarizerModel
					mergeDeploymentData(mdd, summarizerMdd)
					if len(summarizerSecretHash) > 0 {
						secretHashBytes = append(secretHashBytes, summarizerSecretHash...)
					}
				}
			}

			contextCfg.Compaction = compCfg
		}

		cfg.ContextConfig = contextCfg
	}

	// Handle Memory Configuration: presence of Memory field enables it.
	if spec.Declarative.Memory != nil {
		embCfg, embMdd, embHash, err := a.translateEmbeddingConfig(ctx, agent.GetNamespace(), spec.Declarative.Memory.ModelConfig)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to resolve embedding config: %w", err)
		}

		cfg.Memory = &adk.MemoryConfig{
			TTLDays:   spec.Declarative.Memory.TTLDays,
			Embedding: embCfg,
		}

		mergeDeploymentData(mdd, embMdd)
		if spec.Declarative.Memory.ModelConfig != spec.Declarative.ModelConfig {
			secretHashBytes = append(secretHashBytes, embHash...)
		}
	}

	for _, tool := range spec.Declarative.Tools {
		headers, err := tool.ResolveHeaders(ctx, a.kube, agent.GetNamespace())
		if err != nil {
			return nil, nil, nil, err
		}

		switch {
		case tool.McpServer != nil:
			err := a.translateMCPServerTarget(ctx, cfg, agent.GetNamespace(), tool.McpServer, headers, a.globalProxyURL)
			if err != nil {
				return nil, nil, nil, err
			}
		case tool.Agent != nil:
			agentRef := tool.Agent.NamespacedName(agent.GetNamespace())

			if agentRef.Namespace == agent.GetNamespace() && agentRef.Name == agent.GetName() {
				return nil, nil, nil, fmt.Errorf("agent tool cannot be used to reference itself, %s", agentRef)
			}

			toolAgent := &v1alpha2.Agent{}
			err := a.kube.Get(ctx, agentRef, toolAgent)
			if err != nil {
				return nil, nil, nil, err
			}

			switch toolAgent.Spec.Type {
			case v1alpha2.AgentType_BYO, v1alpha2.AgentType_Declarative:
				originalURL := fmt.Sprintf("http://%s.%s:8080", toolAgent.Name, toolAgent.Namespace)

				targetURL := originalURL
				if a.globalProxyURL != "" {
					targetURL, headers, err = applyProxyURL(originalURL, a.globalProxyURL, headers)
					if err != nil {
						return nil, nil, nil, err
					}
				}

				cfg.RemoteAgents = append(cfg.RemoteAgents, adk.RemoteAgentConfig{
					Name:        utils.ConvertToPythonIdentifier(utils.GetObjectRef(toolAgent)),
					Url:         targetURL,
					Headers:     headers,
					Description: toolAgent.Spec.Description,
				})
			default:
				return nil, nil, nil, fmt.Errorf("unknown agent type: %s", toolAgent.Spec.Type)
			}

		default:
			return nil, nil, nil, fmt.Errorf("tool must have a provider or tool server")
		}
	}

	if spec.Declarative.PromptTemplate != nil && len(spec.Declarative.PromptTemplate.DataSources) > 0 {
		lookup, err := resolvePromptSources(ctx, a.kube, agent.GetNamespace(), spec.Declarative.PromptTemplate.DataSources)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to resolve prompt sources: %w", err)
		}

		tplCtx := buildTemplateContext(agent, cfg)

		resolved, err := executeSystemMessageTemplate(cfg.Instruction, lookup, tplCtx)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to execute system message template: %w", err)
		}
		cfg.Instruction = resolved
	}

	return cfg, mdd, secretHashBytes, nil
}

// resolveRawSystemMessage gets the raw system message string from the agent spec
// without applying any template processing.
func (a *adkApiTranslator) resolveRawSystemMessage(ctx context.Context, agent v1alpha2.AgentObject) (string, error) {
	spec := agent.GetAgentSpec()
	if spec.Declarative.SystemMessageFrom != nil {
		return spec.Declarative.SystemMessageFrom.Resolve(ctx, a.kube, agent.GetNamespace())
	}
	if spec.Declarative.SystemMessage != "" {
		return spec.Declarative.SystemMessage, nil
	}
	return "", fmt.Errorf("at least one system message source (SystemMessage or SystemMessageFrom) must be specified")
}

// PlatformConfigMapName is the well-known ConfigMap name for platform-level defaults.
const PlatformConfigMapName = "kagent-platform-config"

// Platform ConfigMap keys for default ModelConfig references.
const (
	PlatformKeyAnthropicModelConfig = "default-anthropic-model-config"
	PlatformKeyOpenAIModelConfig    = "default-openai-model-config"
)

// injectAICoreProxySidecar checks if a CLI runtime (ClaudeCode/Codex) references a ModelConfig
// of provider SAPAICore. If so, it auto-injects a proxy sidecar container and sets the
// appropriate base URL environment variables so the CLI routes traffic through the proxy.
//
// When a CLI runtime does not explicitly set a ModelConfig, the controller falls back to
// the kagent-platform-config ConfigMap in the agent's namespace to find the default.
func (a *adkApiTranslator) injectAICoreProxySidecar(ctx context.Context, agent v1alpha2.AgentObject, dep *resolvedDeployment) error {
	spec := agent.GetAgentSpec()
	if spec.Declarative == nil {
		return nil
	}

	runtime := spec.Declarative.Runtime
	var modelConfigName string
	var provider string // "anthropic" or "openai"

	switch runtime {
	case v1alpha2.DeclarativeRuntime_ClaudeCode:
		provider = "anthropic"
		if spec.Declarative.ClaudeCodeConfig != nil && spec.Declarative.ClaudeCodeConfig.ModelConfig != "" {
			modelConfigName = spec.Declarative.ClaudeCodeConfig.ModelConfig
		}
	case v1alpha2.DeclarativeRuntime_Codex:
		provider = "openai"
		if spec.Declarative.CodexConfig != nil && spec.Declarative.CodexConfig.ModelConfig != "" {
			modelConfigName = spec.Declarative.CodexConfig.ModelConfig
		}
	default:
		return nil
	}

	// Fallback: read default ModelConfig from platform ConfigMap.
	if modelConfigName == "" {
		platformKey := PlatformKeyAnthropicModelConfig
		if provider == "openai" {
			platformKey = PlatformKeyOpenAIModelConfig
		}
		defaultMC, err := utils.GetConfigMapValue(ctx, a.kube, types.NamespacedName{
			Namespace: agent.GetNamespace(),
			Name:      PlatformConfigMapName,
		}, platformKey)
		if err != nil || defaultMC == "" {
			// No platform config or key not found — skip sidecar injection silently.
			return nil
		}
		modelConfigName = defaultMC
	}

	// Look up the ModelConfig resource
	mc := &v1alpha2.ModelConfig{}
	if err := a.kube.Get(ctx, types.NamespacedName{
		Namespace: agent.GetNamespace(),
		Name:      modelConfigName,
	}, mc); err != nil {
		return fmt.Errorf("failed to get ModelConfig %q for CLI runtime proxy: %w", modelConfigName, err)
	}

	// Only inject proxy for SAPAICore provider
	if mc.Spec.Provider != v1alpha2.ModelProviderSAPAICore {
		return nil
	}

	if mc.Spec.SAPAICore == nil {
		return fmt.Errorf("ModelConfig %q has provider SAPAICore but missing sapAICore config", modelConfigName)
	}

	// Build the proxy sidecar image reference using the same registry/tag as the main image
	proxyImage := getAICoreProxyImage()

	// Build sidecar container
	sidecar := corev1.Container{
		Name:            "aicore-proxy",
		Image:           proxyImage,
		ImagePullPolicy: corev1.PullAlways,
		Env: []corev1.EnvVar{
			{Name: "AICORE_PROXY_PROVIDER", Value: provider},
			{Name: "AICORE_PROXY_PORT", Value: "9090"},
			{Name: "SAP_AI_CORE_BASE_URL", Value: mc.Spec.SAPAICore.BaseURL},
			{Name: "SAP_AI_CORE_AUTH_URL", Value: mc.Spec.SAPAICore.AuthURL},
			{Name: "SAP_AI_CORE_RESOURCE_GROUP", Value: mc.Spec.SAPAICore.ResourceGroup},
			{Name: "SAP_AI_CORE_MODEL", Value: mc.Spec.Model},
		},
		Ports: []corev1.ContainerPort{{
			Name:          "proxy",
			ContainerPort: 9090,
		}},
	}

	// Inject credentials from the ModelConfig's apiKeySecret
	if mc.Spec.APIKeySecret != "" {
		sidecar.Env = append(sidecar.Env,
			corev1.EnvVar{
				Name: "SAP_AI_CORE_CLIENT_ID",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: mc.Spec.APIKeySecret},
						Key:                  "client_id",
					},
				},
			},
			corev1.EnvVar{
				Name: "SAP_AI_CORE_CLIENT_SECRET",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: mc.Spec.APIKeySecret},
						Key:                  "client_secret",
					},
				},
			},
		)
	}

	// Add sidecar to deployment
	dep.ExtraContainers = append(dep.ExtraContainers, sidecar)

	// Set base URL and placeholder API key on the main container
	switch provider {
	case "anthropic":
		dep.Env = append(dep.Env,
			corev1.EnvVar{Name: "ANTHROPIC_BASE_URL", Value: "http://localhost:9090"},
			corev1.EnvVar{Name: "ANTHROPIC_API_KEY", Value: "sk-placeholder"},
		)
	case "openai":
		dep.Env = append(dep.Env,
			corev1.EnvVar{Name: "OPENAI_BASE_URL", Value: "http://localhost:9090"},
			corev1.EnvVar{Name: "OPENAI_API_KEY", Value: "sk-placeholder"},
		)
	}

	return nil
}

// getAICoreProxyImage returns the fully qualified image reference for the aicore-proxy sidecar.
func getAICoreProxyImage() string {
	registry := DefaultImageConfig.Registry
	repo := DefaultImageConfig.Repository
	tag := DefaultImageConfig.Tag
	// Derive base path from repository (e.g., "kagent-dev/kagent/app" -> "kagent-dev/kagent")
	repoBase := repo
	if idx := lastIndexByte(repo, '/'); idx != -1 {
		repoBase = repo[:idx]
	}
	return fmt.Sprintf("%s/%s/aicore-proxy:%s", registry, repoBase, tag)
}

func lastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}
