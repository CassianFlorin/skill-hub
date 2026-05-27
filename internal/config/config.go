package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const (
	FileName       = "skillhub.yaml"
	DefaultHubName = "hub"
	DefaultHubURL  = "https://github.com/CassianFlorin/skill-hub-registry.git"
	DefaultHubType = "git"
)

type Config struct {
	InstallDir string              `json:"install_dir"`
	Registries map[string]Registry `json:"registries"`
}

type Registry struct {
	Type string `json:"type"`
	Path string `json:"path,omitempty"`
	URL  string `json:"url,omitempty"`
}

func DefaultHome() (string, error) {
	if home := os.Getenv("SKILLHUB_HOME"); home != "" {
		return filepath.Abs(home)
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userHome, ".skillhub"), nil
}

func Path(workDir string) string {
	return filepath.Join(workDir, FileName)
}

func NewDefault() (Config, error) {
	home, err := DefaultHome()
	if err != nil {
		return Config{}, err
	}
	return Config{
		InstallDir: filepath.Join(home, "installed"),
		Registries: map[string]Registry{
			DefaultHubName: {Type: DefaultHubType, URL: DefaultHubURL},
		},
	}, nil
}

func Load(workDir string) (Config, error) {
	data, err := os.ReadFile(Path(workDir))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return NewDefault()
		}
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.Registries == nil {
		cfg.Registries = map[string]Registry{}
	}
	return cfg, nil
}

func Save(workDir string, cfg Config) error {
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(Path(workDir), data, 0o644)
}
