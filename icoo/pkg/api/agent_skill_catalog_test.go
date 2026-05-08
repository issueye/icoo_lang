package api

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAgentSkillCatalogScansWorkspaceLocalSkillsOnly(t *testing.T) {
	rt := NewRuntime()
	agentRoot := resolveAgentRoot(t)
	rt.SetProjectRoot(agentRoot, "@")

	workspace := t.TempDir()
	workspaceSkills := filepath.Join(workspace, ".agents", "skills", "weather-local")
	if err := os.MkdirAll(workspaceSkills, 0o755); err != nil {
		t.Fatalf("mkdir workspace skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspaceSkills, "SKILL.md"), []byte("---\nname: weather\ndescription: local weather\n---\n"), 0o644); err != nil {
		t.Fatalf("write workspace skill: %v", err)
	}

	globalRoot := t.TempDir()
	globalSkills := filepath.Join(globalRoot, ".agents", "skills", "global-skill")
	if err := os.MkdirAll(globalSkills, 0o755); err != nil {
		t.Fatalf("mkdir global skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(globalSkills, "SKILL.md"), []byte("---\nname: global\ndescription: should not load\n---\n"), 0o644); err != nil {
		t.Fatalf("write global skill: %v", err)
	}

	oldProfile := os.Getenv("USERPROFILE")
	defer func() {
		_ = os.Setenv("USERPROFILE", oldProfile)
	}()
	if err := os.Setenv("USERPROFILE", globalRoot); err != nil {
		t.Fatalf("set USERPROFILE: %v", err)
	}

	source := `
import "@/src/skills/catalog.ic" as skills
import std.core.string as str

fn main() {
  let catalog = skills.SkillCatalog()
  let items = catalog.list({
    workspace: "` + filepath.ToSlash(workspace) + `",
    skillsDir: "E:/should/not/use"
  })
  let summary = catalog.summary({
    workspace: "` + filepath.ToSlash(workspace) + `",
    skillsDir: "E:/should/not/use"
  })
  let dirs = catalog.configuredDirs({
    workspace: "` + filepath.ToSlash(workspace) + `",
    skillsDir: "E:/should/not/use"
  })

  if len(dirs) != 1 {
    panic("expected exactly one configured skills dir")
  }
  if str.indexOf(dirs[0], ".agents") < 0 || str.indexOf(dirs[0], "skills") < 0 {
    panic("expected workspace-local .agents/skills path")
  }
  if len(items) != 1 {
    panic("expected exactly one local skill")
  }
  if items[0].id != "weather-local" {
    panic("expected local skill id")
  }
  if str.indexOf(summary, "global-skill") >= 0 || str.indexOf(summary, "should not load") >= 0 {
    panic("expected global skills to be ignored")
  }
}

main()
`

	if _, err := rt.RunSource(source); err != nil {
		t.Fatalf("expected workspace-local skills only, got: %v", err)
	}
}
