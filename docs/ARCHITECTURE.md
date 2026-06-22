# skill-hub 架构说明

## 项目定位

`skill-hub` 是面向 AI Agent 的 Skill 包管理器。它把一个 Skill 视为可安装、可索引、可校验、可回滚、可部署的软件包，并通过单个 CLI 管理本地 Skill、Git Registry、企业内部 Registry，以及 Codex、Claude、Gemini、Hermes 等运行时目录。

当前实现是一个本地优先的 Go CLI：

- 配置文件位于项目目录的 `skillhub.yaml`。
- 运行时状态位于 `$SKILLHUB_HOME`，默认 `~/.skillhub`。
- Git Registry 缓存在 `$SKILLHUB_HOME/cache/registries`。
- 已安装 Skill 存在 `$SKILLHUB_HOME/installed`。
- 安装状态写入 `$SKILLHUB_HOME/skillhub.lock`。
- 部署目标由运行时默认目录或 `SKILLHUB_CODEX_DIR`、`SKILLHUB_CLAUDE_DIR`、`SKILLHUB_GEMINI_DIR`、`SKILLHUB_HERMES_DIR` 控制；Hermes profile 部署还可通过 `SKILLHUB_HERMES_HOME` 解析。

## 入口与模块职责

### `cmd/skillhub`

`cmd/skillhub/main.go` 是进程入口。它只做三件事：

1. 读取当前工作目录作为 `workDir`。
2. 将 `os.Args[1:]`、标准输出、标准错误和 `workDir` 交给 `internal/cli.Run`。
3. 将错误写到 stderr 并以非零状态退出。

这个入口保持很薄，便于 `internal/cli` 在测试中直接调用。

### `internal/cli`

`internal/cli` 是命令调度层。它负责：

- 解析一级命令：`version`、`doctor`、`init`、`registry`、`catalog`、`search`、`info`、`install`、`rollback`、`uninstall`、`list`、`update`、`deploy`。
- 将参数转换成对应模块的调用。
- 输出人类可读文本或 JSON。
- 处理命令级 usage 和错误。

它不直接承担持久化或复制逻辑，而是协调 `config`、`registry`、`install` 和 `deploy`。

### `internal/config`

`internal/config` 管理项目配置和默认目录：

- `skillhub.yaml` 的读写。
- 默认 `hub` 注册表：`https://github.com/CassianFlorin/skill-hub-registry.git`。
- `$SKILLHUB_HOME` 解析。
- 安装目录 `InstallDir` 和注册表列表 `Registries`。

`Registry` 当前支持两类来源：

- `local`：通过本地路径读取 registry。
- `git`：通过 Git URL 缓存并读取 registry。

### `internal/skill`

`internal/skill` 管理 Skill 元数据和目录校验：

- 读取 `skill.yaml`。
- 兼容只包含 `SKILL.md` 的旧式 Skill 目录，并在安装副本中生成 `skill.yaml`。
- 解析 `name`、`namespace`、`version`、`entry`、`targets`、`tags` 等字段。
- 计算 identity：`namespace/name`。
- 将 identity 转成可用于目录名的安全格式。
- 计算 Skill 目录 checksum。

### `internal/registry`

`internal/registry` 管理 registry 索引、搜索、Git 缓存和 catalog 导出：

- `git.go`：维护 Git Registry 缓存，支持默认分支 `pull --ff-only` 和指定 ref 的 detached checkout。
- `index.go`：读取、生成、校验 `skillhub.index.json`，实现搜索、列表、聚合、静态 catalog 导出。
- 校验 registry source path 不允许绝对路径、反斜杠或逃逸 registry root。
- 对带 checksum 的索引项执行目录 checksum 校验。

### `internal/install`

`internal/install` 管理安装、更新、回滚、卸载和锁文件：

- `Install`：解析安装 spec，复制 Skill 到安装目录，写入 lockfile。
- `UpdateAll`：基于 lockfile 刷新来源并更新版本变化的 Skill。
- `Rollback`：从历史快照恢复上一版。
- `Uninstall`：移除安装副本并更新 lockfile。
- `LoadLock` / `SaveLock`：读写 `$SKILLHUB_HOME/skillhub.lock`。

