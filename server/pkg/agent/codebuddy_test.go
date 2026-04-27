package agent

import (
	"log/slog"
	"strings"
	"testing"
)

func TestCodebuddyHandleAssistantText(t *testing.T) {
	t.Parallel()

	ch := make(chan Message, 10)
	var output strings.Builder

	msg := claudeSDKMessage{
		Type: "assistant",
		Message: mustMarshal(t, claudeMessageContent{
			Role: "assistant",
			Content: []claudeContentBlock{
				{Type: "text", Text: "Hello from codebuddy"},
			},
		}),
	}

	handleCodebuddyAssistant(msg, ch, &output, make(map[string]TokenUsage))

	if output.String() != "Hello from codebuddy" {
		t.Fatalf("expected output 'Hello from codebuddy', got %q", output.String())
	}
	select {
	case m := <-ch:
		if m.Type != MessageText || m.Content != "Hello from codebuddy" {
			t.Fatalf("unexpected message: %+v", m)
		}
	default:
		t.Fatal("expected message on channel")
	}
}

func TestCodebuddyHandleAssistantToolUse(t *testing.T) {
	t.Parallel()

	ch := make(chan Message, 10)
	var output strings.Builder

	msg := claudeSDKMessage{
		Type: "assistant",
		Message: mustMarshal(t, claudeMessageContent{
			Role: "assistant",
			Content: []claudeContentBlock{
				{
					Type:  "tool_use",
					ID:    "call-1",
					Name:  "Read",
					Input: mustMarshal(t, map[string]any{"path": "/tmp/foo"}),
				},
			},
		}),
	}

	handleCodebuddyAssistant(msg, ch, &output, make(map[string]TokenUsage))

	if output.String() != "" {
		t.Fatalf("tool_use should not add to output, got %q", output.String())
	}
	select {
	case m := <-ch:
		if m.Type != MessageToolUse || m.Tool != "Read" || m.CallID != "call-1" {
			t.Fatalf("unexpected message: %+v", m)
		}
		if m.Input["path"] != "/tmp/foo" {
			t.Fatalf("expected input path /tmp/foo, got %v", m.Input["path"])
		}
	default:
		t.Fatal("expected message on channel")
	}
}

func TestCodebuddyHandleAssistantThinking(t *testing.T) {
	t.Parallel()

	ch := make(chan Message, 10)
	var output strings.Builder

	msg := claudeSDKMessage{
		Type: "assistant",
		Message: mustMarshal(t, claudeMessageContent{
			Role: "assistant",
			Content: []claudeContentBlock{
				{Type: "thinking", Text: "Let me think..."},
			},
		}),
	}

	handleCodebuddyAssistant(msg, ch, &output, make(map[string]TokenUsage))

	if output.String() != "" {
		t.Fatalf("thinking should not add to output, got %q", output.String())
	}
	select {
	case m := <-ch:
		if m.Type != MessageThinking || m.Content != "Let me think..." {
			t.Fatalf("unexpected message: %+v", m)
		}
	default:
		t.Fatal("expected message on channel")
	}
}

func TestCodebuddyHandleUser(t *testing.T) {
	t.Parallel()

	ch := make(chan Message, 10)

	msg := claudeSDKMessage{
		Type: "user",
		Message: mustMarshal(t, claudeMessageContent{
			Role: "user",
			Content: []claudeContentBlock{
				{
					Type:      "tool_result",
					ToolUseID: "call-1",
					Content:   mustMarshal(t, "file contents here"),
				},
			},
		}),
	}

	handleCodebuddyUser(msg, ch)

	select {
	case m := <-ch:
		if m.Type != MessageToolResult || m.CallID != "call-1" {
			t.Fatalf("unexpected message: %+v", m)
		}
	default:
		t.Fatal("expected message on channel")
	}
}

func TestBuildCodebuddyArgs(t *testing.T) {
	t.Parallel()
	logger := slog.Default()

	containsArg := func(args []string, want string) bool {
		for _, a := range args {
			if a == want {
				return true
			}
		}
		return false
	}

	t.Run("basic flags present", func(t *testing.T) {
		args := buildCodebuddyArgs(ExecOptions{}, logger)
		for _, want := range []string{"-p", "--output-format", "stream-json", "--permission-mode", "bypassPermissions"} {
			if !containsArg(args, want) {
				t.Errorf("expected arg %q in %v", want, args)
			}
		}
	})

	t.Run("with model", func(t *testing.T) {
		args := buildCodebuddyArgs(ExecOptions{Model: "claude-sonnet-4.6"}, logger)
		if !containsArg(args, "--model") || !containsArg(args, "claude-sonnet-4.6") {
			t.Fatalf("expected --model claude-sonnet-4.6 in %v", args)
		}
	})

	t.Run("with max turns", func(t *testing.T) {
		args := buildCodebuddyArgs(ExecOptions{MaxTurns: 5}, logger)
		if !containsArg(args, "--max-turns") || !containsArg(args, "5") {
			t.Fatalf("expected --max-turns 5 in %v", args)
		}
	})

	t.Run("with resume session", func(t *testing.T) {
		args := buildCodebuddyArgs(ExecOptions{ResumeSessionID: "sess-123"}, logger)
		if !containsArg(args, "--resume") || !containsArg(args, "sess-123") {
			t.Fatalf("expected --resume sess-123 in %v", args)
		}
	})

	t.Run("blocked args filtered", func(t *testing.T) {
		args := buildCodebuddyArgs(ExecOptions{
			CustomArgs: []string{"--output-format", "text", "--model", "gpt-5"},
		}, logger)
		for i, a := range args {
			if a == "--output-format" && i+1 < len(args) && args[i+1] == "text" {
				t.Fatal("blocked arg --output-format text should have been filtered")
			}
		}
		if !containsArg(args, "gpt-5") {
			t.Fatalf("expected --model gpt-5 to pass through in %v", args)
		}
	})
}

func TestCodebuddyAgentRegistration(t *testing.T) {
	t.Parallel()

	_, err := New("codebuddy", Config{Logger: slog.Default(), ExecutablePath: "codebuddy"})
	if err != nil {
		t.Fatalf("New(codebuddy) returned unexpected error: %v", err)
	}
}

func TestCodebuddyLaunchHeader(t *testing.T) {
	t.Parallel()

	h := LaunchHeader("codebuddy")
	if h == "" {
		t.Fatal("expected non-empty launch header for codebuddy")
	}
	if !strings.Contains(h, "codebuddy") {
		t.Fatalf("launch header should contain 'codebuddy', got %q", h)
	}
}
