package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cassian/skill-hub/internal/config"
	"github.com/cassian/skill-hub/internal/skill"
)

const (
	IndexFileName      = "skillhub.index.json"
	IndexSchemaVersion = "2"
	SourceTypeRegistry = "registry"
	SourceTypeGit      = "git"
	TargetCodex        = "codex"
	TargetClaude       = "claude"
	TargetGemini       = "gemini"
	TrustOfficial      = "official"
	TrustCurated       = "curated"
	TrustCommunity     = "community"
	TrustPrivate       = "private"
	TrustUnknown       = "unknown"
)

type Index struct {
	SchemaVersion string       `json:"schema_version"`
	Registry      string       `json:"registry"`
	GeneratedAt   string       `json:"generated_at"`
	Skills        []IndexSkill `json:"skills"`
}

type IndexSkill struct {
	Identity    string      `json:"identity"`
	Name        string      `json:"name"`
	Namespace   string      `json:"namespace"`
	Version     string      `json:"version"`
	Description string      `json:"description"`
	Targets     []string    `json:"targets"`
	Tags        []string    `json:"tags,omitempty"`
	Source      IndexSource `json:"source"`
	Maintainers []string    `json:"maintainers,omitempty"`
	License     string      `json:"license,omitempty"`
	Trust       IndexTrust  `json:"trust"`
	Featured    bool        `json:"featured"`
	UpdatedAt   string      `json:"updated_at"`
	Checksum    string      `json:"checksum,omitempty"`
}

type IndexSource struct {
	Type string `json:"type"`
	URL  string `json:"url,omitempty"`
	Path string `json:"path"`
	Ref  string `json:"ref,omitempty"`
}

type IndexTrust struct {
	Level      string `json:"level"`
	ReviewedAt string `json:"reviewed_at,omitempty"`
	Reviewer   string `json:"reviewer,omitempty"`
}

type SearchResult struct {
	Registry string
	Skill    IndexSkill
}

