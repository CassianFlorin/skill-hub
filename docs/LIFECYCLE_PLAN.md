# Skill Lifecycle Management Implementation Plan

> **For Hermes:** Use subagent-driven-development skill to implement this plan task-by-task.

**Goal:** Turn skill-hub from a basic install/deploy CLI into a user-friendly Skill lifecycle manager: discover updates, preview changes, safely update, pin/hold versions, and rollback when a Skill gets worse.

**Architecture:** Keep `skillhub.lock` as the local source of truth, but enrich it with update policy and history visibility. Add lifecycle-oriented commands that compose the existing registry, install, history, and deploy modules instead of forcing users to manually remember `install → update → deploy → rollback` steps.

**Tech Stack:** Go CLI, existing `internal/install`, `internal/registry`, `internal/deploy`, JSON lock/history files, Git metadata from cached registries.

---

## Product Target

The intended user experience is:

```bash
# See what changed upstream before touching local runtime copies
skillhub check

# Preview an update with old/new version, source commit, changed files, and deploy impact
skillhub update --preview
skillhub diff official/git-commit-cn

# Upgrade safely; previous copy is snapshotted automatically
skillhub update official/git-commit-cn

# If the new behavior is worse, inspect history and go back
skillhub history official/git-commit-cn
skillhub rollback official/git-commit-cn --deploy hermes --profile default

# If a skill should stay stable, hold it
skillhub hold official/git-commit-cn
skillhub update
# held skill is skipped
skillhub unhold official/git-commit-cn
```

For convenience, add a one-command safe path:

```bash
skillhub upgrade hermes --profile default
```

This should:
1. check available updates,
2. show a summary,
3. update managed store,
4. preserve rollback history,
5. deploy changed skills to Hermes,
6. print exact rollback commands.

---

## Existing State

Current code already has important foundations:

- `install.Install` snapshots the previous installed copy with `saveHistory` when replacing an installed Skill.
- `install.UpdateAll` updates installed copies and snapshots history before copying the new version.
- `install.PlanUpdates` supports `skillhub update --dry-run`, but only returns `[identity, oldVersion, newVersion]`.
- `install.Rollback` restores the latest historical snapshot.
- `deploy.DeployRuntime` can deploy to `codex`, `claude`, `gemini`, and now `hermes`.

Gaps from the user's perspective:

- No clear `check` command that says “which Skills have updates?”
- `update --dry-run` is too shallow; it does not show source commit, changed files, changelog, or whether runtime copies will change.
- No way to hold/pin a Skill against updates after finding a good version.
- Rollback exists, but there is no `history` listing, no target snapshot choice, and no automatic redeploy after rollback.
- Users must understand internal separation between managed store and runtime deployment.
- No ergonomic “safe update then deploy to Hermes” workflow.

---

## UX Principles

1. **Update awareness before mutation**
   - Users should know that updates exist before applying them.
   - Default commands should be safe and explanatory.

2. **Preview before upgrade**
   - Show version, source commit, changed files, and runtime impact.
   - Prefer `--preview` wording over `--dry-run` for normal users, while keeping `--dry-run` as alias.

3. **Rollback must be obvious**
   - Every update output should print the rollback command.
   - `rollback` should optionally redeploy to a runtime in the same command.

4. **Stable Skills should stay stable**
   - Add hold/unhold/list-held semantics.
   - Held Skills are skipped by default but can be updated with `--include-held` or `--force`.

5. **Hermes users should not need package-manager mental models**
   - Provide `skillhub upgrade hermes --profile <name>` as the friendly high-level path.

---

## Proposed Commands

### `skillhub check`

Purpose: tell the user whether installed Skills have updates.

Examples:

```bash
skillhub check
skillhub check --json
skillhub check --include-held
```

Human output:

```text
Updates available:

Skill                    Current  Available  Policy  Source
official/git-commit-cn   1.0.0    1.1.0      update  hub@abc1234
company/sql-review       0.4.2    0.5.0      held    company@def5678

Run `skillhub update --preview` to inspect changes.
Run `skillhub update <skill>` to update one Skill.
```

Exit codes:
- `0`: no updates
- `2`: updates available
- `1`: error

### `skillhub update --preview [identity]`

Purpose: richer preview than current `--dry-run`.

