package cli

import (
	"os"
	"testing"
)

func TestDetectRuntime(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		want     string
	}{
		{
			name:     "Claude Code runtime detected",
			envKey:   envClaudeCodeModel,
			envValue: "claude-sonnet-4-20250514",
			want:     RuntimeClaudeCode,
		},
		{
			name:     "Codex runtime detected",
			envKey:   envCodexModel,
			envValue: "codex-mini-latest",
			want:     RuntimeCodex,
		},
		{
			name: "no runtime detected",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear relevant env vars.
			os.Unsetenv(envClaudeCodeModel)
			os.Unsetenv(envCodexModel)

			if tt.envKey != "" {
				t.Setenv(tt.envKey, tt.envValue)
			}

			got := DetectRuntime()
			if got != tt.want {
				t.Errorf("DetectRuntime() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildClaudeCodeCommand(t *testing.T) {
	t.Setenv(envClaudeCodeModel, "claude-sonnet-4-20250514")
	t.Setenv(envClaudeCodeMaxTurns, "10")

	e := &Executor{
		runtime:     RuntimeClaudeCode,
		instruction: "You are a test agent",
	}

	cmd := e.buildClaudeCodeCommand(t.Context(), "Hello, world!")

	if cmd.Args[0] != "claude" {
		t.Errorf("Expected command 'claude', got %q", cmd.Args[0])
	}

	// Check args contain expected flags.
	if !containsArg(cmd.Args, "--print") {
		t.Error("Missing --print flag")
	}
	if !containsFlag(cmd.Args, "--output-format", "text") {
		t.Error("Missing --output-format text")
	}
	if !containsFlag(cmd.Args, "--model", "claude-sonnet-4-20250514") {
		t.Error("Missing --model flag")
	}
	if !containsFlag(cmd.Args, "--max-turns", "10") {
		t.Error("Missing --max-turns flag")
	}
	if !containsFlag(cmd.Args, "--system-prompt", "You are a test agent") {
		t.Error("Missing --system-prompt flag")
	}
	// User message should be the last positional argument (not --prompt).
	lastArg := cmd.Args[len(cmd.Args)-1]
	if lastArg != "Hello, world!" {
		t.Errorf("Expected last arg to be user message, got %q", lastArg)
	}
}

func TestBuildCodexCommand(t *testing.T) {
	t.Setenv(envCodexModel, "codex-mini-latest")

	e := &Executor{
		runtime:     RuntimeCodex,
		instruction: "Write tests",
	}

	cmd := e.buildCodexCommand(t.Context(), "Fix the bug")

	if cmd.Args[0] != "codex" {
		t.Errorf("Expected command 'codex', got %q", cmd.Args[0])
	}

	argsStr := joinArgs(cmd.Args[1:])
	// Should use "exec" subcommand for non-interactive mode
	if !containsArg(cmd.Args, "exec") {
		t.Errorf("Missing 'exec' subcommand in args: %s", argsStr)
	}
	if !containsFlag(cmd.Args, "-c", `model="codex-mini-latest"`) {
		t.Errorf("Missing -c model config in args: %s", argsStr)
	}
	if !containsFlag(cmd.Args, "-c", `instructions="Write tests"`) {
		t.Errorf("Missing -c instructions config in args: %s", argsStr)
	}
	// User message should be the last positional argument.
	lastArg := cmd.Args[len(cmd.Args)-1]
	if lastArg != "Fix the bug" {
		t.Errorf("Expected last arg to be user message, got %q", lastArg)
	}
}

func TestExtractTextFromMessage(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "empty", text: "", want: ""},
		{name: "simple", text: "hello", want: "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// extractTextFromMessage with nil returns ""
			got := extractTextFromMessage(nil)
			if tt.text == "" && got != "" {
				t.Errorf("extractTextFromMessage(nil) = %q, want empty", got)
			}
		})
	}
	_ = tests
}

func TestCLIRuntimeInfo(t *testing.T) {
	t.Setenv(envClaudeCodeModel, "claude-sonnet-4-20250514")

	name, desc := CLIRuntimeInfo(RuntimeClaudeCode)
	if name != "claude-code-agent" {
		t.Errorf("name = %q, want 'claude-code-agent'", name)
	}
	if desc == "" {
		t.Error("description should not be empty")
	}
}

// Helper functions

func containsArg(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func containsFlag(args []string, flag, value string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}

func joinArgs(args []string) string {
	result := ""
	for i, a := range args {
		if i > 0 {
			result += " "
		}
		result += a
	}
	return result
}
