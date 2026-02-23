package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManagerReloadAndAll(t *testing.T) {
	base := t.TempDir()

	writeFile(t, filepath.Join(base, "greet", "skill.toml"), `
name = "greet"
description = "says hello"
run = "echo hello"
triggers = ["^hi$"]
`)
	writeFile(t, filepath.Join(base, "calc", "skill.toml"), `
name = "calc"
description = "does math"
run = "bash run.sh"
triggers = ["^calc\\b"]
`)

	m := NewManager(base)
	if err := m.Reload(); err != nil {
		t.Fatal(err)
	}

	all := m.All()
	if len(all) != 2 {
		t.Fatalf("got %d skills, want 2", len(all))
	}
}

func TestManagerGet(t *testing.T) {
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "foo", "skill.toml"), `
name = "foo"
run = "echo ok"
`)

	m := NewManager(base)
	m.Reload()

	if s := m.Get("foo"); s == nil {
		t.Fatal("expected to find skill 'foo'")
	}
	if s := m.Get("bar"); s != nil {
		t.Fatal("expected nil for nonexistent skill")
	}
}

func TestManagerMatch(t *testing.T) {
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "todo", "skill.toml"), `
name = "todo"
run = "echo ok"
triggers = ["^todo\\b"]
`)
	writeFile(t, filepath.Join(base, "note", "skill.toml"), `
name = "note"
run = "echo ok"
triggers = ["^note\\b"]
`)

	m := NewManager(base)
	m.Reload()

	matched := m.Match("todo buy milk")
	if len(matched) != 1 || matched[0].Manifest.Name != "todo" {
		t.Errorf("expected todo to match, got %d matches", len(matched))
	}

	matched = m.Match("nothing special")
	if len(matched) != 0 {
		t.Errorf("expected no matches, got %d", len(matched))
	}
}

func TestManagerDescribe(t *testing.T) {
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "greet", "skill.toml"), `
name = "greet"
description = "greets the user"
run = "echo hi"
triggers = ["^hi$"]
`)

	m := NewManager(base)
	m.Reload()

	desc := m.Describe()
	if desc == "" {
		t.Fatal("expected non-empty description")
	}
	if !containsAll(desc, "greet", "greets the user", "^hi$") {
		t.Errorf("description missing expected content: %s", desc)
	}
}

func TestManagerDescribeEmpty(t *testing.T) {
	base := t.TempDir()
	m := NewManager(base)
	// no reload — empty dir
	if desc := m.Describe(); desc != "" {
		t.Errorf("expected empty description, got %q", desc)
	}
}

func TestManagerCreate(t *testing.T) {
	base := t.TempDir()
	m := NewManager(base)
	m.Reload()

	err := m.Create(CreateAction{
		Name:        "weather",
		Description: "checks weather",
		Run:         "python3 run.py",
		Triggers:    []string{"weather", "wetter"},
		Script:      "#!/usr/bin/env python3\nprint('sunny')\n",
	})
	if err != nil {
		t.Fatal(err)
	}

	// skill should be loaded
	s := m.Get("weather")
	if s == nil {
		t.Fatal("expected weather skill after create")
	}
	if s.Manifest.Description != "checks weather" {
		t.Errorf("description = %q", s.Manifest.Description)
	}

	// script should exist
	script, _ := os.ReadFile(filepath.Join(base, "weather", "run.py"))
	if string(script) != "#!/usr/bin/env python3\nprint('sunny')\n" {
		t.Errorf("script = %q", string(script))
	}
}

func TestManagerCreateDuplicate(t *testing.T) {
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "existing", "skill.toml"), `
name = "existing"
run = "echo ok"
`)

	m := NewManager(base)
	m.Reload()

	err := m.Create(CreateAction{Name: "existing", Run: "echo fail"})
	if err == nil {
		t.Fatal("expected error for duplicate skill name")
	}
}

func TestManagerEdit(t *testing.T) {
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "editable", "skill.toml"), `
name = "editable"
description = "old description"
run = "bash run.sh"
`)
	writeScript(t, filepath.Join(base, "editable", "run.sh"), "#!/bin/bash\necho old")

	m := NewManager(base)
	m.Reload()

	err := m.Edit(EditAction{
		Name:        "editable",
		Description: "new description",
		Script:      "#!/bin/bash\necho new",
	})
	if err != nil {
		t.Fatal(err)
	}

	s := m.Get("editable")
	if s.Manifest.Description != "new description" {
		t.Errorf("description = %q, want 'new description'", s.Manifest.Description)
	}

	script, _ := os.ReadFile(filepath.Join(base, "editable", "run.sh"))
	if string(script) != "#!/bin/bash\necho new" {
		t.Errorf("script = %q", string(script))
	}
}

func TestManagerEditNotFound(t *testing.T) {
	base := t.TempDir()
	m := NewManager(base)
	m.Reload()

	err := m.Edit(EditAction{Name: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for editing nonexistent skill")
	}
}

func TestManagerDelete(t *testing.T) {
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "deleteme", "skill.toml"), `
name = "deleteme"
run = "echo gone"
`)

	m := NewManager(base)
	m.Reload()

	if m.Get("deleteme") == nil {
		t.Fatal("skill should exist before delete")
	}

	if err := m.Delete("deleteme"); err != nil {
		t.Fatal(err)
	}

	if m.Get("deleteme") != nil {
		t.Fatal("skill should not exist after delete")
	}

	if _, err := os.Stat(filepath.Join(base, "deleteme")); !os.IsNotExist(err) {
		t.Fatal("skill directory should be removed")
	}
}

func TestManagerDeleteNotFound(t *testing.T) {
	base := t.TempDir()
	m := NewManager(base)
	m.Reload()

	if err := m.Delete("ghost"); err == nil {
		t.Fatal("expected error for deleting nonexistent skill")
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
