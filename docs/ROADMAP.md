# skill-hub Roadmap

本文档描述 `skill-hub` 从本地优先 CLI 走向企业级 Skill 发行与治理平台的演进方向。时间线以能力成熟度为主，不承诺具体发布日期。

## v1.4：企业 Registry 安全与审计

目标：让企业内部 registry 可以安全地服务团队使用，并能满足基础审计要求。

### 企业 registry 权限

- 支持私有 Git registry 的认证配置指引和诊断。
- 为 registry 增加 owner、maintainers、visibility 等治理字段。
- 为企业 registry 预留访问策略模型，例如按 namespace、team、target runtime 限制可见性。
- 在 `doctor` 或 `registry validate` 中暴露权限和可访问性检查结果。

### 签名校验

- 在现有 checksum 校验基础上，引入可选签名字段。
- 支持 registry index 对 Skill 包或 source checksum 进行签名。
- 安装和更新时校验签名，失败时默认阻断。
- 设计 trust policy：official、curated、community、private 的默认安全策略。

### 审计日志

- 为 install、update、rollback、deploy、uninstall 增加本地审计记录。
- 审计字段包含：时间、命令、identity、版本、source ref、source commit、checksum、runtime、结果。
- 支持导出审计日志为 JSONL，方便企业接入 SIEM 或内部合规系统。
- 在 `deploy status` 中保留最近部署时间和来源摘要。

## v1.5：Skill 发布与版本兼容

目标：让 Skill 作者可以用标准命令发布 Skill，并让消费者理解兼容性风险。

### Skill 发布命令

- 增加 `skillhub publish` 命令。
- 发布前自动执行包格式校验、entry 校验、targets 校验和 checksum 计算。
- 支持发布到本地 registry 和 Git registry。
- 支持生成或更新 `skillhub.index.json`。
- 支持 dry-run，输出将要写入的 index diff。

### 版本兼容策略

- 在 `skill.yaml` 中增加兼容性字段，例如：
  - `requires.skillhub`
  - `requires.codex`
  - `requires.claude`
  - `requires.gemini`
  - `requires.hermes`
  - `compatibility.breaking`
- 安装和更新前检查兼容性，不满足时给出明确错误。
- `update` 支持兼容性策略：
  - patch/minor 自动更新。
  - major 或 breaking update 需要显式确认。
  - 支持 pin、hold 或 ignore 单个 Skill。
- 在 catalog 和 info 中展示兼容性摘要。

## v2.0：服务端 Hub 与团队治理

目标：从本地 CLI + Git registry 演进为团队级 Skill Hub 平台，同时保持 CLI 作为主要入口。

### 服务端 Hub

- 提供中心化 Skill registry API。
- 支持团队、namespace、maintainers、reviewers、release channels。
- 存储 Skill metadata、版本、checksum、签名、审计事件。
- 支持 CLI 登录、token 管理和远程策略拉取。
- 保持与现有 `skillhub.index.json` 的导入导出兼容。

### Web UI

- 提供 Skill catalog 浏览、搜索和筛选。
- 展示版本历史、安装命令、targets、trust、maintainers 和审计摘要。
- 支持发布审核、变更 diff、签名状态和兼容性提示。
- 为团队管理员提供 registry、namespace、成员和策略管理页面。

### 团队治理

- 引入组织级 policy：
  - 允许或阻止某些 trust level。
  - 限制 runtime target。
  - 要求签名或双人 review。
  - 控制 breaking update 策略。
- 支持审批流：提交、审核、批准、发布、回滚。
- 支持集中审计：谁安装了什么、部署到哪里、何时更新或回滚。
- 支持企业集成：SSO、SCIM、SIEM、内部源码托管和 artifact 存储。

## 持续原则

- 本地优先：CLI 在没有服务端时仍可使用。
- 可审计：所有安装、更新和部署都应能追溯到来源、版本和 checksum。
- 可回滚：升级和部署路径必须保留明确恢复机制。
- 可兼容：新能力应尽量兼容现有 `skill.yaml`、`skillhub.index.json` 和 `skillhub.lock`。
- 可迁移：本地 registry 和 Git registry 应能平滑迁移到服务端 Hub。
