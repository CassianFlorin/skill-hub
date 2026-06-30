# SkillHub.app — native macOS front-end

A native SwiftUI desktop app for `skillhub`. It bundles the `skillhub` CLI
binary inside the `.app` and drives it by shelling out and parsing the CLI's
`--json` output — so the GUI always behaves exactly like the command line.

## Features (v1)

- **Catalog** — browse and search skills across every configured registry, view
  full skill details, and install with one click.
- **Installed** — list managed (global) and project skills; uninstall (optionally
  removing runtime copies), hold/unhold, and roll back.
- **Updates** — check for available updates and apply them per-skill or all at once.
- **Deploy** — deploy managed skills into the Codex / Claude / Gemini / Hermes
  runtime directories and inspect deploy status.
- **Doctor** — local config, runtime directories, registries, and install counts.

## Requirements

- macOS 13 or later
- Xcode 15+ (for the Swift toolchain) and Go 1.25+ (to build the bundled CLI)

## Build

```bash
cd macos
./build.sh            # universal (arm64 + x86_64) release build
./build.sh --arch     # host-architecture only (faster, for local dev)
open build/SkillHub.app
```

`build.sh` compiles the Go CLI (universal via `lipo`), builds the SwiftUI
front-end with `swift build`, assembles `build/SkillHub.app`, and ad-hoc
codesigns it so it launches locally.

## How it locates the CLI

`SkillHubCLI` resolves the binary in this order:

1. Bundled at `SkillHub.app/Contents/Resources/skillhub` (the normal case).
2. `$SKILLHUB_BIN` — an explicit path, handy for `swift run` during development.
3. A `skillhub` binary next to the repo, or anything on `PATH`.

## Working directory & state

The app runs the CLI inside `~/Library/Application Support/SkillHub`, so its
registry config (`skillhub.yaml`) stays out of your home directory. The managed
store, lockfile, and rollback history live where the CLI always keeps them,
under `$SKILLHUB_HOME` (default `~/.skillhub`). Runtime deploy targets are the
usual `~/.codex/skills`, `~/.claude/skills`, etc.

## Develop

```bash
cd macos
SKILLHUB_BIN="$(cd .. && go build -o /tmp/skillhub ./cmd/skillhub && echo /tmp/skillhub)" \
  swift run
```

## Architecture

```
Sources/SkillHub/
  SkillHubApp.swift        # @main App + window scene
  CLI/Models.swift         # Codable mirrors of the CLI --json shapes
  CLI/SkillHubCLI.swift     # locates + runs the binary, decodes JSON
  Store/AppStore.swift      # @MainActor ObservableObject: state + actions
  Views/                    # RootView + one view per sidebar section
```

The CLI grew `--json` output for `list`, `doctor`, `check`, `deploy status`,
`registry list`, `holds`, and `history` to support this GUI (the catalog,
`search`, and `info` commands already had it).
