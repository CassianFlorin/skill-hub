# skill-hub

[English](README.md) · [簡體中文](README.zh-CN.md) · [繁體中文](README.zh-TW.md)

`skill-hub` 是面向 AI Agent 的 Skill 套件管理器。它把 Skill 當作可安裝的軟體套件來管理，支援中繼資料、註冊表索引、鎖定檔狀態、還原歷史和執行時部署目標。

目前發佈線：`v1.3.x`。

## 功能概覽

- 從本機或 Git 註冊表發現 Skills。
- 將 Skills 安裝到 `$SKILLHUB_HOME` 下的託管本機目錄。
- 在 `skillhub.lock` 中記錄已安裝狀態，包含版本、校驗和、來源 ref 和已部署執行時。
- 支援按元資料版本、Git tag、分支或 commit 固定安裝。
- 將已安裝 Skills 部署到 Codex、Claude、Gemini 和 Hermes 的執行時目錄。
- 匯出靜態目錄快照為 `index.html` 和 `catalog.json`。
- 在同步或發佈前驗證註冊表索引。

## 安裝

使用 Homebrew 安裝：

```bash
brew tap CassianFlorin/tap
brew install skillhub
brew upgrade skillhub
```

使用 npm 安裝：

```bash
npm install -g @cassianflorin/skillhub
npm update -g @cassianflorin/skillhub
```

每個 tagged release 也會附帶 npm tarball。如果你需要固定或鏡像某個具體版本，可以使用 release URL：

```bash
VERSION=1.3.0
npm install -g "https://github.com/CassianFlorin/skill-hub/releases/download/v${VERSION}/cassianflorin-skillhub-${VERSION}.tgz"
```

開發者也可以直接從原始碼安裝：

```bash
go install github.com/cassian/skill-hub/cmd/skillhub@latest
```

## 快速開始

檢查 CLI：

```bash
skillhub version
```

初始化專案。這會新增預設的 `hub` 註冊表，指向 `https://github.com/CassianFlorin/skill-hub-registry.git`。

```bash
skillhub init
skillhub registry sync hub
```

發現和檢視 Skills：

```bash
skillhub catalog featured --registry hub
skillhub catalog list --registry hub --target codex
skillhub search git
skillhub info hub/official/git-commit-cn
```

安裝並部署：

```bash
skillhub install hub/official/git-commit-cn
skillhub deploy codex official/git-commit-cn --force
skillhub deploy status
```

`skillhub help` 支援英語、簡體中文和繁體中文：

```bash
skillhub help --lang en
skillhub help --lang zh-CN
skillhub help list --lang zh-TW
SKILLHUB_LANG=zh-TW skillhub help init
```

如果只想建置本機二進位而不安裝：

```bash
go build -o skillhub ./cmd/skillhub
```

## 命令概覽

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
skillhub hold official/git-commit-cn --reason "這個版本效果最好"
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

## 終端圖形化管理

`skillhub tui` 會開啟終端圖形化管理介面，用於管理本機 Skills 和已同步的目錄資料。它會並排展示全域 Skills、專案內 Skills、Codex/Claude/Gemini 部署狀態、Catalog 搜尋結果和操作記錄。

```bash
skillhub tui
```

TUI 使用混合安全模式：install、update、registry sync 和一般 deploy 直接執行並記錄操作；uninstall、rollback、force deploy、刪除 registry、覆蓋專案 Skill 需要二次確認。

## Catalog 發現

`catalog` 會讀取已同步的註冊表索引並列出可安裝的 Skills。新註冊表或索引可能過期時，請先執行 `registry sync`。

```bash
skillhub registry sync hub
skillhub catalog list --registry hub
skillhub catalog featured --registry hub
skillhub catalog list --registry hub --target claude
skillhub catalog list --registry hub --namespace official --trust official
```

可用的發現維度：

```bash
skillhub catalog tags --registry hub
skillhub catalog targets --registry hub
skillhub catalog namespaces --registry hub
skillhub catalog trust --registry hub
```

`search` 會優先回傳更强符合：identity/name 精確或前綴符合優先，其次是 tag 符合，最後是 description 符合。在符合強度相同時，featured 和 official Skills 會排在更前。

```bash
skillhub search git
skillhub search runtime --json
```

面向自動化情境，catalog、search 和 info 命令支援 JSON 輸出：

```bash
skillhub catalog list --registry hub --json
skillhub catalog featured --registry hub --json
skillhub catalog tags --registry hub --json
skillhub catalog targets --registry hub --json
skillhub info hub/official/git-commit-cn --json
```

## 靜態目錄匯出

