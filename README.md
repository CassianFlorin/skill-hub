# skill-hub

`skill-hub` is a Skill package manager for AI agents. It treats a Skill as an installable package with metadata, installed files, lockfile state, registry indexing, and runtime deploy targets.

## MVP scope

Implemented commands:

```bash
skillhub init
skillhub registry add local company ./examples/local-registry
skillhub registry add git company git@gitlab.example.com:ai/skills.git
skillhub registry index generate company
skillhub registry index validate company
skillhub search java
skillhub info company/platform-team/java-review
skillhub install company/java-review
skillhub install ./examples/local-registry/java-review
skillhub list
skillhub update
skillhub deploy codex
skillhub deploy claude
```

The CLI supports local registry installation and git registry installation. Git registries are cloned into a local cache on first use and refreshed with `git pull --ff-only` during install/update.

Skills are displayed as `namespace/name`, using this priority:

1. `skill.yaml.namespace`
2. `skill.yaml.author`
3. registry name
4. local user name
5. `unknown`

Existing Skill directories that only contain `SKILL.md` can still be installed. skill-hub writes a generated `skill.yaml` into the installed copy so the lockfile and deploy pipeline can use the same metadata model.

## State files

- Project config: `skillhub.yaml`
- Runtime home: `$SKILLHUB_HOME`, defaulting to `~/.skillhub`
- Git registry cache: `$SKILLHUB_HOME/cache/registries/<registry-name>`
- Installed skills: `$SKILLHUB_HOME/installed/<skill-name>`
- Lockfile: `$SKILLHUB_HOME/skillhub.lock`
- Codex deploy target: `$SKILLHUB_CODEX_DIR`, defaulting to `~/.codex/skills`
- Claude deploy target: `$SKILLHUB_CLAUDE_DIR`, defaulting to `~/.claude/skills`

`skillhub.yaml` and `skillhub.lock` are JSON documents with YAML-compatible filenames for this MVP. That keeps the first version dependency-free while preserving the intended file names.

## Skill structure

```text
java-review/
├── skill.yaml
├── SKILL.md
├── references/
├── scripts/
└── assets/
```

Minimal `skill.yaml`:

```yaml
name: java-review
namespace: platform-team
version: 1.2.0
description: Java review skill
entry: SKILL.md
targets:
  - codex
tags:
  - java
```

## Development

```bash
go test ./...
go build ./cmd/skillhub
```

Example local run without touching real user state:

```bash
export SKILLHUB_HOME="$PWD/.skillhub-e2e/home"
export SKILLHUB_CODEX_DIR="$PWD/.skillhub-e2e/codex"
./skillhub init
./skillhub registry add local company "$PWD/examples/local-registry"
./skillhub install company/java-review
./skillhub list
./skillhub deploy codex platform-team/java-review --dry-run
./skillhub deploy codex platform-team/java-review --force
```

Git registry example:

```bash
./skillhub registry add git team git@github.com:your-org/skills.git
./skillhub install team/java-review
./skillhub update
```

For git registries, `skillhub update` refreshes the cached repository and updates installed skills when their `skill.yaml` version changes.

Registry index example:

```bash
./skillhub registry index generate company
./skillhub registry index validate company
./skillhub search java
./skillhub info company/platform-team/java-review
./skillhub install company/platform-team/java-review
```

Claude deploy example:

```bash
export SKILLHUB_CLAUDE_DIR="$PWD/.skillhub-e2e/claude"
./skillhub deploy claude platform-team/java-review --force
```

## Release

CI runs `go test ./...` and `go build -v ./cmd/skillhub` on pushes and pull requests.

Tagged releases publish multi-platform archives through GitHub Actions:

```bash
git tag v0.1.0
git push origin v0.1.0
```
