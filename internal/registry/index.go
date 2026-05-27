package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cassian/skill-hub/internal/config"
	"github.com/cassian/skill-hub/internal/skill"
)

const IndexFileName = "skillhub.index.json"

type Index struct {
	Registry    string       `json:"registry"`
	GeneratedAt string       `json:"generated_at"`
	Skills      []IndexSkill `json:"skills"`
}

type IndexSkill struct {
	Identity    string   `json:"identity"`
	Name        string   `json:"name"`
	Namespace   string   `json:"namespace"`
	Version     string   `json:"version"`
	Description string   `json:"description,omitempty"`
	Targets     []string `json:"targets,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	SourceType  string   `json:"source_type"`
	SourcePath  string   `json:"source_path"`
	Checksum    string   `json:"checksum,omitempty"`
}

type SearchResult struct {
	Registry string
	Skill    IndexSkill
}

func SearchIndexes(cfg config.Config, query string) ([]SearchResult, error) {
	query = strings.ToLower(query)
	var results []SearchResult
	for name, reg := range cfg.Registries {
		root, _, err := registryRoot(name, reg)
		if err != nil {
			return nil, err
		}
		index, err := LoadIndex(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, indexed := range index.Skills {
			if matches(indexed, query) {
				results = append(results, SearchResult{Registry: name, Skill: indexed})
			}
		}
	}
	return results, nil
}

func FindIndexedSkill(cfg config.Config, spec string) (SearchResult, bool, error) {
	registryName, identity, ok := strings.Cut(spec, "/")
	if ok {
		if reg, exists := cfg.Registries[registryName]; exists {
			root, _, err := registryRoot(registryName, reg)
			if err != nil {
				return SearchResult{}, false, err
			}
			index, err := LoadIndex(root)
			if err != nil {
				return SearchResult{}, false, err
			}
			for _, indexed := range index.Skills {
				if indexed.Identity == identity || indexed.Name == identity {
					return SearchResult{Registry: registryName, Skill: indexed}, true, nil
				}
			}
			return SearchResult{}, false, nil
		}
	}
	for name, reg := range cfg.Registries {
		root, _, err := registryRoot(name, reg)
		if err != nil {
			return SearchResult{}, false, err
		}
		index, err := LoadIndex(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return SearchResult{}, false, err
		}
		for _, indexed := range index.Skills {
			if indexed.Identity == spec || indexed.Name == spec {
				return SearchResult{Registry: name, Skill: indexed}, true, nil
			}
		}
	}
	return SearchResult{}, false, nil
}

func ValidateIndex(name string, reg config.Registry) (int, error) {
	root, _, err := registryRoot(name, reg)
	if err != nil {
		return 0, err
	}
	index, err := LoadIndex(root)
	if err != nil {
		return 0, err
	}
	seen := map[string]bool{}
	for _, indexed := range index.Skills {
		if indexed.Identity == "" {
			return 0, fmt.Errorf("index contains skill with empty identity")
		}
		if seen[indexed.Identity] {
			return 0, fmt.Errorf("duplicate identity %s", indexed.Identity)
		}
		seen[indexed.Identity] = true
		sourcePath := filepath.Join(root, filepath.FromSlash(indexed.SourcePath))
		meta, err := skill.LoadCompatibleMetadata(sourcePath, name)
		if err != nil {
			return 0, fmt.Errorf("validate %s: %w", indexed.Identity, err)
		}
		if skill.Identity(meta.Namespace, meta.Name) != indexed.Identity {
			return 0, fmt.Errorf("identity mismatch for %s", indexed.Identity)
		}
		checksum, err := skill.ChecksumDir(sourcePath)
		if err != nil {
			return 0, err
		}
		if checksum != indexed.Checksum {
			return 0, fmt.Errorf("checksum mismatch for %s", indexed.Identity)
		}
	}
	return len(index.Skills), nil
}

func matches(indexed IndexSkill, query string) bool {
	fields := []string{
		indexed.Identity,
		indexed.Name,
		indexed.Namespace,
		indexed.Version,
		indexed.Description,
		strings.Join(indexed.Targets, " "),
		strings.Join(indexed.Tags, " "),
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}

func ResolveIndexedPath(root string, spec string) (string, bool, error) {
	index, err := LoadIndex(root)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	for _, indexed := range index.Skills {
		if indexed.Identity == spec || indexed.Name == spec {
			return filepath.Join(root, filepath.FromSlash(indexed.SourcePath)), true, nil
		}
	}
	return "", false, nil
}

func LoadIndex(root string) (Index, error) {
	data, err := os.ReadFile(filepath.Join(root, IndexFileName))
	if err != nil {
		return Index{}, err
	}
	var index Index
	if err := json.Unmarshal(data, &index); err != nil {
		return Index{}, err
	}
	return index, nil
}

func GenerateIndex(name string, reg config.Registry) (Index, string, error) {
	root, sourceType, err := registryRoot(name, reg)
	if err != nil {
		return Index{}, "", err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return Index{}, "", err
	}
	index := Index{
		Registry:    name,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sourcePath := filepath.Join(root, entry.Name())
		meta, err := skill.LoadCompatibleMetadata(sourcePath, name)
		if err != nil {
			return Index{}, "", fmt.Errorf("index %s: %w", entry.Name(), err)
		}
		checksum, err := skill.ChecksumDir(sourcePath)
		if err != nil {
			return Index{}, "", err
		}
		index.Skills = append(index.Skills, IndexSkill{
			Identity:    skill.Identity(meta.Namespace, meta.Name),
			Name:        meta.Name,
			Namespace:   meta.Namespace,
			Version:     meta.Version,
			Description: meta.Description,
			Targets:     meta.Targets,
			Tags:        meta.Tags,
			SourceType:  sourceType,
			SourcePath:  filepath.ToSlash(entry.Name()),
			Checksum:    checksum,
		})
	}
	indexPath := filepath.Join(root, IndexFileName)
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return Index{}, "", err
	}
	data = append(data, '\n')
	if err := os.WriteFile(indexPath, data, 0o644); err != nil {
		return Index{}, "", err
	}
	return index, indexPath, nil
}

func registryRoot(name string, reg config.Registry) (string, string, error) {
	switch reg.Type {
	case "local":
		return reg.Path, "registry", nil
	case "git":
		cachePath, err := EnsureGitCache(name, reg.URL)
		if err != nil {
			return "", "", err
		}
		return cachePath, "git", nil
	default:
		return "", "", fmt.Errorf("unsupported registry type %q", reg.Type)
	}
}