`catalog export` 會寫出可瀏覽的 `index.html` 和結構化的 `catalog.json`，用於發佈或審核 marketplace 快照。

```bash
skillhub catalog export --registry hub --output ./public/catalog
skillhub catalog export --registry hub --target codex --namespace official --output ./public/codex
```

匯出的 JSON 包含：

- Skills 及其註冊表名、元資料和安裝命令。
- 彙總後的 tags。
- 彙總後的執行時 targets。
- 彙總後的 namespaces。
- 彙總後的 trust levels。

## 安裝、更新和還原

從註冊表安裝：

```bash
skillhub install hub/official/git-commit-cn
```

安裝本機 Skill 目錄：

```bash
skillhub install ./examples/local-registry/java-review
```

固定版本安裝範例：

```bash
skillhub install company/java-review@1.2.0
skillhub install team/java-review@v1.2.0
skillhub install team/java-review@main
skillhub install team/java-review@<commit-sha>
```

對於本機註冊表，固定值必須符合 Skill 中繼資料版本。對於 Git 註冊表，固定值會被視為 Git ref，並和解析出的 commit 一起記錄到 `skillhub.lock`。

更新感知和安全預覽：

```bash
skillhub check
skillhub update --preview
skillhub update --dry-run   # 相容舊用法，等同於 --preview
```

`skillhub check` 只檢查已安裝 Skill 是否有新的來源版本，不會修改任何檔案。`skillhub update --preview` 會在寫入前展示版本變化、來源、targets、已部署執行時和還原命令。

凍結暫時不想更新的 Skill：

```bash
skillhub hold platform-team/java-review --reason "1.2.0 效果最好"
skillhub holds
skillhub unhold platform-team/java-review
```

被 hold 的 Skills 仍會出現在 `skillhub check` 和 `skillhub update --preview` 中，policy 顯示為 `held`，但 `skillhub update` 會跳過它們，直到執行 `unhold`。

更新和還原：

```bash
skillhub update platform-team/java-review
skillhub update
skillhub history platform-team/java-review
skillhub rollback platform-team/java-review --to 1.2.0
skillhub rollback platform-team/java-review --to 1.2.0 --deploy hermes --profile work
```

對於 Git 註冊表，`skillhub update` 會透過 `git pull --ff-only` 重新整理快取倉庫，並在 `skill.yaml` 版本變更時更新已安裝 Skills。`skillhub history <identity>` 會列出更新或重新安裝前儲存的回滾快照。`skillhub rollback <identity>` 預設還原最近一次歷史安裝副本，`--to <version>` 可以選擇指定歷史版本。update 和 rollback 預設都不會修改執行時副本；如果傳入 `--deploy <runtime>`，例如 `skillhub rollback platform-team/java-review --to 1.2.0 --deploy hermes --profile work`，會還原託管副本並立即用覆蓋語義重新部署 Hermes profile 副本。

解除安裝：

```bash
skillhub uninstall platform-team/java-review
skillhub uninstall platform-team/java-review --deployed
```

預設情況下，解除安裝只會移除已安裝儲存副本和鎖定檔記錄。使用 `--deployed` 可以同時移除 Codex、Claude、Gemini 和 Hermes 執行時副本。

## 執行時部署

支援的執行時目標：

| Runtime | 環境變數 | 預設目錄 |
| --- | --- | --- |
| Codex | `SKILLHUB_CODEX_DIR` | `~/.codex/skills` |
| Claude | `SKILLHUB_CLAUDE_DIR` | `~/.claude/skills` |
| Gemini | `SKILLHUB_GEMINI_DIR` | `~/.gemini/skills` |
| Hermes | `SKILLHUB_HERMES_DIR` | `~/.hermes/skills` |

部署範例：

```bash
skillhub deploy codex platform-team/java-review --dry-run
skillhub deploy codex platform-team/java-review --force
skillhub deploy claude platform-team/java-review --force
skillhub deploy gemini platform-team/java-review --force
skillhub deploy hermes platform-team/java-review --force
skillhub deploy hermes platform-team/java-review --profile work --force
```

Hermes 支援 `--profile <name>`，會部署到 `~/.hermes/profiles/<name>/skills`。如需覆蓋 profile 根目錄，可設定 `SKILLHUB_HERMES_HOME`。

檢視狀態：

```bash
skillhub deploy status
skillhub deploy status codex
skillhub deploy status claude
skillhub deploy status gemini
skillhub deploy status hermes
```

