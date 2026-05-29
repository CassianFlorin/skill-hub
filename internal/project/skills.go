package project

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/cassian/skill-hub/internal/skill"
)

var skillRoots = []string{
	filepath.Join(".skillhub", "skills"),
	filepath.Join(".codex", "skills"),
	filepath.Join(".claude", "skills"),
	filepath.Join(".agents", "skills"),
	filepath.Join("agent", "skills"),
}

type Skill struct {
	Identity string
	Version  string
	RelPath  string
}

func DiscoverSkills(workDir string) ([]Skill, error) {
	var found []Skill
	for _, root := range skillRoots {
		absRoot := filepath.Join(workDir, root)
		if _, err := os.Stat(absRoot); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if err := filepath.WalkDir(absRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !entry.IsDir() {
				return nil
			}
			if _, err := os.Stat(filepath.Join(path, "SKILL.md")); err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			meta, err := skill.LoadCompatibleMetadata(path, "project")
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(workDir, path)
			if err != nil {
				return err
			}
			found = append(found, Skill{
				Identity: skill.Identity(meta.Namespace, meta.Name),
				Version:  meta.Version,
				RelPath:  filepath.ToSlash(rel),
			})
			return filepath.SkipDir
		}); err != nil {
			return nil, err
		}
	}
	sort.Slice(found, func(i, j int) bool {
		left := found[i].Identity + "\x00" + found[i].RelPath
		right := found[j].Identity + "\x00" + found[j].RelPath
		return left < right
	})
	return found, nil
}
