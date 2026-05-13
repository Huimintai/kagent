// Package cli implements an A2A executor that wraps CLI-based AI tools
// (Claude Code, Codex) as subprocesses. It translates A2A messages into
// CLI invocations and streams output back as A2A events.
package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	a2atype "github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/a2aproject/a2a-go/a2asrv/eventqueue"
	"github.com/go-logr/logr"
	"github.com/kagent-dev/kagent/go/adk/pkg/session"
)

const (
	// RuntimeClaudeCode identifies the Claude Code CLI runtime.
	RuntimeClaudeCode = "claude_code"
	// RuntimeCodex identifies the OpenAI Codex CLI runtime.
	RuntimeCodex = "codex"

	// Environment variables injected by the controller.
	envClaudeCodeModel    = "KAGENT_CLAUDE_CODE_MODEL"
	envClaudeCodeMaxTurns = "KAGENT_CLAUDE_CODE_MAX_TURNS"
	envCodexModel         = "KAGENT_CODEX_MODEL"
	envCodexSandbox       = "KAGENT_CODEX_SANDBOX"
)

// Config holds configuration for the CLI executor.
type Config struct {
	// Runtime is the CLI runtime type: RuntimeClaudeCode or RuntimeCodex.
	Runtime string

	// Instruction is the system message from the agent spec.
	Instruction string

	// SessionService for session persistence (optional).
	SessionService *session.KAgentSessionService

	// AppName is the application name for session/tracing.
	AppName string

	// Logger for structured logging.
	Logger logr.Logger
}

// Executor implements a2asrv.AgentExecutor by delegating to a CLI tool.
type Executor struct {
	runtime        string
	instruction    string
	sessionService *session.KAgentSessionService
	appName        string
	logger         logr.Logger

	// mu protects running processes for cancellation.
	mu        sync.Mutex
	processes map[string]*os.Process
}

var _ a2asrv.AgentExecutor = (*Executor)(nil)

// NewExecutor creates a CLI executor from the given config.
func NewExecutor(cfg Config) *Executor {
	return &Executor{
		runtime:        cfg.Runtime,
		instruction:    cfg.Instruction,
		sessionService: cfg.SessionService,
		appName:        cfg.AppName,
		logger:         cfg.Logger.WithName("cli-executor"),
		processes:      make(map[string]*os.Process),
	}
}

// DetectRuntime checks environment variables to determine if we're running
// as a CLI runtime. Returns RuntimeClaudeCode, RuntimeCodex, or "".
func DetectRuntime() string {
	if os.Getenv(envClaudeCodeModel) != "" {
		return RuntimeClaudeCode
	}
	if os.Getenv(envCodexModel) != "" {
		return RuntimeCodex
	}
	return ""
}