安装覆盖已有 Skill 前会保存历史快照到 `$SKILLHUB_HOME/history/<safe-identity>`。

### `internal/deploy`

`internal/deploy` 管理运行时部署：

- 支持运行时：`codex`、`claude`、`gemini`、`hermes`。
- 根据环境变量或用户 home 下默认目录解析目标路径。
- 从 lockfile 读取已安装 Skills。
- 校验 Skill 的 `targets` 是否支持目标运行时。
- 支持 `--dry-run` 和 `--force`。
- 部署后更新 lockfile 中的 `deployed_runtimes`。
- `deploy status` 通过 checksum 区分 `deployed`、`missing`、`drifted`、`unsupported`。

## 核心概念

### Skill

Skill 是一个目录，推荐包含：

```text
skill-name/
├── skill.yaml
├── SKILL.md
├── references/
├── scripts/
└── assets/
```

`skill.yaml` 是元数据入口，`SKILL.md` 是默认执行说明入口。没有 `skill.yaml` 的旧目录仍可安装，安装时会生成兼容元数据。

### Registry

Registry 是 Skill 集合来源。当前支持：

- 本地 registry：适合企业内部共享目录、monorepo、开发调试。
- Git registry：适合公开或私有 Git 仓库，支持 tag、branch、commit pinning。

项目配置通过 `skillhub.yaml` 记录 registry 名称、类型和位置。

### Index

Index 是 registry 的 catalog 文件，文件名为 `skillhub.index.json`，当前 schema version 为 `2`。

Index 记录每个 Skill 的 identity、版本、描述、targets、tags、source、maintainers、trust、checksum 等信息。它让 catalog、search、info 和安装解析不必扫描整个 registry 目录。

### Lockfile

Lockfile 是 `$SKILLHUB_HOME/skillhub.lock`。它记录本机已安装 Skills 的实际状态：

- identity、name、namespace、version。
- source type、registry、URL、ref、commit、path、subpath、cache name。
- installed path、checksum、targets、deployed runtimes、更新时间。

lockfile 是更新、回滚、部署状态判断的事实来源。

### Runtime Target

Runtime Target 是 Skill 被复制到的 Agent 运行时目录。当前内置目标：

| Runtime | 环境变量 | 默认目录 |
| --- | --- | --- |
| Codex | `SKILLHUB_CODEX_DIR` | `~/.codex/skills` |
| Claude | `SKILLHUB_CLAUDE_DIR` | `~/.claude/skills` |
| Gemini | `SKILLHUB_GEMINI_DIR` | `~/.gemini/skills` |
| Hermes | `SKILLHUB_HERMES_DIR` | `~/.hermes/skills` |

Skill 没有声明 `targets` 时视为兼容全部运行时，用于兼容旧包。

## 核心流程

### Registry 流程

1. `skillhub registry add local <name> <path>` 将本地路径写入 `skillhub.yaml`。
2. `skillhub registry add git <name> <url>` 将 Git URL 写入 `skillhub.yaml`。
3. `skillhub registry sync <name>`：
   - local registry 直接读取本地 root。
   - git registry 先维护 `$SKILLHUB_HOME/cache/registries/<name>`。
   - 读取 `skillhub.index.json`。
   - 校验 source path、identity 和 checksum。
4. `skillhub registry index generate <name>` 扫描 registry root 下的 Skill 目录，生成 schema v2 index。
5. `skillhub registry index validate <name>` 读取并校验已有 index。

### Install 流程

1. CLI 接收 `skillhub install <path|registry/skill>`。
2. `install.Install` 读取项目配置。
3. `resolveSource` 解析安装来源：
   - 路径安装：转成绝对路径，source type 为 `local`。
   - local registry：优先通过 index 定位，找不到则回退到 registry root 下的相对目录。
   - git registry：确保 Git cache 存在；带 ref 时使用独立 ref cache 并 checkout 到指定 ref。
   - indexed source 可指向 registry 内路径，也可指向另一个 Git URL。
4. 读取 Skill 元数据，计算 identity。
5. 若已安装同 identity，先保存历史快照。
6. 复制 source 目录到 `cfg.InstallDir/<safe-identity>`。
7. 对旧式 Skill 生成 `skill.yaml`。
8. 计算 checksum。
9. upsert 到 lockfile 并保存。

