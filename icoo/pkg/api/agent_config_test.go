package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func resolveAgentRoot(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", "..", "..", "apps", "agent"))
	if err != nil {
		t.Fatalf("resolve agent root: %v", err)
	}
	return root
}

func TestAgentConfigFileLoadsConfigTomlOnly(t *testing.T) {
	rt := NewRuntime()
	agentRoot := resolveAgentRoot(t)
	rt.SetProjectRoot(agentRoot, "@")

	workspace := t.TempDir()
	configPath := filepath.Join(workspace, "config.toml")
	configBody := `
model = "gpt-4.1"
base_url = "https://example.invalid/v1"
api_key = "test-key"
max_turns = 7
stream_final_answer = false
agent_name = "icoo_agent"
server_host = "0.0.0.0"
server_port = 9090
server_read_timeout_ms = 1000
server_read_header_timeout_ms = 2000
server_write_timeout_ms = 3000
server_idle_timeout_ms = 4000
session_dir = ".agent/sessions"
log_path = ".agent/agent.log"
`
	if err := os.WriteFile(configPath, []byte(configBody), 0o644); err != nil {
		t.Fatalf("write config.toml: %v", err)
	}

	source := `
import "@/pkg/config/src/file.ic" as config

fn main() {
  let cfg = config.applyFile({
    workspace: "` + filepath.ToSlash(workspace) + `"
  })

  if cfg.model != "gpt-4.1" {
    panic("expected config.toml model")
  }
  if cfg.baseUrl != "https://example.invalid/v1" {
    panic("expected config.toml base_url")
  }
  if cfg.apiKey != "test-key" {
    panic("expected config.toml api_key")
  }
  if cfg.maxTurns != 7 {
    panic("expected config.toml max_turns")
  }
  if cfg.streamFinalAnswer != false {
    panic("expected config.toml stream_final_answer")
  }
  if cfg.agentName != "icoo_agent" {
    panic("expected config.toml agent_name")
  }
  if cfg.serverHost != "0.0.0.0" {
    panic("expected config.toml server_host")
  }
  if cfg.serverPort != 9090 {
    panic("expected config.toml server_port")
  }
  if cfg.serverReadTimeoutMs != 1000 {
    panic("expected config.toml server_read_timeout_ms")
  }
  if cfg.serverReadHeaderTimeoutMs != 2000 {
    panic("expected config.toml server_read_header_timeout_ms")
  }
  if cfg.serverWriteTimeoutMs != 3000 {
    panic("expected config.toml server_write_timeout_ms")
  }
  if cfg.serverIdleTimeoutMs != 4000 {
    panic("expected config.toml server_idle_timeout_ms")
  }
  if cfg.sessionDir != "` + filepath.Join(workspace, ".agent", "sessions") + `" {
    panic("expected config.toml session_dir")
  }
  if cfg.logPath != "` + filepath.Join(workspace, ".agent", "agent.log") + `" {
    panic("expected config.toml log_path")
  }
}

main()
`

	if _, err := rt.RunSource(source); err != nil {
		t.Fatalf("expected config.toml to load, got: %v", err)
	}
}

