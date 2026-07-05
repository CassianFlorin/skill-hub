# skill-hub

[English](README.md) · [简体中文](README.zh-CN.md) · [繁體中文](README.zh-TW.md)

`skill-hub` 是面向 AI Agent 的 Skill 包管理器。它把 Skill 当作可安装的软件包来管理，支持元数据、注册表索引、锁文件状态、回滚历史和运行时部署目标。

当前发布线：`v1.3.x`。

## 功能概览

- 从本地或 Git 注册表发现 Skills。
- 将 Skills 安装到 `$SKILLHUB_HOME` 下的托管本地目录。
- 在 `skillhub.lock` 中记录已安装状态，包括版本、校验和、来源 ref 和已部署运行时。
- 支持按元数据版本、Git tag、分支或 commit 固定安装。
- 将已安装 Skills 部署到 Codex、Claude、Gemini 和 Hermes 的运行时目录。
- 导出静态目录快照为 `index.html` 和 `catalog.json`。
- 在同步或发布前校验注册表索引。

## 安装

使用 Homebrew 安装：

```bash
brew tap CassianFlorin/tap
brew install skillhub
brew upgrade skillhub
```

使用 npm 安装：

```bash
npm install -g @cassianflorin/skillhub
npm update -g @cassianflorin/skillhub
```

每个 tagged release 也会附带 npm tarball。如果你需要固定或镜像某个具体版本，可以使用 release URL：

```bash
VERSION=1.3.0
npm install -g "https://github.com/CassianFlorin/skill-hub/releases/download/v${VERSION}/cassianflorin-skillhub-${VERSION}.tgz"
```

开发者也可以直接从源码安装：

```bash
go install github.com/CassianFlorin/skill-hub/cmd/skillhub@latest
```

## 快速开始

检查 CLI：

```bash
skillhub version
```

初始化项目。这会添加默认的 `hub` 注册表，指向 `https://github.com/CassianFlorin/skill-hub-registry.git`。

```bash
skillhub init
skillhub registry sync hub
```

发现和查看 Skills：

```bash
skillhub catalog featured --registry hub
skillhub catalog list --registry hub --target codex
skillhub search git
skillhub info hub/official/git-commit-cn
```

安装并部署：

```bash
skillhub install hub/official/git-commit-cn
skillhub deploy codex official/git-commit-cn --force
skillhub deploy status
```

`skillhub help` 支持英语、简体中文和繁体中文：

```bash
skillhub help --lang en
skillhub help --lang zh-CN
skillhub help list --lang zh-TW
SKILLHUB_LANG=zh-CN skillhub help init
```

如果只想构建本地二进制而不安装：

```bash
go build -o skillhub ./cmd/skillhub
```

## 命令概览

```bash
skillhub version
skillhub help
skillhub doctor
skillhub init

skillhub registry add local company ./examples/local-registry
skillhub registry add git team git@github.com:your-org/skills.git
skillhub registry list
skillhub registry sync hub
skillhub registry sync --all
skillhub registry index generate company
skillhub registry index validate company

skillhub catalog list --registry hub
skillhub catalog featured --registry hub
skillhub catalog tags --registry hub
skillhub catalog targets --registry hub
skillhub catalog namespaces --registry hub
skillhub catalog trust --registry hub
skillhub catalog export --registry hub --output ./public/catalog

skillhub search java
skillhub info hub/official/git-commit-cn
skillhub install hub/official/git-commit-cn
skillhub install hub/official/git-commit-cn@v0.1.0
skillhub list
skillhub list --scope all
skillhub list --scope project
skillhub check
skillhub update --preview
skillhub hold official/git-commit-cn --reason "这个版本效果最好"
skillhub holds
skillhub unhold official/git-commit-cn
skillhub update
skillhub history official/git-commit-cn
skillhub rollback official/git-commit-cn --to 0.1.0
skillhub rollback official/git-commit-cn --to 0.1.0 --deploy hermes --profile work
skillhub uninstall official/git-commit-cn
skillhub tui

skillhub deploy codex
skillhub deploy claude
skillhub deploy gemini
skillhub deploy hermes
skillhub deploy status
```

## 终端图形化管理

`skillhub tui` 会打开终端图形化管理界面，用于管理本机 Skills 和已同步的目录数据。它会并排展示全局 Skills、项目内 Skills、Codex/Claude/Gemini 部署状态、Catalog 搜索结果和操作日志。

```bash
skillhub tui
```

TUI 使用混合安全模式：install、update、registry sync 和普通 deploy 直接执行并记录操作日志；uninstall、rollback、force deploy、删除 registry、覆盖项目 Skill 需要二次确认。

## Catalog 发现

`catalog` 会读取已同步的注册表索引并列出可安装的 Skills。新注册表或索引可能过期时，请先执行 `registry sync`。

```bash
skillhub registry sync hub
skillhub catalog list --registry hub
skillhub catalog featured --registry hub
skillhub catalog list --registry hub --target claude
skillhub catalog list --registry hub --namespace official --trust official
```

可用的发现维度：

