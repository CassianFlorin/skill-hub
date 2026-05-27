# skill-hub

`skill-hub` is an MVP Skill package manager for AI agents. It treats a Skill as an installable package with metadata, installed files, lockfile state, and deploy targets.

## MVP scope

Implemented commands:

```bash
skillhub init
skillhub registry add local company ./examples/local-registry
skillhub registry add git company git@gitlab.example.com:ai/skills.git
skillhub install company/java-review
skillhub install ./examples/local-registry/java-review
skillhub list
skillhub update
skillhub deploy codex
```

The MVP supports local registry installation and git registry installation. Git registries are cloned into a local cache on first use and refreshed with `git pull --ff-only` during install/update.

## State files

- Project config: `skillhub.yaml`
- Runtime home: `$SKILLHUB_HOME`, defaulting to `~/.skillhub`
- Git registry cache: `$SKILLHUB_HOME/cache/registries/<registry-name>`
- Installed skills: `$SKILLHUB_HOME/installed/<skill-name>`
- Lockfile: `$SKILLHUB_HOME/skillhub.lock`
- Codex deploy target: `$SKILLHUB_CODEX_DIR`, defaulting to `~/.codex/skills`

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
./skillhub deploy codex
```

Git registry example:

```bash
./skillhub registry add git team git@github.com:your-org/skills.git
./skillhub install team/java-review
./skillhub update
```

For git registries, `skillhub update` refreshes the cached repository and updates installed skills when their `skill.yaml` version changes.
