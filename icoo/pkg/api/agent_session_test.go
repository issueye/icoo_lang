package api

import (
	"path/filepath"
	"strings"
	"testing"
)

func setAgentProjectRoot(t *testing.T, rt *Runtime) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "apps", "agent"))
	if err != nil {
		t.Fatalf("resolve agent project root: %v", err)
	}
	rt.SetProjectRoot(root, "@")
	return root
}

func TestAgentSessionBuildMessagesReplaysReasoningContent(t *testing.T) {
	rt := NewRuntime()
	setAgentProjectRoot(t, rt)

source := `
import "@/pkg/session/lib.ic" as sessionPkg

fn main() {
  let session = {
    turns: [
      {
        kind: "user_message",
        payload: {
          role: "user",
          content: "inspect repo"
        }
      },
      {
        kind: "assistant_message",
        payload: {
          content: "",
          reasoningContent: "need tools first",
          toolCalls: [
            {
              id: "call_1",
              type: "function",
              function: {
                name: "listDir",
                arguments: "{\"path\":\"@/\"}"
              }
            }
          ],
          hasToolCalls: true
        }
      },
      {
        kind: "tool_message",
        payload: {
          toolCallId: "call_1",
          name: "listDir",
          output: {
            ok: true
          }
        }
      }
    ]
  }

  let messages = sessionPkg.SessionStore(sessionPkg.Schema()).buildMessages({}, {userPrompt: "fallback"}, session)
  if len(messages) != 4 {
    panic("unexpected message count")
  }

  let assistant = messages[2]
  if assistant.role != "assistant" {
    panic("assistant role lost")
  }
  if assistant.reasoning_content != "need tools first" {
    panic("reasoning_content lost during replay")
  }
  if len(assistant.tool_calls) != 1 {
    panic("tool_calls lost during replay")
  }

  let tool = messages[3]
  if tool.role != "tool" {
    panic("tool role lost")
  }
  if tool.tool_call_id != "call_1" {
    panic("tool_call_id lost during replay")
  }
}

main()
`

	if _, err := rt.RunSource(source); err != nil {
		t.Fatalf("expected session replay script to succeed, got: %v", err)
	}
}

func TestAgentSessionLoadRejectsDeprecatedTurnKinds(t *testing.T) {
	rt := NewRuntime()
	root := setAgentProjectRoot(t, rt)
	sessionDir := t.TempDir()
	sessionPath := filepath.Join(sessionDir, "legacy.json")

source := `
import std.io.fs as fs
import std.data.json as json
import "@/pkg/session/lib.ic" as sessionPkg

fn main() {
  let filePath = "` + filepath.ToSlash(sessionPath) + `"
  json.saveToFile(filePath, {
    sessionSchemaVersion: 2,
    sessionId: "legacy",
    workspace: "@/workspace",
    mode: "interactive",
    status: "completed",
    createdAt: 0,
    updatedAt: 0,
    config: {},
    turns: [
      {
        kind: "assistant_response",
        payload: {
          text: "legacy"
        }
      }
    ],
    events: [],
    artifacts: [],
    budgets: {
      turnCount: 1,
      toolCallCount: 0
    },
    approvals: []
  })

  sessionPkg.SessionStore(sessionPkg.Schema()).load({
    sessionDir: "` + filepath.ToSlash(sessionDir) + `"
  }, "legacy")
}

main()
`

	rt.SetProjectRoot(root, "@")
	_, err := rt.RunSource(source)
	if err == nil {
		t.Fatal("expected deprecated session turn kind to be rejected")
	}
	if !strings.Contains(err.Error(), "deprecated session turn kind: assistant_response") {
		t.Fatalf("expected deprecated turn kind error, got: %v", err)
	}
}

func TestAgentSessionLoadAcceptsCurrentTurnKinds(t *testing.T) {
	rt := NewRuntime()
	root := setAgentProjectRoot(t, rt)
	sessionDir := t.TempDir()
	sessionPath := filepath.Join(sessionDir, "current.json")

source := `
import std.data.json as json
import "@/pkg/session/lib.ic" as sessionPkg

fn main() {
  let filePath = "` + filepath.ToSlash(sessionPath) + `"
  json.saveToFile(filePath, {
    sessionSchemaVersion: 2,
    sessionId: "current",
    workspace: "@/workspace",
    mode: "interactive",
    status: "completed",
    createdAt: 0,
    updatedAt: 0,
    config: {},
    turns: [
      {
        kind: "user_message",
        payload: {
          role: "user",
          content: "hello"
        }
      },
      {
        kind: "assistant_message",
        payload: {
          content: "world",
          reasoningContent: "",
          toolCalls: [],
          hasToolCalls: false
        }
      }
    ],
    events: [],
    artifacts: [],
    budgets: {
      turnCount: 2,
      toolCallCount: 0
    },
    approvals: []
  })

  let loaded = sessionPkg.SessionStore(sessionPkg.Schema()).load({
    sessionDir: "` + filepath.ToSlash(sessionDir) + `"
  }, "current")
  if loaded == null {
    panic("expected current session file to load")
  }
}

main()
`

	rt.SetProjectRoot(root, "@")
	if _, err := rt.RunSource(source); err != nil {
		t.Fatalf("expected current session turn kinds to load, got: %v", err)
	}
}