// Execute implements a2asrv.AgentExecutor. It invokes the CLI tool with the
// user's message and streams output back via A2A events.
func (e *Executor) Execute(ctx context.Context, reqCtx *a2asrv.RequestContext, queue eventqueue.Queue) error {
	if reqCtx.Message == nil {
		return fmt.Errorf("A2A request message cannot be nil")
	}

	taskID := string(reqCtx.TaskID)
	userMessage := extractTextFromMessage(reqCtx.Message)

	e.logger.Info("Execute",
		"taskID", taskID,
		"runtime", e.runtime,
		"messageLength", len(userMessage),
	)

	// Emit submitted status.
	submitted := a2atype.NewStatusUpdateEvent(reqCtx, a2atype.TaskStateSubmitted, reqCtx.Message)
	if err := queue.Write(ctx, submitted); err != nil {
		return fmt.Errorf("failed to write submitted event: %w", err)
	}

	// Emit working status.
	working := a2atype.NewStatusUpdateEvent(reqCtx, a2atype.TaskStateWorking, nil)
	if err := queue.Write(ctx, working); err != nil {
		return fmt.Errorf("failed to write working event: %w", err)
	}

	// Build and run the CLI command.
	cmd := e.buildCommand(ctx, userMessage)
	e.logger.Info("Running CLI command", "path", cmd.Path, "args", cmd.Args)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return e.emitError(ctx, reqCtx, queue, fmt.Errorf("failed to create stdout pipe: %w", err))
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return e.emitError(ctx, reqCtx, queue, fmt.Errorf("failed to create stderr pipe: %w", err))
	}

	if err := cmd.Start(); err != nil {
		return e.emitError(ctx, reqCtx, queue, fmt.Errorf("failed to start CLI: %w", err))
	}

	// Track process for cancellation.
	e.mu.Lock()
	e.processes[taskID] = cmd.Process
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		delete(e.processes, taskID)
		e.mu.Unlock()
	}()

	// Read stdout and stream as partial events.
	var outputBuilder strings.Builder
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 1MB max line

	for scanner.Scan() {
		line := scanner.Text()
		outputBuilder.WriteString(line)
		outputBuilder.WriteString("\n")

		// Emit partial working event.
		partialMsg := a2atype.NewMessage(a2atype.MessageRoleAgent,
			a2atype.TextPart{Text: line + "\n"})
		partial := a2atype.NewStatusUpdateEvent(reqCtx, a2atype.TaskStateWorking, partialMsg)
		if err := queue.Write(ctx, partial); err != nil {
			e.logger.V(1).Info("Failed to write partial event", "error", err)
		}
	}

	// Capture stderr for error reporting.
	var stderrBuilder strings.Builder
	stderrScanner := bufio.NewScanner(stderr)
	for stderrScanner.Scan() {
		stderrBuilder.WriteString(stderrScanner.Text())
		stderrBuilder.WriteString("\n")
	}

	// Wait for process completion.
	if err := cmd.Wait(); err != nil {
		errOutput := stderrBuilder.String()
		if errOutput == "" {
			errOutput = err.Error()
		}
		return e.emitError(ctx, reqCtx, queue, fmt.Errorf("CLI exited with error: %s", errOutput))
	}

	// Emit final artifact with complete output.
	finalText := outputBuilder.String()
	if finalText == "" {
		finalText = "(no output)"
	}

	artifactEvent := a2atype.NewArtifactEvent(reqCtx, a2atype.TextPart{Text: finalText})
	artifactEvent.LastChunk = true
	if err := queue.Write(ctx, artifactEvent); err != nil {
		return fmt.Errorf("failed to write artifact event: %w", err)
	}

	// Emit completed.
	completed := a2atype.NewStatusUpdateEvent(reqCtx, a2atype.TaskStateCompleted, nil)
	completed.Final = true
	return queue.Write(ctx, completed)
}

// Cancel implements a2asrv.AgentExecutor by killing the running CLI process.
func (e *Executor) Cancel(ctx context.Context, reqCtx *a2asrv.RequestContext, queue eventqueue.Queue) error {
	taskID := string(reqCtx.TaskID)

	e.mu.Lock()
	proc, ok := e.processes[taskID]
	e.mu.Unlock()

	if ok && proc != nil {
		e.logger.Info("Cancelling CLI process", "taskID", taskID)
		_ = proc.Kill()
	}

	canceled := a2atype.NewStatusUpdateEvent(reqCtx, a2atype.TaskStateCanceled, nil)
	canceled.Final = true
	return queue.Write(ctx, canceled)
}

// buildCommand constructs the CLI command based on runtime type.
func (e *Executor) buildCommand(ctx context.Context, userMessage string) *exec.Cmd {
	switch e.runtime {
	case RuntimeClaudeCode:
		return e.buildClaudeCodeCommand(ctx, userMessage)
	case RuntimeCodex:
		return e.buildCodexCommand(ctx, userMessage)
	default:
		// Should never happen — fall back to echo.
		return exec.CommandContext(ctx, "echo", "Unknown runtime: "+e.runtime)
	}
}

