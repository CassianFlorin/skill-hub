package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChecksumDirDeterministic(t *testing.T) {
	build := func() string {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# a\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "scripts", "run.sh"), []byte("echo hi\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		sum, err := ChecksumDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		return sum
	}
	first := build()
	second := build()
	if first != second {
		t.Errorf("checksum not deterministic: %q vs %q", first, second)
	}
	if !strings.HasPrefix(first, "sha256:") {
		t.Errorf("checksum missing sha256 prefix: %q", first)
	}
}

func TestChecksumDirDetectsContentChange(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(file, []byte("v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := ChecksumDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := ChecksumDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Error("checksum did not change after content edit")
	}
}

func TestChecksumDirDetectsRename(t *testing.T) {
	// Same bytes under a different filename must change the checksum,
	// because the path is mixed into the hash.
	dir1 := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir1, "a.md"), []byte("same\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sum1, err := ChecksumDir(dir1)
	if err != nil {
		t.Fatal(err)
	}
	dir2 := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir2, "b.md"), []byte("same\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sum2, err := ChecksumDir(dir2)
	if err != nil {
		t.Fatal(err)
	}
	if sum1 == sum2 {
		t.Error("checksum ignored filename in path")
	}
}
