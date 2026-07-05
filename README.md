<div align="center">

# skill-hub

**A Skill package manager for AI agents.**

Install, version, update, roll back, and deploy Skills across Codex, Claude, Gemini, and Hermes from one CLI.

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](go.mod)
[![npm](https://img.shields.io/npm/v/@cassianflorin/skillhub?logo=npm&label=npm)](https://www.npmjs.com/package/@cassianflorin/skillhub)
[![Homebrew](https://img.shields.io/badge/Homebrew-CassianFlorin%2Ftap-FBB040?logo=homebrew&logoColor=black)](https://github.com/CassianFlorin/homebrew-tap)
[![Release](https://img.shields.io/github/v/release/CassianFlorin/skill-hub?label=release)](https://github.com/CassianFlorin/skill-hub/releases)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

[English](README.md) · [简体中文](README.zh-CN.md) · [繁體中文](README.zh-TW.md)

</div>

`skill-hub` treats a Skill as an installable package with metadata, registry indexing, lockfile state, rollback history, and runtime deploy targets.

Current release line: `v1.3.x`.

## Does This Manage My Existing Skills?

skill-hub separates local Skill state into three layers:

| Layer | Location | Meaning | Agent impact |
| --- | --- | --- | --- |
| Managed store | `$SKILLHUB_HOME/installed` and `skillhub.lock` | Skills installed through `skillhub install`; these can be updated and rolled back. | No direct runtime impact. |
| Project discovered | `.skillhub/skills`, `.codex/skills`, `.claude/skills`, `.agents/skills`, `agent/skills` | Skills found in the current project; visible in list/TUI, but not automatically adopted into the managed store. | No direct runtime impact. |
| Runtime copy | `~/.codex/skills`, `~/.claude/skills`, `~/.gemini/skills`, `~/.hermes/skills`, or `~/.hermes/profiles/<profile>/skills` | The copy Codex, Claude, Gemini, or Hermes actually loads. | This affects the agent. |

`skillhub update` updates the managed store only. It does not modify Codex, Claude, Gemini, or Hermes runtime directories. Use `skillhub check` to see whether installed Skills have updates, `skillhub update --preview` (or the compatible `--dry-run`) to inspect changes before writing, `skillhub deploy status` to inspect runtime copies, and `skillhub deploy <runtime> ... --force` only when you intend to replace an existing runtime copy.

## Contents

- [Why skill-hub](#why-skill-hub)
- [Does This Manage My Existing Skills?](#does-this-manage-my-existing-skills)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Command Overview](#command-overview)
- [Catalog Discovery](#catalog-discovery)
- [Static Catalog Export](#static-catalog-export)
- [Install, Update, And Rollback](#install-update-and-rollback)
- [Publish A Skill](#publish-a-skill)
- [Runtime Deploy](#runtime-deploy)
- [Skill Package Format](#skill-package-format)
- [Registry Index Format](#registry-index-format)
- [State Files](#state-files)
- [Local Development](#local-development)
- [Version Policy](#version-policy)
- [Release](#release)
- [Contributing](#contributing)
- [License](#license)

## Why skill-hub

| Capability | What it gives you |
| --- | --- |
| Registry discovery | Find Skills from local or Git-backed registries. |
| Reproducible installs | Track installed versions, checksums, source refs, and deployments in `skillhub.lock`. |
| Safe upgrades | Pin installs to versions, tags, branches, or commits; update and roll back when needed. |
| Runtime deployment | Copy installed Skills into Codex, Claude, Gemini, and Hermes runtime directories. |
| Catalog publishing | Export a static marketplace snapshot as `index.html` and `catalog.json`. |
| Registry validation | Validate indexes before sync or publication. |

## Installation

| Method | Command |
| --- | --- |
| Homebrew | `brew tap CassianFlorin/tap && brew install skillhub` |
| npm | `npm install -g @cassianflorin/skillhub` |
| Go source install | `go install github.com/CassianFlorin/skill-hub/cmd/skillhub@latest` |

Homebrew:

```bash
brew tap CassianFlorin/tap
brew install skillhub
brew upgrade skillhub
```

npm:

```bash
npm install -g @cassianflorin/skillhub
npm update -g @cassianflorin/skillhub
```

Every tagged release also attaches an npm tarball. Use the release URL when you need to pin or mirror a specific release:

```bash
VERSION=1.3.11
npm install -g "https://github.com/CassianFlorin/skill-hub/releases/download/v${VERSION}/cassianflorin-skillhub-${VERSION}.tgz"
```

Go source install:

```bash
go install github.com/CassianFlorin/skill-hub/cmd/skillhub@latest
```

## Quick Start

Initialize a project and sync the default `hub` registry:

```bash
skillhub version
skillhub init
skillhub registry sync hub
```

Discover, inspect, install, and deploy a Skill:

```bash
skillhub catalog featured --registry hub
skillhub catalog list --registry hub --target codex
skillhub search git
skillhub info hub/official/git-commit-cn
skillhub install hub/official/git-commit-cn
skillhub deploy codex official/git-commit-cn --force
skillhub deploy status
skillhub tui
```

To build a local binary without installing it:

```bash
go build -o skillhub ./cmd/skillhub
```

## Command Overview

| Area | Commands |
| --- | --- |
| Project setup | `skillhub help`, `skillhub version`, `skillhub doctor`, `skillhub init` |
| Registries | `skillhub registry add`, `skillhub registry list`, `skillhub registry sync`, `skillhub registry index` |
| Discovery | `skillhub catalog list`, `skillhub catalog featured`, `skillhub catalog tags`, `skillhub search`, `skillhub info` |
| Lifecycle | `skillhub install`, `skillhub list`, `skillhub check`, `skillhub update --preview`, `skillhub hold`, `skillhub holds`, `skillhub unhold`, `skillhub update`, `skillhub history`, `skillhub rollback`, `skillhub uninstall` |
| Runtime deploy | `skillhub deploy codex`, `skillhub deploy claude`, `skillhub deploy gemini`, `skillhub deploy hermes`, `skillhub deploy status` |
| Terminal UI | `skillhub tui` |
| Publication | `skillhub publish`, `skillhub catalog export` |

Common examples:

```bash
skillhub registry add local company ./examples/local-registry
skillhub registry add git team git@github.com:your-org/skills.git
skillhub catalog export --registry hub --output ./public/catalog
skillhub install hub/official/git-commit-cn@v0.1.0
skillhub list --scope all
skillhub list --scope project
```

Help output supports English, Simplified Chinese, and Traditional Chinese:

```bash
skillhub help --lang en
skillhub help --lang zh-CN
skillhub help list --lang zh-TW
SKILLHUB_LANG=zh-CN skillhub help init
```

## Terminal Management

`skillhub tui` opens a terminal management interface for local Skills and synced catalog data. It shows global and project-only Skills side by side, deployment status for Codex, Claude, Gemini, and Hermes, catalog search results, and operation logs.

```bash
skillhub tui
```

The TUI uses a mixed safety model: install, update, registry sync, and normal deploy run directly and write an operation log; uninstall, rollback, force deploy, registry deletion, and project Skill overwrite require confirmation.

## Catalog Discovery

`catalog` reads synced registry indexes and lists installable Skills. Use `registry sync` before discovery when the registry is new or stale.

```bash
./skillhub registry sync hub
./skillhub catalog list --registry hub
./skillhub catalog featured --registry hub
./skillhub catalog list --registry hub --target claude
./skillhub catalog list --registry hub --namespace official --trust official
```

Discovery facets:

```bash
./skillhub catalog tags --registry hub
./skillhub catalog targets --registry hub
./skillhub catalog namespaces --registry hub
./skillhub catalog trust --registry hub
```

`search` ranks stronger matches first: exact or prefix identity/name matches, then tag matches, then description matches. Featured and official Skills are preferred when match strength is otherwise equal.

```bash
./skillhub search git
./skillhub search runtime --json
```

For automation, catalog, search, and info commands support JSON output:

```bash
./skillhub catalog list --registry hub --json
./skillhub catalog featured --registry hub --json
./skillhub catalog tags --registry hub --json
./skillhub catalog targets --registry hub --json
./skillhub info hub/official/git-commit-cn --json
```

## Static Catalog Export

`catalog export` writes a browsable `index.html` and structured `catalog.json` for publishing or reviewing a marketplace snapshot.

```bash
./skillhub catalog export --registry hub --output ./public/catalog
./skillhub catalog export --registry hub --target codex --namespace official --output ./public/codex
```

The exported JSON includes:

- Skills with registry name, metadata, and install command.
- Aggregated tags.
- Aggregated runtime targets.
- Aggregated namespaces.
- Aggregated trust levels.

## Install, Update, And Rollback

Install from a registry:

```bash
./skillhub install hub/official/git-commit-cn
```

Install a local Skill directory:

```bash
./skillhub install ./examples/local-registry/java-review
```

Pinned install examples:

```bash
./skillhub install company/java-review@1.2.0
./skillhub install team/java-review@v1.2.0
./skillhub install team/java-review@main
./skillhub install team/java-review@<commit-sha>
```

For local registries, the pinned value must match the Skill metadata version. For git registries, the pinned value is treated as a git ref and is recorded in `skillhub.lock` together with the resolved commit.

Update awareness and safe preview:

```bash
skillhub check
skillhub update --preview
skillhub hold official/git-commit-cn --reason "this version works best"
skillhub holds
skillhub unhold official/git-commit-cn
skillhub update
```

`skillhub check` reports installed Skills with newer source versions without changing files. `skillhub update --preview` shows the version transition, source, targets, deployed runtimes, and rollback command before any write.

Freeze a Skill that should not be updated yet:

```bash
./skillhub hold platform-team/java-review --reason "1.2.0 works best"
./skillhub holds
./skillhub unhold platform-team/java-review
```

Held Skills still appear in `skillhub check` and `skillhub update --preview` with policy `held`, but `skillhub update` skips them until `unhold` is run.

Update and rollback:

```bash
./skillhub update platform-team/java-review
./skillhub update
./skillhub history platform-team/java-review
./skillhub rollback platform-team/java-review --to 1.2.0
./skillhub rollback platform-team/java-review --to 1.2.0 --deploy hermes --profile work
```

For git registries, `skillhub update` refreshes the cached repository with `git pull --ff-only` and updates installed Skills when their `skill.yaml` version changes. `skillhub history <identity>` lists rollback snapshots saved before updates or reinstalls. `skillhub rollback <identity>` restores the latest previous installed copy, while `--to <version>` selects a specific history version. Runtime copies are not changed by update or rollback unless you pass `--deploy <runtime>`; for example, `skillhub rollback platform-team/java-review --to 1.2.0 --deploy hermes --profile work` restores the managed copy and immediately redeploys the Hermes profile copy with force semantics.

Uninstall:

```bash
./skillhub uninstall platform-team/java-review
./skillhub uninstall platform-team/java-review --deployed
```

By default, uninstall removes the installed store copy and lockfile entry only. Use `--deployed` to also remove Codex, Claude, Gemini, and Hermes runtime copies.

## Publish A Skill

`skillhub publish` validates a local Skill package, copies it into a registry, and updates `skillhub.index.json` in one step:

```bash
skillhub publish ./skills/git-commit-cn --registry company --dry-run
skillhub publish ./skills/git-commit-cn --registry company
skillhub publish ./skills/git-commit-cn --registry team --branch publish/git-commit-cn
skillhub publish ./skills/git-commit-cn --registry hub --pr
```

Before writing anything, publish checks that `skill.yaml` declares an explicit `version`, a `description`, at least one supported target, and a `namespace` or `author`, then computes the package checksum. Republishing the same version with different content is rejected, and semver versions lower than the published entry are rejected; bump the version instead.

- Local registries are updated in place.
- Git registries are cloned to a temporary workspace, committed, and pushed. Use `--branch` to push a review branch when you have write access, or `--pr` to fork the registry (when needed) and open a pull request through the `gh` CLI.
- `--dry-run` prints the index entry diff without writing.
- Updates preserve existing `trust`, `featured`, and `license` review metadata; override trust explicitly with `--trust`.

To publish into the public catalog, run `skillhub publish --pr` against the hub registry (requires an authenticated `gh` CLI), or follow the manual pull-request workflow in [skill-hub-registry](https://github.com/CassianFlorin/skill-hub-registry).

## Runtime Deploy

Supported runtime targets:

| Runtime | Env var | Default directory |
| --- | --- | --- |
| Codex | `SKILLHUB_CODEX_DIR` | `~/.codex/skills` |
| Claude | `SKILLHUB_CLAUDE_DIR` | `~/.claude/skills` |
| Gemini | `SKILLHUB_GEMINI_DIR` | `~/.gemini/skills` |
| Hermes | `SKILLHUB_HERMES_DIR` | `~/.hermes/skills` |

Deploy examples:

```bash
./skillhub deploy codex platform-team/java-review --dry-run
./skillhub deploy codex platform-team/java-review --force
./skillhub deploy claude platform-team/java-review --force
./skillhub deploy gemini platform-team/java-review --force
./skillhub deploy hermes platform-team/java-review --force
./skillhub deploy hermes platform-team/java-review --profile work --force
```

For Hermes, `--profile <name>` deploys to `~/.hermes/profiles/<name>/skills`. Set `SKILLHUB_HERMES_HOME` to override the Hermes home root used for profile deployments.

Status:

```bash
./skillhub deploy status
./skillhub deploy status codex
./skillhub deploy status claude
./skillhub deploy status gemini
./skillhub deploy status hermes
```

Deploy respects installed Skill `targets` metadata. Skills with no targets are treated as compatible with all supported runtimes for backward compatibility. In batch deploys, unsupported Skills are skipped; explicitly deploying an unsupported Skill to a runtime returns an error.

Without `--force`, deploy preflights all selected Skills before copying files. If any selected target already exists, the command fails before partial writes. Use `--force` to replace existing runtime copies.

Deploy status values:

- `deployed`: runtime copy matches the installed Skill checksum.
- `missing`: the Skill supports the runtime but has not been deployed there.
- `drifted`: runtime copy exists but differs from the installed Skill checksum.
- `unsupported`: the Skill does not list that runtime in `targets`.

## Skill Package Format

Recommended structure:

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
  - claude
  - gemini
  - hermes
tags:
  - java
  - review
author: platform-team
```

Existing Skill directories that only contain `SKILL.md` can still be installed. skill-hub writes a generated `skill.yaml` into the installed copy so the lockfile and deploy pipeline can use the same metadata model.

Installed Skill identities are displayed as `namespace/name`, using this priority:

1. `skill.yaml.namespace`
2. `skill.yaml.author`
3. registry name
4. local user name
5. `unknown`

## Registry Index Format

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
      "targets": ["codex", "claude", "gemini", "hermes"],
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
        "reviewed_at": "2026-05-27",
        "reviewer": "CassianFlorin"
      },
      "featured": true,
      "updated_at": "2026-05-27"
    }
  ]
}
```

Validate a registry:

```bash
./skillhub registry index validate hub
```

Generate an index for a local registry:

```bash
./skillhub registry index generate company
```

## State Files

- Project config: `skillhub.yaml`
- Runtime home: `$SKILLHUB_HOME`, defaulting to `~/.skillhub`
- Git registry cache: `$SKILLHUB_HOME/cache/registries/<registry-name>`
- Installed Skills: `$SKILLHUB_HOME/installed/<safe-identity>`
- Lockfile: `$SKILLHUB_HOME/skillhub.lock`
- Project-only Skill roots discovered by `skillhub list`: `.skillhub/skills`, `.codex/skills`, `.claude/skills`, `.agents/skills`, and `agent/skills`
- Runtime copies: configured by the runtime env vars listed above

`skillhub.yaml` and `skillhub.lock` are JSON documents with YAML-compatible filenames. This keeps the CLI dependency-light while preserving the intended file names.

## Local Development

Run the test suite and build the CLI:

```bash
go test ./...
npm test --prefix npm
go build ./cmd/skillhub
```

Run without touching real user state:

```bash
export SKILLHUB_HOME="$PWD/.skillhub-e2e/home"
export SKILLHUB_CODEX_DIR="$PWD/.skillhub-e2e/codex"
export SKILLHUB_CLAUDE_DIR="$PWD/.skillhub-e2e/claude"
export SKILLHUB_GEMINI_DIR="$PWD/.skillhub-e2e/gemini"

./skillhub init
./skillhub doctor
./skillhub registry add local company "$PWD/examples/local-registry"
./skillhub registry sync company
./skillhub catalog list --registry company
./skillhub install company/java-review
./skillhub deploy codex platform-team/java-review --dry-run
```

## Version Policy

`v1.3.x` is the hardening line for the current public installer flow. Patch releases in this line should keep tightening installation, update, rollback, and release reliability without changing the package model.

Use the next available `v1.3.x` patch tag for installer and CLI hardening. Keep `v1.4.0` reserved for the first broader team rollout once the `v1.3.x` CLI and installer experience is stable enough.

## Release

CI runs `go test ./...`, `npm test --prefix npm`, and `go build -v ./cmd/skillhub` on pushes and pull requests.

Tagged releases publish multi-platform archives through GitHub Actions. The same release workflow can also publish:

- Homebrew formula updates to `CassianFlorin/homebrew-tap` when `HOMEBREW_TAP_TOKEN` is configured with write access to that tap repository.
- The npm package `@cassianflorin/skillhub` when `NPM_TOKEN` is configured with publish access to the npm scope.
- An npm tarball attached to the GitHub Release, so users can still install with npm from the release URL if npm registry publishing is not configured.
- `Formula/skillhub.rb` in this repository can be refreshed with `scripts/generate-homebrew-formula.sh` when cutting a release.

```bash
NEXT_TAG=v1.3.11
git tag -a "${NEXT_TAG}" -m "${NEXT_TAG}"
git push origin "${NEXT_TAG}"
```

The npm package version is set from the Git tag during release. The package downloads the matching GitHub Release archive during `postinstall` and verifies it against `checksums.txt`.

Homebrew requires formulae to live in a tap. The checked-in `Formula/skillhub.rb` mirrors the latest release formula and is the source used for the `CassianFlorin/homebrew-tap` publication path.

If release assets already exist and only npm or Homebrew publishing needs to be retried after configuring secrets, run the `Publish Installers` workflow manually with the existing tag, such as `v1.3.11`.

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and pull request guidelines. To publish a Skill to the public catalog, see [skill-hub-registry](https://github.com/CassianFlorin/skill-hub-registry).

Security issues should be reported privately — see [SECURITY.md](SECURITY.md).

## License

skill-hub is released under the [MIT License](LICENSE).
