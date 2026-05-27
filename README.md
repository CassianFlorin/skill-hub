# skill-hub

`skill-hub` is a Skill package manager for AI agents. It treats a Skill as an installable package with metadata, installed files, lockfile state, registry indexing, and runtime deploy targets.

## Current scope

Implemented commands:

```bash
skillhub init
skillhub registry add local company ./examples/local-registry
skillhub registry add git company git@gitlab.example.com:ai/skills.git
skillhub registry list
skillhub registry sync hub
skillhub registry sync --all
skillhub registry index generate company
skillhub registry index validate company
skillhub catalog list
skillhub catalog featured
skillhub catalog list --target codex --tag java
skillhub search java
skillhub info company/platform-team/java-review
skillhub install company/java-review
skillhub install company/java-review@1.2.0
skillhub install ./examples/local-registry/java-review
skillhub rollback platform-team/java-review
skillhub uninstall platform-team/java-review
skillhub list
skillhub update
skillhub deploy codex
skillhub deploy claude
skillhub deploy status
```

The CLI supports local registry installation, git registry installation, and catalog discovery. Git registries are cloned into a local cache during explicit sync or install, and refreshed with `git pull --ff-only`.

## Catalog discovery

`skillhub init` configures the default `hub` registry at `https://github.com/CassianFlorin/skill-hub-registry.git`. Run `skillhub registry sync hub` to clone or refresh it locally, then use `skillhub catalog featured`, `skillhub catalog list`, `skillhub search <query>`, and `skillhub info <registry>/<identity>` to discover installable skills.

Registries use `skillhub.index.json` schema v2. Old index files without `schema_version: "2"` are rejected by validation.

```json
{
  "schema_version": "2",
  "registry": "hub",
  "generated_at": "2026-05-27T00:00:00Z",
  "skills": [
    {
      "identity": "official/java-review",
      "name": "java-review",
      "namespace": "official",
      "version": "0.1.0",
      "description": "Review Java service code with repo-aware checks.",
      "targets": ["codex", "claude"],
      "tags": ["java", "review"],
      "source": {
        "type": "git",
        "url": "https://github.com/CassianFlorin/skills.git",
        "path": "java-review",
        "ref": "v0.1.0"
      },
      "maintainers": ["CassianFlorin"],
      "license": "MIT",
      "trust": {
        "level": "official",
        "reviewed_at": "2026-05-27"
      },
      "featured": true,
      "updated_at": "2026-05-27"
    }
  ]
}
```

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
./skillhub registry sync team
./skillhub catalog list --registry team
./skillhub install team/java-review
./skillhub update
```

For git registries, `skillhub update` refreshes the cached repository and updates installed skills when their `skill.yaml` version changes.

Pinned install examples:

```bash
./skillhub install company/java-review@1.2.0
./skillhub install team/java-review@v1.2.0
./skillhub install team/java-review@main
./skillhub install team/java-review@<commit-sha>
```

For local registries, the pinned value must match the Skill metadata version. For git registries, the pinned value is treated as a git ref and is recorded in `skillhub.lock` together with the resolved commit.

Rollback example:

```bash
./skillhub rollback platform-team/java-review
```

skill-hub saves a history snapshot before replacing an installed Skill. Rollback restores the latest previous installed copy and updates `skillhub.lock`.

Uninstall examples:

```bash
./skillhub uninstall platform-team/java-review
./skillhub uninstall platform-team/java-review --deployed
```

By default, uninstall removes the installed store copy and lockfile entry only. Use `--deployed` to also remove Codex and Claude runtime copies.

Registry index example:

```bash
./skillhub registry index generate company
./skillhub registry index validate company
./skillhub registry list
./skillhub catalog list --registry company
./skillhub search java
./skillhub info company/platform-team/java-review
./skillhub install company/platform-team/java-review
```

Claude deploy example:

```bash
export SKILLHUB_CLAUDE_DIR="$PWD/.skillhub-e2e/claude"
./skillhub deploy claude platform-team/java-review --force
./skillhub deploy status
./skillhub deploy status codex
```

## Release

CI runs `go test ./...` and `go build -v ./cmd/skillhub` on pushes and pull requests.

Tagged releases publish multi-platform archives through GitHub Actions:

```bash
git tag v0.1.0
git push origin v0.1.0
```