// buildClaudeCodeCommand creates the claude CLI command.
func (e *Executor) buildClaudeCodeCommand(ctx context.Context, userMessage string) *exec.Cmd {
	model := os.Getenv(envClaudeCodeModel)

	args := []string{
		"--print",             // Non-interactive single-shot mode
		"--output-format", "text", // Plain text output
	}

	if model != "" {
		args = append(args, "--model", model)
	}

	if maxTurns := os.Getenv(envClaudeCodeMaxTurns); maxTurns != "" {
		if _, err := strconv.Atoi(maxTurns); err == nil {
			args = append(args, "--max-turns", maxTurns)
		}
	}

	// System prompt via --system-prompt flag.
	if e.instruction != "" {
		args = append(args, "--system-prompt", e.instruction)
	}

	// User message as positional argument (last).
	args = append(args, userMessage)

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Env = os.Environ()
	return cmd
}

// buildCodexCommand creates the codex CLI command.
// Codex CLI uses `codex exec` subcommand for non-interactive execution.
func (e *Executor) buildCodexCommand(ctx context.Context, userMessage string) *exec.Cmd {
	model := os.Getenv(envCodexModel)
	baseURL := os.Getenv("OPENAI_BASE_URL")

	args := []string{
		"exec", // Non-interactive subcommand
		"--skip-git-repo-check",
	}

	if model != "" {
		args = append(args, "-c", fmt.Sprintf("model=%q", model))
	}

	// Explicit base URL via config to ensure proxy routing.
	if baseURL != "" {
		args = append(args, "-c", fmt.Sprintf("openai_base_url=%q", baseURL))
	}

	// Disable web_search tool — AI Core models do not support it.
	args = append(args, "-c", `web_search="disabled"`)

	// System instructions via config override (no --instructions flag in exec).
	if e.instruction != "" {
		args = append(args, "-c", fmt.Sprintf("instructions=%q", e.instruction))
	}

	// User message as positional argument.
	args = append(args, userMessage)

	cmd := exec.CommandContext(ctx, "codex", args...)
	cmd.Env = os.Environ()
	cmd.Stdin = nil // Prevent reading from stdin
	return cmd
}

// emitError writes a TaskStateFailed event.
func (e *Executor) emitError(ctx context.Context, reqCtx *a2asrv.RequestContext, queue eventqueue.Queue, err error) error {
	e.logger.Error(err, "CLI execution failed")

	errMsg := a2atype.NewMessage(a2atype.MessageRoleAgent,
		a2atype.TextPart{Text: fmt.Sprintf("Error: %s", err.Error())})
	failed := a2atype.NewStatusUpdateEvent(reqCtx, a2atype.TaskStateFailed, errMsg)
	failed.Final = true
	return queue.Write(ctx, failed)
}

// extractTextFromMessage extracts plain text from an A2A message.
func extractTextFromMessage(msg *a2atype.Message) string {
	if msg == nil {
		return ""
	}
	var parts []string
	for _, part := range msg.Parts {
		switch p := part.(type) {
		case a2atype.TextPart:
			parts = append(parts, p.Text)
		case *a2atype.TextPart:
			parts = append(parts, p.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// CLIRuntimeInfo returns runtime information for the agent card.
func CLIRuntimeInfo(runtime string) (name, description string) {
	switch runtime {
	case RuntimeClaudeCode:
		model := os.Getenv(envClaudeCodeModel)
		return "claude-code-agent", fmt.Sprintf("Claude Code agent (model: %s)", model)
	case RuntimeCodex:
		model := os.Getenv(envCodexModel)
		return "codex-agent", fmt.Sprintf("Codex agent (model: %s)", model)
	default:
		return "cli-agent", "CLI-based agent"
	}
}

// SkillsFromEnv parses the KAGENT_CLAUDE_CODE_SKILLS env var (JSON array).
func SkillsFromEnv() []string {
	raw := os.Getenv("KAGENT_CLAUDE_CODE_SKILLS")
	if raw == "" {
		return nil
	}
	var skills []string
	if err := json.Unmarshal([]byte(raw), &skills); err != nil {
		return nil
	}
	return skills
}