func SearchIndexes(cfg config.Config, query string) ([]SearchResult, error) {
	query = strings.ToLower(query)
	var results []SearchResult
	for name, reg := range cfg.Registries {
		root, err := registryRootNoSync(name, reg)
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
	sortSearchResultsByQuery(results, query)
	return results, nil
}

func FindIndexedSkill(cfg config.Config, spec string) (SearchResult, bool, error) {
	registryName, identity, ok := strings.Cut(spec, "/")
	if ok {
		if reg, exists := cfg.Registries[registryName]; exists {
			root, err := registryRootNoSync(registryName, reg)
			if err != nil {
				return SearchResult{}, false, err
			}
			indexed, found, err := ResolveIndexedSkill(root, identity)
			if err != nil {
				if os.IsNotExist(err) {
					return SearchResult{}, false, nil
				}
				return SearchResult{}, false, err
			}
			return SearchResult{Registry: registryName, Skill: indexed}, found, nil
		}
	}
	for name, reg := range cfg.Registries {
		root, err := registryRootNoSync(name, reg)
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
	if err := validateRegistrySources(root, name, index); err != nil {
		return 0, err
	}
	return len(index.Skills), nil
}

func validateRegistrySources(root string, name string, index Index) error {
	seen := map[string]bool{}
	for _, indexed := range index.Skills {
		if indexed.Source.Type != SourceTypeRegistry {
			continue
		}
		if seen[indexed.Identity] {
			continue
		}
		seen[indexed.Identity] = true
		sourcePath, err := RegistrySourcePath(root, indexed.Source.Path)
		if err != nil {
			return fmt.Errorf("validate %s: %w", indexed.Identity, err)
		}
		meta, err := skill.LoadCompatibleMetadata(sourcePath, name)
		if err != nil {
			return fmt.Errorf("validate %s: %w", indexed.Identity, err)
		}
		if skill.Identity(meta.Namespace, meta.Name) != indexed.Identity {
			return fmt.Errorf("identity mismatch for %s", indexed.Identity)
		}
		if indexed.Checksum != "" {
			checksum, err := skill.ChecksumDir(sourcePath)
			if err != nil {
				return err
			}
			if checksum != indexed.Checksum {
				return fmt.Errorf("checksum mismatch for %s", indexed.Identity)
			}
		}
	}
	return nil
}

func RegistrySourcePath(root string, sourcePath string) (string, error) {
	if filepath.IsAbs(sourcePath) {
		return "", fmt.Errorf("source path %q must be relative", sourcePath)
	}
	if strings.Contains(sourcePath, "\\") {
		return "", fmt.Errorf("source path %q must use slash separators", sourcePath)
	}
	for _, part := range strings.Split(sourcePath, "/") {
		if part == ".." {
			return "", fmt.Errorf("source path %q escapes registry root", sourcePath)
		}
	}
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	target, err := filepath.Abs(filepath.Join(cleanRoot, filepath.FromSlash(sourcePath)))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(cleanRoot, target)
	if err != nil {
		return "", err
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
		return "", fmt.Errorf("source path %q escapes registry root", sourcePath)
	}
	return target, nil
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
		indexed.Source.Type,
		indexed.Source.URL,
		indexed.Source.Path,
		indexed.Source.Ref,
		strings.Join(indexed.Maintainers, " "),
		indexed.License,
		indexed.Trust.Level,
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}

func ResolveIndexedPath(root string, spec string) (string, bool, error) {
	indexed, ok, err := ResolveIndexedSkill(root, spec)
	if err != nil || !ok {
		return "", ok, err
	}
	if indexed.Source.Type != SourceTypeRegistry {
		return "", false, nil
	}
	sourcePath, err := RegistrySourcePath(root, indexed.Source.Path)
	if err != nil {
		return "", false, err
	}
	return sourcePath, true, nil
}

func ResolveIndexedSkill(root string, spec string) (IndexSkill, bool, error) {
	index, err := LoadIndex(root)
	if err != nil {
		if os.IsNotExist(err) {
			return IndexSkill{}, false, nil
		}
		return IndexSkill{}, false, err
	}
	for _, indexed := range index.Skills {
		if indexed.Identity == spec || indexed.Name == spec {
			return indexed, true, nil
		}
	}
	return IndexSkill{}, false, nil
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
	if err := validateCatalogSchema(index); err != nil {
		return Index{}, err
	}
	return index, nil
}

func GenerateIndex(name string, reg config.Registry) (Index, string, error) {
	root, _, err := registryRoot(name, reg)
	if err != nil {
		return Index{}, "", err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return Index{}, "", err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	index := Index{
		SchemaVersion: IndexSchemaVersion,
		Registry:      name,
		GeneratedAt:   now,
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
			Source:      IndexSource{Type: SourceTypeRegistry, Path: filepath.ToSlash(entry.Name())},
			Maintainers: maintainersFrom(meta),
			Trust:       IndexTrust{Level: TrustPrivate},
			Featured:    false,
			UpdatedAt:   now,
			Checksum:    checksum,
		})
	}
	if err := validateCatalogSchema(index); err != nil {
		return Index{}, "", err
	}
	if err := validateRegistrySources(root, name, index); err != nil {
		return Index{}, "", err
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

func maintainersFrom(meta skill.Metadata) []string {
	if meta.Author == "" {
		return nil
	}
	return []string{meta.Author}
}

func validateCatalogSchema(index Index) error {
	if index.SchemaVersion != IndexSchemaVersion {
		return fmt.Errorf("unsupported index schema %q, expected %q", index.SchemaVersion, IndexSchemaVersion)
	}
	if index.Registry == "" {
		return fmt.Errorf("index missing registry")
	}
	if index.GeneratedAt == "" {
		return fmt.Errorf("index missing generated_at")
	}
	seen := map[string]bool{}
	for _, indexed := range index.Skills {
		if indexed.Identity == "" || indexed.Name == "" || indexed.Namespace == "" {
			return fmt.Errorf("index contains skill with incomplete identity fields")
		}
		if indexed.Version == "" {
			return fmt.Errorf("index %s missing version", indexed.Identity)
		}
		if indexed.Description == "" {
			return fmt.Errorf("index %s missing description", indexed.Identity)
		}
		if len(indexed.Targets) == 0 {
			return fmt.Errorf("index %s missing targets", indexed.Identity)
		}
		for _, target := range indexed.Targets {
			if !validTarget(target) {
				return fmt.Errorf("index %s unsupported target %q", indexed.Identity, target)
			}
		}
		if indexed.Source.Type == "" || indexed.Source.Path == "" {
			return fmt.Errorf("index %s missing source", indexed.Identity)
		}
		if !validSourceType(indexed.Source.Type) {
			return fmt.Errorf("index %s unsupported source type %q", indexed.Identity, indexed.Source.Type)
		}
		if indexed.Source.Type == SourceTypeGit && indexed.Source.URL == "" {
			return fmt.Errorf("index %s git source missing url", indexed.Identity)
		}
		if indexed.Trust.Level == "" {
			return fmt.Errorf("index %s missing trust level", indexed.Identity)
		}
		if !validTrustLevel(indexed.Trust.Level) {
			return fmt.Errorf("index %s unsupported trust level %q", indexed.Identity, indexed.Trust.Level)
		}
		if indexed.UpdatedAt == "" {
			return fmt.Errorf("index %s missing updated_at", indexed.Identity)
		}
		if indexed.Featured && len(indexed.Tags) == 0 {
			return fmt.Errorf("featured skill %s missing tags", indexed.Identity)
		}
		if seen[indexed.Identity] {
			return fmt.Errorf("duplicate identity %s", indexed.Identity)
		}
		seen[indexed.Identity] = true
	}
	return nil
}

func validTarget(target string) bool {
	switch target {
	case TargetCodex, TargetClaude, TargetGemini:
		return true
	default:
		return false
	}
}

func validSourceType(sourceType string) bool {
	switch sourceType {
	case SourceTypeRegistry, SourceTypeGit:
		return true
	default:
		return false
	}
}

func validTrustLevel(level string) bool {
	switch level {
	case TrustOfficial, TrustCurated, TrustCommunity, TrustPrivate, TrustUnknown:
		return true
	default:
		return false
	}
}

type RegistryStatus struct {
	Name        string
	Type        string
	Location    string
	CachePath   string
	SkillCount  int
	GeneratedAt string
}

func ListRegistries(cfg config.Config) []RegistryStatus {
	names := make([]string, 0, len(cfg.Registries))
	for name := range cfg.Registries {
		names = append(names, name)
	}
	sort.Strings(names)

	statuses := make([]RegistryStatus, 0, len(names))
	for _, name := range names {
		reg := cfg.Registries[name]
		status := RegistryStatus{Name: name, Type: reg.Type}
		switch reg.Type {
		case "local":
			status.Location = reg.Path
			status.CachePath = reg.Path
		case "git":
			status.Location = reg.URL
			if cachePath, err := GitCachePath(name); err == nil {
				status.CachePath = cachePath
			}
		}
		if root, err := registryRootNoSync(name, reg); err == nil {
			if index, err := LoadIndex(root); err == nil {
				status.SkillCount = len(index.Skills)
				status.GeneratedAt = index.GeneratedAt
			}
		}
		statuses = append(statuses, status)
	}
	return statuses
}

type CatalogFilter struct {
	Registry  string
	Target    string
	Tag       string
	Namespace string
	Trust     string
	Featured  *bool
	Official  bool
}

func ListCatalog(cfg config.Config, filter CatalogFilter) ([]SearchResult, error) {
	var results []SearchResult
	for name, reg := range cfg.Registries {
		if filter.Registry != "" && filter.Registry != name {
			continue
		}
		root, err := registryRootNoSync(name, reg)
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
			if filter.Featured != nil && indexed.Featured != *filter.Featured {
				continue
			}
			if filter.Official && indexed.Trust.Level != TrustOfficial {
				continue
			}
			if filter.Target != "" && !contains(indexed.Targets, filter.Target) {
				continue
			}
			if filter.Tag != "" && !contains(indexed.Tags, filter.Tag) {
				continue
			}
			if filter.Namespace != "" && indexed.Namespace != filter.Namespace {
				continue
			}
			if filter.Trust != "" && indexed.Trust.Level != filter.Trust {
				continue
			}
			results = append(results, SearchResult{Registry: name, Skill: indexed})
		}
	}
	sortSearchResults(results)
	return results, nil
}

func sortSearchResults(results []SearchResult) {
	sort.Slice(results, func(i, j int) bool {
		left := results[i].Registry + "/" + results[i].Skill.Identity
		right := results[j].Registry + "/" + results[j].Skill.Identity
		return left < right
	})
}

func sortSearchResultsByQuery(results []SearchResult, query string) {
	sort.Slice(results, func(i, j int) bool {
		leftScore := matchScore(results[i].Skill, query)
		rightScore := matchScore(results[j].Skill, query)
		if leftScore != rightScore {
			return leftScore < rightScore
		}
		if results[i].Skill.Featured != results[j].Skill.Featured {
			return results[i].Skill.Featured
		}
		leftOfficial := results[i].Skill.Trust.Level == TrustOfficial
		rightOfficial := results[j].Skill.Trust.Level == TrustOfficial
		if leftOfficial != rightOfficial {
			return leftOfficial
		}
		left := results[i].Registry + "/" + results[i].Skill.Identity
		right := results[j].Registry + "/" + results[j].Skill.Identity
		return left < right
	})
}

func matchScore(indexed IndexSkill, query string) int {
	identity := strings.ToLower(indexed.Identity)
	name := strings.ToLower(indexed.Name)
	if identity == query || name == query {
		return 0
	}
	if strings.HasPrefix(identity, query) || strings.HasPrefix(name, query) {
		return 1
	}
	for _, tag := range indexed.Tags {
		tag = strings.ToLower(tag)
		if tag == query {
			return 2
		}
		if strings.Contains(tag, query) {
			return 3
		}
	}
	if strings.Contains(strings.ToLower(indexed.Description), query) {
		return 4
	}
	return 5
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func SyncRegistry(name string, reg config.Registry) (int, error) {
	root, _, err := registryRoot(name, reg)
	if err != nil {
		return 0, err
	}
	index, err := LoadIndex(root)
	if err != nil {
		return 0, err
	}
	if err := validateRegistrySources(root, name, index); err != nil {
		return 0, err
	}
	return len(index.Skills), nil
}

func registryRootNoSync(name string, reg config.Registry) (string, error) {
	switch reg.Type {
	case "local":
		return reg.Path, nil
	case "git":
		return GitCachePath(name)
	default:
		return "", fmt.Errorf("unsupported registry type %q", reg.Type)
	}
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
