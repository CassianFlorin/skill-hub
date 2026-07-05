# Contributing to skill-hub

Thanks for your interest in improving skill-hub! This document covers how to build, test, and submit changes to the CLI. To contribute a **Skill** to the public catalog instead, see [skill-hub-registry](https://github.com/CassianFlorin/skill-hub-registry).

## Development Setup

Requirements:

- Go 1.25+
- Node.js 18+ (only for the npm installer package tests)
- git

Build and test:

```bash
go build ./cmd/skillhub
go test ./...
npm test --prefix npm
```

Run the CLI against isolated state so you never touch your real Skill directories:

```bash
export SKILLHUB_HOME="$PWD/.skillhub-e2e/home"
export SKILLHUB_CODEX_DIR="$PWD/.skillhub-e2e/codex"
export SKILLHUB_CLAUDE_DIR="$PWD/.skillhub-e2e/claude"
export SKILLHUB_GEMINI_DIR="$PWD/.skillhub-e2e/gemini"

./skillhub init
./skillhub doctor
./skillhub registry add local company "$PWD/examples/local-registry"
./skillhub registry sync company
./skillhub install company/java-review
```

## Making Changes

1. Fork the repository and create a branch from `master`.
2. Keep changes focused: one feature or fix per pull request.
3. Add or update tests for any behavior change. CI enforces `go vet`, the full test suite with `-race`, and a cross-platform build matrix.
4. Match the existing code style; run `gofmt` before committing.
5. If you change user-facing behavior, update `README.md` (and the zh-CN / zh-TW translations if you can — otherwise note it in the PR and a maintainer will follow up).

Commit messages follow the conventional style used in the history, for example:

```
feat(deploy): add windows runtime path support
fix(registry): tolerate missing updated_at in index v2
docs: clarify rollback semantics
```

## Pull Requests

- Describe what changed and why, plus how you verified it.
- Link the related issue when one exists.
- CI must pass before review.

## Reporting Bugs and Requesting Features

Use the [issue templates](https://github.com/CassianFlorin/skill-hub/issues/new/choose). For bugs, include your OS, `skillhub version` output, and the exact command plus output that failed. `skillhub doctor` output is often useful.

## Security Issues

Please do not open public issues for security vulnerabilities. See [SECURITY.md](SECURITY.md).

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
