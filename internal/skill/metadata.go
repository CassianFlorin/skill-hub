package skill

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

const Unversioned = "unversioned"

type Metadata struct {
	Name        string
	Namespace   string
	Version     string
	Description string
	Author      string
	Entry       string
	Targets     []string
	Tags        []string
	Generated   bool
}

func LoadMetadata(dir string) (Metadata, error) {
	data, err := os.Open(filepath.Join(dir, "skill.yaml"))
	if err != nil {
		return Metadata{}, err
	}
	defer data.Close()

	meta := Metadata{Entry: "SKILL.md"}
	var listKey string
	scanner := bufio.NewScanner(data)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "- ") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			switch listKey {
			case "targets":
				meta.Targets = append(meta.Targets, value)
			case "tags":
				meta.Tags = append(meta.Tags, value)
			}
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return Metadata{}, fmt.Errorf("invalid skill.yaml line %q", line)
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		listKey = ""
		if value == "" {
			listKey = key
			continue
		}
		switch key {
		case "name":
			meta.Name = value
		case "namespace":
			meta.Namespace = value
		case "version":
			meta.Version = value
		case "description":
			meta.Description = value
		case "author":
			meta.Author = value
		case "entry":
			meta.Entry = value
		case "generated":
			meta.Generated = value == "true"
		}
	}
	if err := scanner.Err(); err != nil {
		return Metadata{}, err
	}
	if meta.Name == "" {
		return Metadata{}, fmt.Errorf("skill.yaml missing name")
	}
	if meta.Version == "" {
		meta.Version = Unversioned
	}
	if _, err := os.Stat(filepath.Join(dir, meta.Entry)); err != nil {
		return Metadata{}, fmt.Errorf("entry %s: %w", meta.Entry, err)
	}
	return meta, nil
}

func LoadCompatibleMetadata(dir string, fallbackNamespace string) (Metadata, error) {
	meta, err := LoadMetadata(dir)
	if err == nil {
		meta.Namespace = ResolveNamespace(meta, fallbackNamespace)
		return meta, nil
	}
	if !os.IsNotExist(err) {
		return Metadata{}, err
	}
	if _, statErr := os.Stat(filepath.Join(dir, "SKILL.md")); statErr != nil {
		return Metadata{}, err
	}
	meta = Metadata{
		Name:      filepath.Base(dir),
		Namespace: fallbackNamespace,
		Version:   Unversioned,
		Entry:     "SKILL.md",
		Generated: true,
	}
	meta.Namespace = ResolveNamespace(meta, fallbackNamespace)
	return meta, nil
}

func ResolveNamespace(meta Metadata, fallback string) string {
	switch {
	case meta.Namespace != "":
		return meta.Namespace
	case meta.Author != "":
		return meta.Author
	case fallback != "":
		return fallback
	}
	if current, err := user.Current(); err == nil && current.Username != "" {
		parts := strings.Split(current.Username, string(os.PathSeparator))
		return parts[len(parts)-1]
	}
	return "unknown"
}

func Identity(namespace string, name string) string {
	return namespace + "/" + name
}

func SafeIdentity(identity string) string {
	replacer := strings.NewReplacer("/", "__", "\\", "__", ":", "_", " ", "-")
	return replacer.Replace(identity)
}

func WriteGeneratedMetadata(dir string, meta Metadata) error {
	var builder strings.Builder
	builder.WriteString("name: " + meta.Name + "\n")
	builder.WriteString("namespace: " + meta.Namespace + "\n")
	builder.WriteString("entry: " + meta.Entry + "\n")
	builder.WriteString("version: " + meta.Version + "\n")
	builder.WriteString("generated: true\n")
	return os.WriteFile(filepath.Join(dir, "skill.yaml"), []byte(builder.String()), 0o644)
}
