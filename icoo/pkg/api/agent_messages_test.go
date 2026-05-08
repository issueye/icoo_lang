package api

import (
	"testing"
)

func TestAgentModelMessagesPutSkillsIntoSystemPrompt(t *testing.T) {
	rt := NewRuntime()
	agentRoot := resolveAgentRoot(t)
	rt.SetProjectRoot(agentRoot, "@")

	source := `
import "@/src/model/messages.ic" as messages
import std.core.string as str

fn main() {
  let modelMessages = messages.ModelMessages({
    definitions: fn() {
      return []
    }
  })

  let built = modelMessages.buildBase({
    task: "test"
  }, {
    workspace: "E:/workspace",
    summaryJson: "{\"fileCount\":1}",
    skillsSummary: "Available Skills:\n- code: coding workflow",
    cliUnknownArgs: []
  })

  let system = built[0].content
  let user = built[1].content

  if system == null || system == "" {
    panic("expected system prompt")
  }
  if user == null || user == "" {
    panic("expected user prompt")
  }
  if str.indexOf(system, "Available Skills:") < 0 {
    panic("expected skills in system prompt")
  }
  if str.indexOf(user, "Available Skills:") >= 0 {
    panic("expected skills to be absent from user prompt")
  }
}

main()
`

	if _, err := rt.RunSource(source); err != nil {
		t.Fatalf("expected system prompt to contain skills summary, got: %v", err)
	}
}

func TestAgentModelMessagesSystemPromptStillContainsBaseInstruction(t *testing.T) {
	rt := NewRuntime()
	agentRoot := resolveAgentRoot(t)
	rt.SetProjectRoot(agentRoot, "@")

	source := `
import "@/src/model/messages.ic" as messages
import std.core.string as str

fn main() {
  let modelMessages = messages.ModelMessages({
    definitions: fn() {
      return []
    }
  })

  let text = modelMessages.systemPrompt({
    skillsSummary: "Available Skills:\n- code: coding workflow"
  })

  if str.indexOf(text, "You are an Icoo agent.") < 0 {
    panic("expected base system instruction")
  }
}

main()
`

	if _, err := rt.RunSource(source); err != nil {
		t.Fatalf("expected base instruction to remain in system prompt, got: %v", err)
	}
}
