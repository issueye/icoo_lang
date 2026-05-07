package api

import (
	"path/filepath"
	"testing"
)

func TestAgentConfigFileLoadsProjectDefaults(t *testing.T) {
	rt := NewRuntime()
	agentRoot, err := filepath.Abs(filepath.Join("..", "..", "apps", "agent"))
	if err != nil {
		t.Fatalf("resolve agent root: %v", err)
	}
	rt.SetProjectRoot(agentRoot, "@")

	source := `
import "@/src/config/file.ic" as config

fn main() {
  let cfg = config.applyFile({
    workspace: "` + filepath.ToSlash(agentRoot) + `"
  })

  if cfg.model != "gpt-4.1-mini" {
    panic("expected config.toml model default")
  }
  if cfg.baseUrl != "https://api.openai.com/v1" {
    panic("expected config.toml base_url default")
  }
	if cfg.maxTurns != 12 {
		panic("expected config.toml max_turns default")
	}
  if cfg.sessionDir != "` + filepath.Join(agentRoot, ".agent", "sessions") + `" {
    panic("expected config.toml session_dir to resolve from config file directory")
  }
  if cfg.logPath != "` + filepath.Join(agentRoot, ".agent", "agent.log") + `" {
    panic("expected config.toml log_path to resolve from config file directory")
  }
}

main()
`

	if _, err := rt.RunSource(source); err != nil {
		t.Fatalf("expected config.toml defaults to load, got: %v", err)
	}
}