```bash
skillhub catalog tags --registry hub
skillhub catalog targets --registry hub
skillhub catalog namespaces --registry hub
skillhub catalog trust --registry hub
```

`search` 会优先返回更强匹配：identity/name 精确或前缀匹配优先，其次是 tag 匹配，最后是 description 匹配。在匹配强度相同时，featured 和 official Skills 会排在更前。

```bash
skillhub search git
skillhub search runtime --json
```

面向自动化场景，catalog、search 和 info 命令支持 JSON 输出：

```bash
skillhub catalog list --registry hub --json
skillhub catalog featured --registry hub --json
skillhub catalog tags --registry hub --json
skillhub catalog targets --registry hub --json
skillhub info hub/official/git-commit-cn --json
```

## 静态目录导出

`catalog export` 会写出可浏览的 `index.html` 和结构化的 `catalog.json`，用于发布或审核 marketplace 快照。

```bash
skillhub catalog export --registry hub --output ./public/catalog
skillhub catalog export --registry hub --target codex --namespace official --output ./public/codex
```

导出的 JSON 包含：

- Skills 及其注册表名、元数据和安装命令。
- 聚合后的 tags。
- 聚合后的运行时 targets。
- 聚合后的 namespaces。
- 聚合后的 trust levels。

## 安装、更新和回滚

从注册表安装：

```bash
skillhub install hub/official/git-commit-cn
```

安装本地 Skill 目录：

```bash
skillhub install ./examples/local-registry/java-review
```

固定版本安装示例：

```bash
skillhub install company/java-review@1.2.0
skillhub install team/java-review@v1.2.0
skillhub install team/java-review@main
skillhub install team/java-review@<commit-sha>
```

对于本地注册表，固定值必须匹配 Skill 元数据版本。对于 Git 注册表，固定值会被视为 Git ref，并和解析出的 commit 一起记录到 `skillhub.lock`。

更新感知和安全预览：

```bash
skillhub check
skillhub update --preview
skillhub update --dry-run   # 兼容旧用法，等同于 --preview
```

`skillhub check` 只检查已安装 Skill 是否有新的来源版本，不会修改任何文件。`skillhub update --preview` 会在写入前展示版本变化、来源、targets、已部署运行时和回滚命令。

冻结暂时不想更新的 Skill：

```bash
skillhub hold platform-team/java-review --reason "1.2.0 效果最好"
skillhub holds
skillhub unhold platform-team/java-review
```

被 hold 的 Skills 仍会出现在 `skillhub check` 和 `skillhub update --preview` 中，policy 显示为 `held`，但 `skillhub update` 会跳过它们，直到执行 `unhold`。

更新和回滚：

```bash
skillhub update platform-team/java-review
skillhub update
skillhub history platform-team/java-review
skillhub rollback platform-team/java-review --to 1.2.0
skillhub rollback platform-team/java-review --to 1.2.0 --deploy hermes --profile work
```

对于 Git 注册表，`skillhub update` 会通过 `git pull --ff-only` 刷新缓存仓库，并在 `skill.yaml` 版本变化时更新已安装 Skills。`skillhub history <identity>` 会列出更新或重新安装前保存的回滚快照。`skillhub rollback <identity>` 默认恢复最近一次历史安装副本，`--to <version>` 可以选择指定历史版本。update 和 rollback 默认都不会修改运行时副本；如果传入 `--deploy <runtime>`，例如 `skillhub rollback platform-team/java-review --to 1.2.0 --deploy hermes --profile work`，会恢复托管副本并立即用覆盖语义重新部署 Hermes profile 副本。

卸载：

```bash
skillhub uninstall platform-team/java-review
skillhub uninstall platform-team/java-review --deployed
```

默认情况下，卸载只会移除已安装存储副本和锁文件记录。使用 `--deployed` 可以同时移除 Codex、Claude、Gemini 和 Hermes 运行时副本。

## 运行时部署

支持的运行时目标：

| Runtime | 环境变量 | 默认目录 |
| --- | --- | --- |
| Codex | `SKILLHUB_CODEX_DIR` | `~/.codex/skills` |
| Claude | `SKILLHUB_CLAUDE_DIR` | `~/.claude/skills` |
| Gemini | `SKILLHUB_GEMINI_DIR` | `~/.gemini/skills` |
| Hermes | `SKILLHUB_HERMES_DIR` | `~/.hermes/skills` |

部署示例：

```bash
skillhub deploy codex platform-team/java-review --dry-run
skillhub deploy codex platform-team/java-review --force
skillhub deploy claude platform-team/java-review --force
skillhub deploy gemini platform-team/java-review --force
skillhub deploy hermes platform-team/java-review --force
skillhub deploy hermes platform-team/java-review --profile work --force
```

Hermes 支持 `--profile <name>`，会部署到 `~/.hermes/profiles/<name>/skills`。如需覆盖 profile 根目录，可设置 `SKILLHUB_HERMES_HOME`。

查看状态：

```bash
skillhub deploy status
skillhub deploy status codex
skillhub deploy status claude
skillhub deploy status gemini
skillhub deploy status hermes
```

