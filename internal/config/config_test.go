package config

import (
	"path/filepath"
	"testing"
)

func TestDefaultHomeFromEnv(t *testing.T) {
	t.Setenv("SKILLHUB_HOME", "/tmp/custom-home")
	home, err := DefaultHome()
	if err != nil {
		t.Fatalf("DefaultHome: %v", err)
	}
	want, _ := filepath.Abs("/tmp/custom-home")
	if home != want {
		t.Errorf("DefaultHome = %q, want %q", home, want)
	}
}

func TestNewDefaultHasHubRegistry(t *testing.T) {
	t.Setenv("SKILLHUB_HOME", t.TempDir())
	cfg, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault: %v", err)
	}
	hub, ok := cfg.Registries[DefaultHubName]
	if !ok {
		t.Fatalf("default config missing %q registry", DefaultHubName)
	}
	if hub.Type != DefaultHubType || hub.URL != DefaultHubURL {
		t.Errorf("hub registry = %+v, want type=%s url=%s", hub, DefaultHubType, DefaultHubURL)
	}
}

func TestLoadMissingReturnsDefault(t *testing.T) {
	t.Setenv("SKILLHUB_HOME", t.TempDir())
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, ok := cfg.Registries[DefaultHubName]; !ok {
		t.Error("Load of missing config did not fall back to default")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Setenv("SKILLHUB_HOME", t.TempDir())
	workDir := t.TempDir()
	in := Config{
		InstallDir: "/opt/skills",
		Registries: map[string]Registry{
			"company": {Type: "local", Path: "/srv/skills"},
		},
	}
	if err := Save(workDir, in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	out, err := Load(workDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.InstallDir != in.InstallDir {
		t.Errorf("InstallDir = %q, want %q", out.InstallDir, in.InstallDir)
	}
	company, ok := out.Registries["company"]
	if !ok || company.Type != "local" || company.Path != "/srv/skills" {
		t.Errorf("company registry round trip mismatch: %+v", out.Registries)
	}
}
