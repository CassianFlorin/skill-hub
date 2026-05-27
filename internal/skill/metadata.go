package skill

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Metadata struct {
	Name        string
	Version     string
	Description string
	Author      string
	Entry       string
	Targets     []string
	Tags        []string
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
		case "version":
			meta.Version = value
		case "description":
			meta.Description = value
		case "author":
			meta.Author = value
		case "entry":
			meta.Entry = value
		}
	}
	if err := scanner.Err(); err != nil {
		return Metadata{}, err
	}
	if meta.Name == "" {
		return Metadata{}, fmt.Errorf("skill.yaml missing name")
	}
	if meta.Version == "" {
		return Metadata{}, fmt.Errorf("skill.yaml missing version")
	}
	if _, err := os.Stat(filepath.Join(dir, meta.Entry)); err != nil {
		return Metadata{}, fmt.Errorf("entry %s: %w", meta.Entry, err)
	}
	return meta, nil
}