部署会遵守已安装 Skill 的 `targets` 元数据。没有声明 targets 的 Skills 会被视为兼容所有支持的运行时，以便向后兼容。批量部署时，不兼容的 Skills 会被跳过；如果显式把不兼容的 Skill 部署到某个运行时，命令会返回错误。

没有 `--force` 时，deploy 会在复制前预检所有选中的 Skills。如果任意目标已存在，命令会在产生部分写入前失败。使用 `--force` 可以替换已有运行时副本。

部署状态：

- `deployed`：运行时副本与已安装 Skill 校验和一致。
- `missing`：该 Skill 支持运行时，但尚未部署。
- `drifted`：运行时副本存在，但与已安装 Skill 校验和不同。
- `unsupported`：该 Skill 的 targets 不包含该运行时。

## Skill 包格式

推荐结构：

```text
java-review/
├── skill.yaml
├── SKILL.md
├── references/
├── scripts/
└── assets/
```

最小 `skill.yaml`：

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

只包含 `SKILL.md` 的既有 Skill 目录仍然可以安装。skill-hub 会在已安装副本中写入生成的 `skill.yaml`，这样锁文件和部署流程可以使用同一套元数据模型。

已安装 Skill identity 的显示格式为 `namespace/name`，namespace 按以下优先级确定：

1. `skill.yaml.namespace`
2. `skill.yaml.author`
3. 注册表名称
4. 本地用户名
5. `unknown`

## 注册表索引格式

注册表使用 `skillhub.index.json` schema v2。没有 `schema_version: "2"` 的旧索引文件会在校验时被拒绝。

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

校验注册表：

```bash
skillhub registry index validate hub
```

为本地注册表生成索引：

```bash
skillhub registry index generate company
```

## 状态文件

- 项目配置：`skillhub.yaml`
- 运行时 home：`$SKILLHUB_HOME`，默认 `~/.skillhub`
- Git 注册表缓存：`$SKILLHUB_HOME/cache/registries/<registry-name>`
- 已安装 Skills：`$SKILLHUB_HOME/installed/<safe-identity>`
- 锁文件：`$SKILLHUB_HOME/skillhub.lock`
- `skillhub list` 会发现的项目内 Skill 根目录：`.skillhub/skills`、`.codex/skills`、`.claude/skills`、`.agents/skills`、`agent/skills`
- 运行时副本：由上方运行时环境变量配置

`skillhub.yaml` 和 `skillhub.lock` 是带 YAML 兼容文件名的 JSON 文档。这样可以保持 CLI 依赖轻量，同时保留预期文件名。

## 本地开发

运行测试套件并构建 CLI：

```bash
go test ./...
npm test --prefix npm
go build ./cmd/skillhub
```

在不影响真实用户状态的情况下运行：

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

## 版本策略

`v1.3.x` 是当前公开安装链路的打磨线。这个系列的 patch release 应继续收口安装、更新、回滚和发布可靠性，不改变 Skill 包模型。

安装器和 CLI 打磨继续使用下一个可用的 `v1.3.x` patch tag。`v1.4.0` 保留为第一次更广泛团队更新/推广的版本，等 `v1.3.x` 的 CLI 和安装体验稳定后再发布。

## 发布

CI 会在 push 和 pull request 上运行 `go test ./...`、`npm test --prefix npm` 和 `go build -v ./cmd/skillhub`。

Tagged releases 会通过 GitHub Actions 发布多平台归档。同一个 release workflow 也可以发布：

- Homebrew formula 更新到 `CassianFlorin/homebrew-tap`，前提是 `HOMEBREW_TAP_TOKEN` 具有该 tap 仓库的写权限。
- npm 包 `@cassianflorin/skillhub`，前提是 `NPM_TOKEN` 具有 npm scope 的发布权限。
- 附加到 GitHub Release 的 npm tarball。
- `Formula/skillhub.rb` 可以在发布时通过 `scripts/generate-homebrew-formula.sh` 刷新。

```bash
NEXT_TAG=v1.3.11
git tag -a "${NEXT_TAG}" -m "${NEXT_TAG}"
git push origin "${NEXT_TAG}"
```

npm 包版本会从 Git tag 设置。安装时 npm 包会在 `postinstall` 中下载匹配的 GitHub Release 归档，并用 `checksums.txt` 校验。

Homebrew 要求 formula 位于 tap 中。仓库中的 `Formula/skillhub.rb` 镜像最新 release formula，并作为 `CassianFlorin/homebrew-tap` 发布路径的来源。

如果 release assets 已经存在，只需要在配置 secrets 后重试 npm 或 Homebrew 发布，可以手动运行 `Publish Installers` workflow，并输入已有 tag，例如 `v1.3.11`。

## 贡献与许可证

欢迎贡献。开发环境搭建与 PR 规范见 [CONTRIBUTING.md](CONTRIBUTING.md)；向公共目录投稿 Skill 请前往 [skill-hub-registry](https://github.com/CassianFlorin/skill-hub-registry)。安全问题请通过 [SECURITY.md](SECURITY.md) 中的方式私下报告。

skill-hub 使用 [MIT 许可证](LICENSE) 发布。