部署會遵守已安裝 Skill 的 `targets` 中繼資料。沒有宣告 targets 的 Skills 會被視為相容所有支援的執行時，以便向後相容。批次部署時，不相容的 Skills 會被略過；如果明確把不相容的 Skill 部署到某個執行時，命令會回傳錯誤。

沒有 `--force` 時，deploy 會在複製前預先檢查所有選取的 Skills。如果任一目標已存在，命令會在產生部分寫入前失敗。使用 `--force` 可以取代既有執行時副本。

部署狀態：

- `deployed`：執行時副本與已安裝 Skill 校驗和一致。
- `missing`：該 Skill 支援執行時，但尚未部署。
- `drifted`：執行時副本存在，但與已安裝 Skill 校驗和不同。
- `unsupported`：該 Skill 的 targets 不包含該執行時。

## Skill 包格式

建議結構：

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

只包含 `SKILL.md` 的既有 Skill 目錄仍然可以安裝。skill-hub 會在已安裝副本中寫入產生的 `skill.yaml`，這樣鎖定檔和部署流程可以使用同一套中繼資料模型。

已安裝 Skill identity 的顯示格式為 `namespace/name`，namespace 按以下優先順序決定：

1. `skill.yaml.namespace`
2. `skill.yaml.author`
3. 註冊表名稱
4. 本機使用者名
5. `unknown`

## 註冊表索引格式

註冊表使用 `skillhub.index.json` schema v2。沒有 `schema_version: "2"` 的舊索引檔會在驗證時被拒絕。

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

驗證註冊表：

```bash
skillhub registry index validate hub
```

為本機註冊表生成索引：

```bash
skillhub registry index generate company
```

## 狀態檔案

- 專案設定：`skillhub.yaml`
- 執行時 home：`$SKILLHUB_HOME`，預設 `~/.skillhub`
- Git 註冊表快取：`$SKILLHUB_HOME/cache/registries/<registry-name>`
- 已安裝 Skills：`$SKILLHUB_HOME/installed/<safe-identity>`
- 鎖定檔：`$SKILLHUB_HOME/skillhub.lock`
- `skillhub list` 會發現的專案內 Skill 根目錄：`.skillhub/skills`、`.codex/skills`、`.claude/skills`、`.agents/skills`、`agent/skills`
- 執行時副本：由上方執行時環境變數設定

`skillhub.yaml` 和 `skillhub.lock` 是帶有 YAML 相容檔名的 JSON 檔案。這樣可以保持 CLI 依賴輕量，同時保留預期檔名。

## 本機開發

執行測試套件並建置 CLI：

```bash
go test ./...
npm test --prefix npm
go build ./cmd/skillhub
```

在不影響真實使用者狀態的情況下執行：

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

`v1.3.x` 是目前公開安裝流程的打磨線。這個系列的 patch release 應繼續收口安裝、更新、還原和發佈可靠性，不變更 Skill 包模型。

安裝器和 CLI 打磨繼續使用下一個可用的 `v1.3.x` patch tag。`v1.4.0` 保留為第一次更廣泛團隊更新/推廣的版本，等 `v1.3.x` 的 CLI 和安裝體驗穩定後再發佈。

## 發佈

CI 會在 push 和 pull request 上執行 `go test ./...`、`npm test --prefix npm` 和 `go build -v ./cmd/skillhub`。

Tagged releases 會透過 GitHub Actions 發佈多平台歸檔。同一個 release workflow 也可以發佈：

- Homebrew formula 更新到 `CassianFlorin/homebrew-tap`，前提是 `HOMEBREW_TAP_TOKEN` 具有該 tap 倉庫的寫入權限。
- npm 包 `@cassianflorin/skillhub`，前提是 `NPM_TOKEN` 具有 npm scope 的發佈權限。
- 附加到 GitHub Release 的 npm tarball。
- `Formula/skillhub.rb` 可以在發佈時透過 `scripts/generate-homebrew-formula.sh` 重新整理。

```bash
NEXT_TAG=v1.3.8
git tag -a "${NEXT_TAG}" -m "${NEXT_TAG}"
git push origin "${NEXT_TAG}"
```

npm 包版本會從 Git tag 設定。安裝時 npm 包會在 `postinstall` 中下載相符的 GitHub Release 歸檔，並用 `checksums.txt` 驗證。

Homebrew 要求 formula 位於 tap 中。倉庫中的 `Formula/skillhub.rb` 鏡像最新 release formula，並作為 `CassianFlorin/homebrew-tap` 發佈路徑的來源。

如果 release assets 已經存在，只需要在設定 secrets 後重試 npm 或 Homebrew 發佈，可以手動執行 `Publish Installers` workflow，並輸入既有 tag，例如 `v1.3.8`。