### Update 流程

1. `skillhub update` 调用 `install.UpdateAll`。
2. 遍历 lockfile 中所有已安装 Skills。
3. Git 来源会刷新 cache：
   - 无 ref：`git pull --ff-only` 后读取当前 commit。
   - 有 ref：`fetch --tags --prune` 并 detached checkout 到 ref。
4. 重新读取 source path 下的 metadata。
5. 只有 `meta.Version != locked.Version` 时才复制新内容。
6. 更新 installed copy、checksum、identity、targets、更新时间和 lockfile。
7. CLI 输出 `<identity> <old-version> -> <new-version>`。

### Deploy 流程

1. `skillhub deploy <runtime> [identity] [--dry-run] [--force] [--profile <name>]` 进入 `internal/deploy`。
2. 解析 runtime 目录：优先环境变量，否则使用默认目录；Hermes 带 `--profile` 时部署到 `~/.hermes/profiles/<profile>/skills`，可用 `SKILLHUB_HERMES_HOME` 覆盖 Hermes home。
3. 读取 lockfile，筛选指定 identity 或全部 Skills。
4. 检查 `targets` 是否支持 runtime。
5. 预检目标路径：
   - 目标已存在且没有 `--force` 时报告 conflict。
   - `--dry-run` 只返回 would-deploy 或 conflict，不写文件。
6. 非 dry-run 且无冲突时复制 installed copy 到 runtime target。
7. 更新 lockfile 中的 `deployed_runtimes`。
8. `deploy status` 通过目标目录 checksum 与 lockfile checksum 判断部署状态。

### Rollback 流程

1. 安装覆盖已有 Skill 时，`saveHistory` 将当前安装副本复制到 `$SKILLHUB_HOME/history/<safe-identity>/<timestamp>/files`，并写入 `manifest.json`。
2. `skillhub rollback <identity>` 查找当前 lockfile 记录。
3. 读取该 identity 的最新历史快照。
4. 将历史 `files` 复制回当前 `InstalledPath`。
5. 用历史 manifest 中的 locked state 更新 lockfile，并刷新 `UpdatedAt`。

## 工作流

### 本地安装工作流

适合开发者调试单个 Skill：

```bash
skillhub install ./examples/local-registry/java-review
skillhub deploy codex platform-team/java-review --dry-run
skillhub deploy codex platform-team/java-review --force
```

本地安装不会依赖 registry index。identity 来自 `skill.yaml`，没有元数据时由目录名和 fallback namespace 推导。

### Git Registry 工作流

适合公开分发或团队共享：

```bash
skillhub init
skillhub registry sync hub
skillhub catalog list --registry hub
skillhub install hub/official/git-commit-cn
skillhub install hub/official/git-commit-cn@v1.0.0
skillhub update
```

Git Registry 会缓存到 `$SKILLHUB_HOME/cache/registries`。带 ref 的安装会记录 resolved commit，便于审计来源。

### 企业内部 Registry 工作流

适合企业内部平台团队维护一组受控 Skills：

```bash
skillhub registry add local company /path/to/company-skills
skillhub registry index generate company
skillhub registry index validate company
skillhub catalog list --registry company --trust private
skillhub install company/platform-review
skillhub deploy codex company/platform-review --force
```

企业 Registry 可以从本地目录开始，后续迁移到私有 Git 仓库。当前 trust、maintainers、checksum 已在 index 模型中预留，后续可扩展权限、签名、审计能力。

## 后续扩展方向

- 权限与身份：为企业 registry 增加认证、授权和多团队访问控制。
- 供应链安全：在 checksum 基础上增加签名校验、发布者身份、可验证 provenance。
- 审计日志：记录 install、update、rollback、deploy 的操作者、时间、来源和结果。
- 发布体验：增加 `skillhub publish` 命令，自动校验包格式、生成 index、打 tag、推送 registry。
- 版本兼容：为 Skill 声明 CLI 版本、运行时版本、Agent 类型和目标能力约束。
- 服务端 Hub：集中托管 registry、团队治理、Web UI、审批流和组织级策略。