Output should include:
- identity
- current version
- available version
- current source commit
- available source commit
- held/pinned status
- changed file count and top changed files when available
- deploy impact: which runtimes are currently deployed and will become drifted after update

Keep backward compatibility:

```bash
skillhub update --dry-run
```

Alias it to preview.

### `skillhub diff <identity>`

Purpose: show actual file-level diff between installed copy and available update.

Minimum viable implementation:
- Resolve available source using the same logic as update planning.
- Compare installed path vs resolved source path.
- Print unified diff for text files, and binary/change summaries for non-text files.

Examples:

```bash
skillhub diff official/git-commit-cn
skillhub diff official/git-commit-cn --stat
```

### `skillhub update [identity]`

Extend current update command:
- Allow updating one specific Skill.
- Skip held Skills unless `--force` or `--include-held`.
- Print rollback instructions.
- Print deploy instructions when runtime copies exist.

Output:

```text
updated official/git-commit-cn 1.0.0 -> 1.1.0
rollback: skillhub rollback official/git-commit-cn
runtime copies were not changed; run:
  skillhub deploy hermes official/git-commit-cn --profile default --force
```

### `skillhub hold <identity>` / `skillhub unhold <identity>` / `skillhub holds`

Purpose: let users freeze a Skill at a known-good version.

Lockfile extension:

```json
{
  "identity": "official/git-commit-cn",
  "version": "1.0.0",
  "update_policy": "hold",
  "held_at": "2026-06-22T...Z",
  "hold_reason": "newer prompt produced worse commits"
}
```

CLI examples:

```bash
skillhub hold official/git-commit-cn
skillhub hold official/git-commit-cn --reason "v1.1 is too verbose"
skillhub holds
skillhub unhold official/git-commit-cn
```

### `skillhub history <identity>`

Purpose: show snapshots available for rollback.

Output:

```text
History for official/git-commit-cn:

Index  Version  Updated At               Source Commit  Checksum
0      1.0.0    2026-06-20T10:00:00Z     abc1234        sha256:...
1      0.9.0    2026-06-15T08:00:00Z     9999999        sha256:...

Rollback latest:
  skillhub rollback official/git-commit-cn
Rollback specific snapshot:
  skillhub rollback official/git-commit-cn --to 1
```

### `skillhub rollback <identity> [--to <index|version>] [--deploy <runtime>] [--profile <name>]`

Extend current rollback:
- Support selecting a snapshot by index or version.
- Support redeploy after rollback.
- Print what changed.

Examples:

```bash
skillhub rollback official/git-commit-cn
skillhub rollback official/git-commit-cn --to 1.0.0
skillhub rollback official/git-commit-cn --deploy hermes --profile default
```

### `skillhub upgrade <runtime> [identity] [--profile <name>] [--preview]`

Purpose: convenient safe workflow for normal users.

For Hermes:

```bash
skillhub upgrade hermes --profile default --preview
skillhub upgrade hermes --profile default
```

Behavior:
- preview mode shows updates + deploy impact.
- real mode updates managed store and deploys changed supported Skills to runtime.
- held Skills skipped.
- output includes rollback commands.

---

## Task 1: Add richer update planning model

**Objective:** Replace `[3]string` update change tuples with typed update plan structs while preserving CLI output compatibility.

**Files:**
- Modify: `internal/install/install.go`
- Modify: `internal/cli/cli.go`
- Test: `internal/cli/cli_test.go`

**Implementation sketch:**

```go
type UpdatePlan struct {
    Identity string
    CurrentVersion string
    AvailableVersion string
    CurrentCommit string
    AvailableCommit string
    SourceRegistry string
    SourceURL string
    SourcePath string
    AvailablePath string
    Targets []string
    DeployedTo []string
    Held bool
    HoldReason string
}
```

Add:

```go
func PlanUpdateDetails(identity string, opts UpdateOptions) ([]UpdatePlan, error)
```

Keep:

```go
func PlanUpdates() ([][3]string, error)
func UpdateAll() ([][3]string, error)
```

as compatibility wrappers until CLI is migrated.

**Verification:**

```bash
go test ./internal/install ./internal/cli
```

---

## Task 2: Implement `skillhub check`

**Objective:** Add a read-only update awareness command.

