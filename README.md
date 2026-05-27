# skill-hub

`skill-hub` is a Skill package manager for AI agents. It treats a Skill as an installable package with metadata, installed files, lockfile state, registry indexing, and runtime deploy targets.

## Current scope

Implemented commands:

```bash
skillhub version
skillhub doctor
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
skillhub catalog tags --registry hub
skillhub catalog targets --registry hub
skillhub catalog namespaces --registry hub
skillhub catalog trust --registry hub
skillhub catalog list --registry hub --namespace official --trust official
skillhub catalog list --registry hub --json
skillhub catalog export --registry hub --output ./public/catalog
skillhub search java
skillhub search git --json
skillhub info company/platform-team/java-review
skillhub info company/platform-team/java-review --json
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

Discovery commands:

```bash
skillhub catalog featured --registry hub
skillhub catalog list --registry hub --target codex
skillhub catalog list --registry hub --target claude
skillhub catalog list --registry hub --tag git
skillhub catalog tags --registry hub
skillhub catalog targets --registry hub
skillhub catalog namespaces --registry hub
skillhub catalog trust --registry hub
skillhub catalog list --registry hub --namespace official --trust official
skillhub catalog export --registry hub --output ./public/catalog
skillhub search git
skillhub info hub/official/git-commit-cn
```

`search` ranks stronger matches first: exact or prefix identity/name matches, then tag matches, then description matches, with featured and official skills preferred when the match strength is otherwise equal.

For automation, catalog, search, and info commands support JSON output:

```bash
skillhub catalog list --registry hub --json
skillhub catalog featured --registry hub --json
skillhub catalog tags --registry hub --json
skillhub catalog targets --registry hub --json
skillhub search git --json
skillhub info hub/official/git-commit-cn --json
```

Static catalog export writes a browsable `index.html` and structured `catalog.json` for publishing or reviewing a marketplace snapshot:

```bash
skillhub catalog export --registry hub --output ./public/catalog
skillhub catalog export --registry hub --target codex --namespace official --output ./public/codex
```

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
export SKILLHUB_CLAUDE_DIR="$PWD/.skillhub-e2e/claude"
./skillhub init
./skillhub version
./skillhub doctor
./skillhub registry add local company "$PWD/examples/local-registry"
./skillhub install company/java-review
./skillhub list
./skillhub deploy codex platform-team/java-review --dry-run
./skillhub deploy codex platform-team/java-review --force
./skillhub deploy status
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
./skillhub catalog tags --registry company
./skillhub catalog targets --registry company
./skillhub catalog namespaces --registry company
./skillhub catalog trust --registry company
./skillhub catalog export --registry company --output ./public/catalog
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

Deploy respects the installed Skill `targets` metadata. Skills with no targets are treated as compatible with all supported runtimes for backward compatibility. In batch deploys, unsupported skills are skipped; explicitly deploying an unsupported skill to a runtime returns an error.

Dry-run mode performs the same selection and conflict checks without writing runtime directories or updating `skillhub.lock`:

```bash
./skillhub deploy codex --dry-run
./skillhub deploy claude platform-team/java-review --dry-run
```

Without `--force`, deploy preflights all selected skills before copying files. If any selected target already exists, the command fails before partial writes. Use `--force` to replace existing runtime copies.

Deploy status values:

- `deployed`: runtime copy matches the installed skill checksum.
- `missing`: the skill supports the runtime but has not been deployed there.
- `drifted`: runtime copy exists but differs from the installed skill checksum.
- `unsupported`: the skill does not list that runtime in `targets`.

## Release

CI runs `go test ./...` and `go build -v ./cmd/skillhub` on pushes and pull requests.

Tagged releases publish multi-platform archives through GitHub Actions:

```bash
git tag v0.1.0
git push origin v0.1.0
```