func TestAgentConfigFileIgnoresNonTomlFiles(t *testing.T) {
	rt := NewRuntime()
	agentRoot := resolveAgentRoot(t)
	rt.SetProjectRoot(agentRoot, "@")

	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "config.json"), []byte(`{"model":"json-model"}`), 0o644); err != nil {
		t.Fatalf("write config.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "config.yaml"), []byte("model: yaml-model\n"), 0o644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}

	source := `
import "@/pkg/config/src/file.ic" as config

fn main() {
  let cfg = config.applyFile({
    workspace: "` + filepath.ToSlash(workspace) + `",
    model: "default-model"
  })

  if cfg.model != "default-model" {
    panic("expected non-toml config files to be ignored")
  }
}

main()
`

	if _, err := rt.RunSource(source); err != nil {
		t.Fatalf("expected non-toml config files to be ignored, got: %v", err)
	}
}

func TestAgentConfigFileLoadsWorkspaceNestedConfigToml(t *testing.T) {
	rt := NewRuntime()
	agentRoot := resolveAgentRoot(t)
	rt.SetProjectRoot(agentRoot, "@")

	baseDir := t.TempDir()
	workspaceDir := filepath.Join(baseDir, "runtime")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	rootConfig := `
workspace = "./runtime"
model = "root-model"
base_url = "https://example.invalid/root"
session_dir = "./runtime/.agents/sessions"
log_path = "./runtime/.agents/agent.log"
`
	if err := os.WriteFile(filepath.Join(baseDir, "config.toml"), []byte(rootConfig), 0o644); err != nil {
		t.Fatalf("write root config.toml: %v", err)
	}

	workspaceConfig := `
task = "今天成都的天气"
model = "workspace-model"
`
	if err := os.WriteFile(filepath.Join(workspaceDir, "config.toml"), []byte(workspaceConfig), 0o644); err != nil {
		t.Fatalf("write workspace config.toml: %v", err)
	}

	source := `
import "@/pkg/config/src/file.ic" as config

fn main() {
  let cfg = config.applyFile({
    workspace: "` + filepath.ToSlash(baseDir) + `"
  })

  if cfg.model != "workspace-model" {
    panic("expected workspace config.toml to override model")
  }
  if cfg.task != "今天成都的天气" {
    panic("expected workspace config.toml task")
  }
  if cfg.baseUrl != "https://example.invalid/root" {
    panic("expected root config.toml base_url")
  }
}

main()
`

	if _, err := rt.RunSource(source); err != nil {
		t.Fatalf("expected nested workspace config.toml to load, got: %v", err)
	}
}

func TestAgentMergeConfigIgnoresEnvAndArgs(t *testing.T) {
	rt := NewRuntime()
	agentRoot := resolveAgentRoot(t)
	rt.SetProjectRoot(agentRoot, "@")
	rt.SetScriptArgs([]string{
		"--workspace", t.TempDir(),
		"--model", "args-model",
		"--approval", "never",
	})

	oldModel := os.Getenv("ICOO_AGENTS_MODEL")
	oldApproval := os.Getenv("ICOO_AGENTS_APPROVAL")
	defer func() {
		_ = os.Setenv("ICOO_AGENTS_MODEL", oldModel)
		_ = os.Setenv("ICOO_AGENTS_APPROVAL", oldApproval)
	}()
	if err := os.Setenv("ICOO_AGENTS_MODEL", "env-model"); err != nil {
		t.Fatalf("set env model: %v", err)
	}
	if err := os.Setenv("ICOO_AGENTS_APPROVAL", "never"); err != nil {
		t.Fatalf("set env approval: %v", err)
	}

	source := `
import "@/pkg/config/src/main.ic" as config

fn main() {
  let merge = config.MergeConfig(
    config.Defaults(),
    config.ConfigFile()
  )
  let loaded = merge.loadConfig(null)
  let cfg = loaded.config

  if cfg.model != "gpt-4.1-mini" {
    panic("expected env and args model to be ignored")
  }
  if cfg.approval != "on-request" {
    panic("expected env and args approval to be ignored")
  }
  if loaded.help != false {
    panic("expected help to stay false")
  }
}

main()
`

	if _, err := rt.RunSource(source); err != nil {
		t.Fatalf("expected merge config to ignore env and args, got: %v", err)
	}
}

func TestAgentAppResolveCliLaunchRejectsCliArgs(t *testing.T) {
	rt := NewRuntime()
	agentRoot := resolveAgentRoot(t)
	rt.SetProjectRoot(agentRoot, "@")
	rt.SetScriptArgs([]string{"--tui"})

	source := `
import "@/src/app/app.ic" as appModule

fn main() {
  let app = appModule.App()
  app.resolveCliLaunch()
}

main()
`

	if _, err := rt.RunSource(source); err == nil {
		t.Fatal("expected CLI args to be rejected")
	} else if !strings.Contains(err.Error(), "CLI args are not supported") {
		t.Fatalf("expected CLI args rejection, got: %v", err)
	}
}