**Files:**
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/cli_test.go`

**Tests:**
- No installed skills → friendly no updates message.
- Installed skill with newer version → table row appears.
- Held skill → shown as held or skipped depending on flag.
- `--json` emits machine-readable plan.

---

## Task 3: Add hold/unhold/holds

**Objective:** Let users freeze known-good Skill versions.

**Files:**
- Modify: `internal/install/install.go`
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/cli_test.go`

**Lockfile additions:**

```go
UpdatePolicy string `json:"update_policy,omitempty"`
HeldAt string `json:"held_at,omitempty"`
HoldReason string `json:"hold_reason,omitempty"`
```

**Commands:**

```bash
skillhub hold <identity> [--reason <text>]
skillhub unhold <identity>
skillhub holds
```

**Verification:**

```bash
go test ./internal/install ./internal/cli
```

---

## Task 4: Upgrade `update --preview` and single-skill update

**Objective:** Make update safe and understandable.

**Files:**
- Modify: `internal/install/install.go`
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/cli_test.go`

**Behavior:**
- `skillhub update --preview` shows detailed plan.
- `skillhub update --dry-run` remains alias.
- `skillhub update <identity>` updates one Skill.
- held Skills skipped unless `--force`.
- output includes rollback/deploy next steps.

---

## Task 5: Add history listing and selectable rollback

**Objective:** Make rollback transparent instead of hidden magic.

**Files:**
- Modify: `internal/install/install.go`
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/cli_test.go`

**Implementation:**
- Export history listing helper.
- Add `RollbackTo(identity, selector)`.
- Keep existing `Rollback(identity)` as latest-history wrapper.

**Commands:**

```bash
skillhub history <identity>
skillhub rollback <identity> --to <version-or-index>
```

---

## Task 6: Add rollback redeploy support

**Objective:** Let users recover actual runtime behavior in one command.

**Files:**
- Modify: `internal/cli/cli.go`
- Possibly modify: `internal/deploy/codex.go`
- Test: `internal/cli/cli_test.go`

**Command:**

```bash
skillhub rollback official/git-commit-cn --deploy hermes --profile default
```

**Verification:**
- Update Skill.
- Deploy to temp Hermes profile.
- Rollback with `--deploy hermes`.
- Assert profile skill files match rolled-back checksum.

---

## Task 7: Add `skillhub diff <identity>`

**Objective:** Let users see what changed before upgrading.

**Files:**
- Create: `internal/diff/diff.go` or add scoped helper under `internal/install`
- Modify: `internal/cli/cli.go`
- Test: `internal/cli/cli_test.go`

**Minimum implementation:**
- `--stat` first.
- Unified diff for small text files.
- Limit output size with clear truncation message.

---

## Task 8: Add `skillhub upgrade <runtime>` workflow

**Objective:** Provide a single friendly command for normal Hermes/Codex/Claude/Gemini users.

**Files:**
- Modify: `internal/cli/cli.go`
- Test: `internal/cli/cli_test.go`

**Command:**

```bash
skillhub upgrade hermes --profile default --preview
skillhub upgrade hermes --profile default
```

**Behavior:**
- Preview: show update plan + deploy impact.
- Real: update managed store, then deploy changed Skills to runtime with `Force: true`.
- Print rollback commands for every updated Skill.

---

## Task 9: Documentation and examples

**Objective:** Explain the lifecycle workflow to normal users.

**Files:**
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `README.zh-TW.md`
- Modify: `docs/ARCHITECTURE.md`
- Modify: `docs/ROADMAP.md`

Add a “Safe Skill Updates” section:

```bash
skillhub check
skillhub update --preview
skillhub diff <identity>
skillhub update <identity>
skillhub deploy hermes <identity> --profile default --force
skillhub rollback <identity> --deploy hermes --profile default
skillhub hold <identity>
```

---

## Acceptance Criteria

A normal Hermes user should be able to answer these questions without reading source code:

1. Do any of my installed Skills have updates?
2. What exactly will change if I update?
3. Which Skills are deployed into my Hermes profile?
4. Can I update only one Skill?
5. Can I freeze a Skill that currently works well?
6. If a new version is worse, what versions can I roll back to?
7. Can I roll back and redeploy to Hermes in one command?
8. Is there one safe high-level command for “update my Hermes Skills”? 

The feature is complete only when all answers are covered by CLI commands, tests, and README examples.
