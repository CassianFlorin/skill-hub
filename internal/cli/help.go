package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

type helpCommand struct {
	Name        string
	Description string
}

type helpTopic struct {
	Usage       string
	Description string
	Sections    []helpSection
	Examples    []string
}

type helpSection struct {
	Title string
	Lines []string
}

type helpMessages struct {
	RootUsage       string
	CommandsTitle   string
	Footer          string
	ExamplesTitle   string
	Commands        []helpCommand
	Topics          map[string]helpTopic
	UsageError      string
	UnknownTemplate string
}

func runHelp(args []string, stdout io.Writer) error {
	lang, args, err := parseHelpLanguage(args)
	if err != nil {
		return err
	}
	messages := localizedHelp(lang)
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stdout, messages.RootUsage)
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintln(stdout, messages.CommandsTitle)
		for _, command := range messages.Commands {
			_, _ = fmt.Fprintf(stdout, "  %-11s %s\n", command.Name, command.Description)
		}
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintln(stdout, messages.Footer)
		return nil
	}
	if len(args) > 1 {
		return errors.New(messages.UsageError)
	}
	topic, ok := messages.Topics[args[0]]
	if !ok {
		return fmt.Errorf(messages.UnknownTemplate, args[0])
	}
	printHelpTopic(stdout, messages, topic)
	return nil
}

func parseHelpLanguage(args []string) (string, []string, error) {
	lang := normalizeHelpLanguage(os.Getenv("SKILLHUB_LANG"))
	if lang == "" {
		lang = normalizeHelpLanguage(os.Getenv("LANG"))
	}
	if lang == "" {
		lang = "en"
	}
	remaining := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] != "--lang" {
			remaining = append(remaining, args[i])
			continue
		}
		i++
		if i >= len(args) {
			return "", nil, fmt.Errorf("--lang requires a value")
		}
		normalized := normalizeHelpLanguage(args[i])
		if normalized == "" {
			return "", nil, fmt.Errorf("unsupported help language %q", args[i])
		}
		lang = normalized
	}
	return lang, remaining, nil
}

func normalizeHelpLanguage(lang string) string {
	lang = strings.ToLower(strings.ReplaceAll(lang, "_", "-"))
	switch {
	case strings.HasPrefix(lang, "en"):
		return "en"
	case strings.HasPrefix(lang, "zh-tw"), strings.HasPrefix(lang, "zh-hk"), strings.HasPrefix(lang, "zh-hant"):
		return "zh-TW"
	case strings.HasPrefix(lang, "zh-cn"), strings.HasPrefix(lang, "zh-sg"), strings.HasPrefix(lang, "zh-hans"), lang == "zh":
		return "zh-CN"
	default:
		return ""
	}
}

func printHelpTopic(stdout io.Writer, messages helpMessages, topic helpTopic) {
	_, _ = fmt.Fprintln(stdout, topic.Usage)
	_, _ = fmt.Fprintln(stdout)
	if topic.Description != "" {
		_, _ = fmt.Fprintln(stdout, topic.Description)
		_, _ = fmt.Fprintln(stdout)
	}
	for _, section := range topic.Sections {
		_, _ = fmt.Fprintln(stdout, section.Title)
		for _, line := range section.Lines {
			_, _ = fmt.Fprintln(stdout, line)
		}
		_, _ = fmt.Fprintln(stdout)
	}
	if len(topic.Examples) == 0 {
		return
	}
	_, _ = fmt.Fprintln(stdout, messages.ExamplesTitle)
	for _, example := range topic.Examples {
		_, _ = fmt.Fprintln(stdout, example)
	}
}

func localizedHelp(lang string) helpMessages {
	switch lang {
	case "zh-CN":
		return simplifiedHelp()
	case "zh-TW":
		return traditionalHelp()
	default:
		return englishHelp()
	}
}

func englishHelp() helpMessages {
	return helpMessages{
		RootUsage:       rootUsage,
		CommandsTitle:   "Commands:",
		Footer:          "Run \"skillhub help <command>\" for command-specific usage.",
		ExamplesTitle:   "Examples:",
		UsageError:      "usage: skillhub help [command] [--lang en|zh-CN|zh-TW]",
		UnknownTemplate: "unknown help topic %q; run `skillhub help`",
		Commands: []helpCommand{
			{"version", "Print the skillhub build version"},
			{"doctor", "Show local config and runtime readiness"},
			{"init", "Create skillhub.yaml in the current project"},
			{"registry", "Add, list, sync, and index registries"},
			{"catalog", "Browse synced registry catalog data"},
			{"search", "Search synced catalog data"},
			{"info", "Show one catalog Skill"},
			{"install", "Install a Skill from a path or registry"},
			{"history", "List rollback history for an installed Skill"},
			{"rollback", "Restore a previous installed copy"},
			{"hold", "Freeze an installed Skill at its current version"},
			{"unhold", "Allow a held Skill to update again"},
			{"holds", "List held Skills"},
			{"uninstall", "Remove an installed Skill"},
			{"list", "List global and project Skills"},
			{"check", "Check installed Skills for available updates"},
			{"update", "Update installed Skills from their sources"},
			{"deploy", "Copy installed Skills into runtime directories"},
			{"tui", "Open the terminal management interface"},
		},
		Topics: englishTopics(),
	}
}

func simplifiedHelp() helpMessages {
	return helpMessages{
		RootUsage:       "用法：skillhub <命令>",
		CommandsTitle:   "命令：",
		Footer:          "运行 \"skillhub help <命令>\" 查看命令说明。可使用 --lang en|zh-CN|zh-TW 切换语言。",
		ExamplesTitle:   "示例：",
		UsageError:      "用法：skillhub help [命令] [--lang en|zh-CN|zh-TW]",
		UnknownTemplate: "未知帮助主题 %q；请运行 `skillhub help`",
		Commands: []helpCommand{
			{"version", "打印 skillhub 构建版本"},
			{"doctor", "显示本地配置和运行时状态"},
			{"init", "在当前项目中创建 skillhub.yaml"},
			{"registry", "添加、列出、同步和索引注册表"},
			{"catalog", "浏览已同步的注册表目录数据"},
			{"search", "搜索已同步的目录数据"},
			{"info", "显示一个目录 Skill"},
			{"install", "从路径或注册表安装 Skill"},
			{"history", "列出已安装 Skill 的回滚历史"},
			{"rollback", "恢复某个历史副本"},
			{"hold", "冻结已安装 Skill 的当前版本"},
			{"unhold", "允许 held Skill 再次更新"},
			{"holds", "列出 held Skills"},
			{"uninstall", "移除已安装 Skill"},
			{"list", "列出全局和项目 Skill"},
			{"check", "检查已安装 Skill 是否有更新"},
			{"update", "从来源更新已安装 Skill"},
			{"deploy", "复制已安装 Skill 到运行时目录"},
			{"tui", "打开终端图形化管理界面"},
		},
		Topics: simplifiedTopics(),
	}
}

func traditionalHelp() helpMessages {
	return helpMessages{
		RootUsage:       "用法：skillhub <命令>",
		CommandsTitle:   "命令：",
		Footer:          "執行 \"skillhub help <命令>\" 查看命令說明。可使用 --lang en|zh-CN|zh-TW 切換語言。",
		ExamplesTitle:   "範例：",
		UsageError:      "用法：skillhub help [命令] [--lang en|zh-CN|zh-TW]",
		UnknownTemplate: "未知說明主題 %q；請執行 `skillhub help`",
		Commands: []helpCommand{
			{"version", "列印 skillhub 建置版本"},
			{"doctor", "顯示本機設定與執行時狀態"},
			{"init", "在目前專案中建立 skillhub.yaml"},
			{"registry", "新增、列出、同步與索引註冊表"},
			{"catalog", "瀏覽已同步的註冊表目錄資料"},
			{"search", "搜尋已同步的目錄資料"},
			{"info", "顯示一個目錄 Skill"},
			{"install", "從路徑或註冊表安裝 Skill"},
			{"history", "列出已安裝 Skill 的回滾歷史"},
			{"rollback", "還原某個歷史副本"},
			{"hold", "凍結已安裝 Skill 的目前版本"},
			{"unhold", "允許 held Skill 再次更新"},
			{"holds", "列出 held Skills"},
			{"uninstall", "移除已安裝 Skill"},
			{"list", "列出全域與專案 Skill"},
			{"check", "檢查已安裝 Skill 是否有更新"},
			{"update", "從來源更新已安裝 Skill"},
			{"deploy", "複製已安裝 Skill 到執行時目錄"},
			{"tui", "開啟終端圖形化管理介面"},
		},
		Topics: traditionalTopics(),
	}
}

func englishTopics() map[string]helpTopic {
	return map[string]helpTopic{
		"version":   {Usage: "usage: skillhub version", Description: "Print the skillhub build version.", Examples: []string{"  skillhub version"}},
		"doctor":    {Usage: "usage: skillhub doctor", Description: "Show local config, runtime directories, registries, and installed Skill count.", Examples: []string{"  skillhub doctor"}},
		"init":      {Usage: "usage: skillhub init", Description: "Create skillhub.yaml in the current project with the default hub registry.", Examples: []string{"  skillhub init"}},
		"registry":  {Usage: registryUsage, Examples: []string{"  skillhub registry add local company ./examples/local-registry", "  skillhub registry add git team git@github.com:your-org/skills.git", "  skillhub registry list", "  skillhub registry sync hub", "  skillhub registry sync --all", "  skillhub registry index generate company", "  skillhub registry index validate company"}},
		"catalog":   {Usage: catalogUsage, Sections: []helpSection{{Title: "Options:", Lines: []string{"  --registry, --target, --tag, --namespace, --trust, --featured, --official, --json", "  export also requires --output <dir>"}}}, Examples: []string{"  skillhub catalog list --registry hub", "  skillhub catalog featured --registry hub", "  skillhub catalog tags --registry hub", "  skillhub catalog export --registry hub --output ./public/catalog"}},
		"search":    {Usage: "usage: skillhub search <query> [--json]", Description: "Search synced registry catalog data.", Examples: []string{"  skillhub search git", "  skillhub search runtime --json"}},
		"info":      {Usage: "usage: skillhub info <registry/identity|identity> [--json]", Description: "Show details for one catalog Skill from synced registry indexes.", Examples: []string{"  skillhub info hub/official/git-commit-cn", "  skillhub info official/git-commit-cn --json"}},
		"install":   {Usage: "usage: skillhub install <path|registry/skill>", Description: "Install a Skill from a local path, local registry, or Git registry.", Examples: []string{"  skillhub install hub/official/git-commit-cn", "  skillhub install ./agent/skills/commerce-data-fix-sql", "  skillhub install company/java-review@1.2.0"}},
		"history":   {Usage: "usage: skillhub history <identity>", Description: "List rollback snapshots saved before updates and reinstalls.", Examples: []string{"  skillhub history platform-team/java-review"}},
		"rollback":  {Usage: "usage: skillhub rollback <identity> [--to <version>] [--deploy <runtime>] [--profile <name>]", Description: "Restore a previous installed copy. Without --to, restores the latest snapshot. --deploy redeploys the restored copy with --force.", Examples: []string{"  skillhub rollback platform-team/java-review", "  skillhub rollback platform-team/java-review --to 1.2.0", "  skillhub rollback platform-team/java-review --to 1.2.0 --deploy hermes --profile work"}},
		"hold":      {Usage: "usage: skillhub hold <identity> [--reason <text>]", Description: "Freeze an installed Skill at its current version so update skips it.", Examples: []string{"  skillhub hold platform-team/java-review --reason '1.2.0 works best'"}},
		"unhold":    {Usage: "usage: skillhub unhold <identity>", Description: "Allow a held Skill to update again.", Examples: []string{"  skillhub unhold platform-team/java-review"}},
		"holds":     {Usage: "usage: skillhub holds", Description: "List Skills currently held from update.", Examples: []string{"  skillhub holds"}},
		"uninstall": {Usage: "usage: skillhub uninstall <identity> [--deployed]", Description: "Remove an installed Skill. Use --deployed to also remove runtime copies.", Examples: []string{"  skillhub uninstall platform-team/java-review", "  skillhub uninstall platform-team/java-review --deployed"}},
		"deploy":    {Usage: deployUsage(), Description: "Deploy copies managed Skills into runtime directories. Use --force only when you intend to replace an existing runtime copy.", Sections: []helpSection{{Title: "Runtimes:", Lines: []string{"  " + supportedRuntimeList()}}, {Title: "Options:", Lines: []string{"  --dry-run, --force, --profile <name>"}}}, Examples: []string{"  skillhub deploy codex official/git-commit-cn", "  skillhub deploy codex official/git-commit-cn --dry-run", "  skillhub deploy hermes official/git-commit-cn --profile work", "  skillhub deploy status"}},
		"list":      {Usage: listUsage, Sections: []helpSection{{Title: "Scopes:", Lines: []string{"  --scope all      Show global installed Skills and project-only Skills", "  --scope global   Show Skills from $SKILLHUB_HOME/skillhub.lock", "  --scope project  Show Skills found in the current project"}}, {Title: "Project roots:", Lines: []string{"  .skillhub/skills, .codex/skills, .claude/skills, .agents/skills, agent/skills"}}}, Examples: []string{"  skillhub list", "  skillhub list --scope global", "  skillhub list --scope project"}},
		"check":     {Usage: "usage: skillhub check", Description: "Check installed Skills for available updates without changing managed or runtime copies.", Examples: []string{"  skillhub check", "  skillhub update --preview"}},
		"update":    {Usage: "usage: skillhub update [identity] [--preview|--dry-run]", Description: "Update managed store copies from recorded sources. Runtime copies are not changed until deploy runs.", Examples: []string{"  skillhub update --preview", "  skillhub update --dry-run", "  skillhub update official/git-commit-cn", "  skillhub update"}},
		"tui":       {Usage: "usage: skillhub tui", Description: "Open the terminal management interface. Managed means skillhub can update and rollback; Runtime means the copy agents load.", Examples: []string{"  skillhub tui"}},
	}
}

func simplifiedTopics() map[string]helpTopic {
	return map[string]helpTopic{
		"version":   {Usage: "用法：skillhub version", Description: "打印 skillhub 构建版本。", Examples: []string{"  skillhub version"}},
		"doctor":    {Usage: "用法：skillhub doctor", Description: "显示本地配置、运行时目录、注册表和已安装 Skill 数量。", Examples: []string{"  skillhub doctor"}},
		"init":      {Usage: "用法：skillhub init", Description: "在当前项目中创建 skillhub.yaml，并写入默认 hub 注册表。", Examples: []string{"  skillhub init"}},
		"registry":  {Usage: "用法：skillhub registry <add|list|sync|index>", Examples: []string{"  skillhub registry add local company ./examples/local-registry", "  skillhub registry add git team git@github.com:your-org/skills.git", "  skillhub registry list", "  skillhub registry sync hub", "  skillhub registry sync --all", "  skillhub registry index generate company", "  skillhub registry index validate company"}},
		"catalog":   {Usage: "用法：skillhub catalog <list|featured|tags|targets|namespaces|trust|export>", Sections: []helpSection{{Title: "选项：", Lines: []string{"  --registry, --target, --tag, --namespace, --trust, --featured, --official, --json", "  export 还需要 --output <目录>"}}}, Examples: []string{"  skillhub catalog list --registry hub", "  skillhub catalog featured --registry hub", "  skillhub catalog tags --registry hub", "  skillhub catalog export --registry hub --output ./public/catalog"}},
		"search":    {Usage: "用法：skillhub search <查询词> [--json]", Description: "搜索已同步的注册表目录数据。", Examples: []string{"  skillhub search git", "  skillhub search runtime --json"}},
		"info":      {Usage: "用法：skillhub info <registry/identity|identity> [--json]", Description: "显示一个目录 Skill 的详细信息。", Examples: []string{"  skillhub info hub/official/git-commit-cn", "  skillhub info official/git-commit-cn --json"}},
		"install":   {Usage: "用法：skillhub install <路径|registry/skill>", Description: "从本地路径、本地注册表或 Git 注册表安装 Skill。", Examples: []string{"  skillhub install hub/official/git-commit-cn", "  skillhub install ./agent/skills/commerce-data-fix-sql", "  skillhub install company/java-review@1.2.0"}},
		"history":   {Usage: "用法：skillhub history <identity>", Description: "列出更新或重新安装前保存的回滚快照。", Examples: []string{"  skillhub history platform-team/java-review"}},
		"rollback":  {Usage: "用法：skillhub rollback <identity> [--to <version>] [--deploy <runtime>] [--profile <name>]", Description: "恢复某个历史副本。没有 --to 时恢复最近快照；--deploy 会用 --force 重新部署恢复后的副本。", Examples: []string{"  skillhub rollback platform-team/java-review", "  skillhub rollback platform-team/java-review --to 1.2.0", "  skillhub rollback platform-team/java-review --to 1.2.0 --deploy hermes --profile work"}},
		"hold":      {Usage: "用法：skillhub hold <identity> [--reason <文本>]", Description: "冻结已安装 Skill 的当前版本，update 会跳过它。", Examples: []string{"  skillhub hold platform-team/java-review --reason '1.2.0 效果最好'"}},
		"unhold":    {Usage: "用法：skillhub unhold <identity>", Description: "允许 held Skill 再次更新。", Examples: []string{"  skillhub unhold platform-team/java-review"}},
		"holds":     {Usage: "用法：skillhub holds", Description: "列出当前被 hold 的 Skills。", Examples: []string{"  skillhub holds"}},
		"uninstall": {Usage: "用法：skillhub uninstall <identity> [--deployed]", Description: "移除已安装 Skill。使用 --deployed 可同时移除运行时副本。", Examples: []string{"  skillhub uninstall platform-team/java-review", "  skillhub uninstall platform-team/java-review --deployed"}},
		"deploy":    {Usage: "用法：" + deployUsage()[len("usage: "):], Description: "deploy 才会修改运行时目录；--force 会覆盖已有运行时副本。", Sections: []helpSection{{Title: "运行时：", Lines: []string{"  " + supportedRuntimeList()}}, {Title: "选项：", Lines: []string{"  --dry-run, --force, --profile <name>"}}}, Examples: []string{"  skillhub deploy codex official/git-commit-cn", "  skillhub deploy codex official/git-commit-cn --dry-run", "  skillhub deploy hermes official/git-commit-cn --profile work", "  skillhub deploy status"}},
		"list":      {Usage: "用法：" + listUsage[len("usage: "):], Sections: []helpSection{{Title: "范围：", Lines: []string{"  --scope all      显示全局已安装 Skill 和项目内 Skill", "  --scope global   显示 $SKILLHUB_HOME/skillhub.lock 中的 Skill", "  --scope project  显示当前项目中的 Skill"}}, {Title: "项目目录：", Lines: []string{"  .skillhub/skills, .codex/skills, .claude/skills, .agents/skills, agent/skills"}}}, Examples: []string{"  skillhub list", "  skillhub list --scope global", "  skillhub list --scope project"}},
		"check":     {Usage: "用法：skillhub check", Description: "只检查已安装 Skill 是否有更新，不修改托管副本或运行时目录。", Examples: []string{"  skillhub check", "  skillhub update --preview"}},
		"update":    {Usage: "用法：skillhub update [identity] [--preview|--dry-run]", Description: "只更新 skillhub 托管副本，不会修改 Codex、Claude、Gemini 或 Hermes 运行时目录。", Examples: []string{"  skillhub update --preview", "  skillhub update --dry-run", "  skillhub update official/git-commit-cn", "  skillhub update"}},
		"tui":       {Usage: "用法：skillhub tui", Description: "打开终端图形化管理界面。Managed 表示 skillhub 可更新和回滚；Runtime 表示 Agent 实际加载的副本。", Examples: []string{"  skillhub tui"}},
	}
}

func traditionalTopics() map[string]helpTopic {
	return map[string]helpTopic{
		"version":   {Usage: "用法：skillhub version", Description: "列印 skillhub 建置版本。", Examples: []string{"  skillhub version"}},
		"doctor":    {Usage: "用法：skillhub doctor", Description: "顯示本機設定、執行時目錄、註冊表和已安裝 Skill 數量。", Examples: []string{"  skillhub doctor"}},
		"init":      {Usage: "用法：skillhub init", Description: "在目前專案中建立 skillhub.yaml，並寫入預設 hub 註冊表。", Examples: []string{"  skillhub init"}},
		"registry":  {Usage: "用法：skillhub registry <add|list|sync|index>", Examples: []string{"  skillhub registry add local company ./examples/local-registry", "  skillhub registry add git team git@github.com:your-org/skills.git", "  skillhub registry list", "  skillhub registry sync hub", "  skillhub registry sync --all", "  skillhub registry index generate company", "  skillhub registry index validate company"}},
		"catalog":   {Usage: "用法：skillhub catalog <list|featured|tags|targets|namespaces|trust|export>", Sections: []helpSection{{Title: "選項：", Lines: []string{"  --registry, --target, --tag, --namespace, --trust, --featured, --official, --json", "  export 還需要 --output <目錄>"}}}, Examples: []string{"  skillhub catalog list --registry hub", "  skillhub catalog featured --registry hub", "  skillhub catalog tags --registry hub", "  skillhub catalog export --registry hub --output ./public/catalog"}},
		"search":    {Usage: "用法：skillhub search <查詢詞> [--json]", Description: "搜尋已同步的註冊表目錄資料。", Examples: []string{"  skillhub search git", "  skillhub search runtime --json"}},
		"info":      {Usage: "用法：skillhub info <registry/identity|identity> [--json]", Description: "顯示一個目錄 Skill 的詳細資訊。", Examples: []string{"  skillhub info hub/official/git-commit-cn", "  skillhub info official/git-commit-cn --json"}},
		"install":   {Usage: "用法：skillhub install <路徑|registry/skill>", Description: "從本機路徑、本機註冊表或 Git 註冊表安裝 Skill。", Examples: []string{"  skillhub install hub/official/git-commit-cn", "  skillhub install ./agent/skills/commerce-data-fix-sql", "  skillhub install company/java-review@1.2.0"}},
		"history":   {Usage: "用法：skillhub history <identity>", Description: "列出更新或重新安裝前儲存的回滾快照。", Examples: []string{"  skillhub history platform-team/java-review"}},
		"rollback":  {Usage: "用法：skillhub rollback <identity> [--to <version>] [--deploy <runtime>] [--profile <name>]", Description: "還原某個歷史副本。沒有 --to 時還原最近快照；--deploy 會用 --force 重新部署還原後的副本。", Examples: []string{"  skillhub rollback platform-team/java-review", "  skillhub rollback platform-team/java-review --to 1.2.0", "  skillhub rollback platform-team/java-review --to 1.2.0 --deploy hermes --profile work"}},
		"hold":      {Usage: "用法：skillhub hold <identity> [--reason <文字>]", Description: "凍結已安裝 Skill 的目前版本，update 會跳過它。", Examples: []string{"  skillhub hold platform-team/java-review --reason '1.2.0 效果最好'"}},
		"unhold":    {Usage: "用法：skillhub unhold <identity>", Description: "允許 held Skill 再次更新。", Examples: []string{"  skillhub unhold platform-team/java-review"}},
		"holds":     {Usage: "用法：skillhub holds", Description: "列出目前被 hold 的 Skills。", Examples: []string{"  skillhub holds"}},
		"uninstall": {Usage: "用法：skillhub uninstall <identity> [--deployed]", Description: "移除已安裝 Skill。使用 --deployed 可同時移除執行時副本。", Examples: []string{"  skillhub uninstall platform-team/java-review", "  skillhub uninstall platform-team/java-review --deployed"}},
		"deploy":    {Usage: "用法：" + deployUsage()[len("usage: "):], Description: "deploy 才會修改執行時目錄；--force 會覆蓋既有執行時副本。", Sections: []helpSection{{Title: "執行時：", Lines: []string{"  " + supportedRuntimeList()}}, {Title: "選項：", Lines: []string{"  --dry-run, --force, --profile <name>"}}}, Examples: []string{"  skillhub deploy codex official/git-commit-cn", "  skillhub deploy codex official/git-commit-cn --dry-run", "  skillhub deploy hermes official/git-commit-cn --profile work", "  skillhub deploy status"}},
		"list":      {Usage: "用法：" + listUsage[len("usage: "):], Sections: []helpSection{{Title: "範圍：", Lines: []string{"  --scope all      顯示全域已安裝 Skill 和專案內 Skill", "  --scope global   顯示 $SKILLHUB_HOME/skillhub.lock 中的 Skill", "  --scope project  顯示目前專案中的 Skill"}}, {Title: "專案目錄：", Lines: []string{"  .skillhub/skills, .codex/skills, .claude/skills, .agents/skills, agent/skills"}}}, Examples: []string{"  skillhub list", "  skillhub list --scope global", "  skillhub list --scope project"}},
		"check":     {Usage: "用法：skillhub check", Description: "只檢查已安裝 Skill 是否有更新，不修改託管副本或執行時目錄。", Examples: []string{"  skillhub check", "  skillhub update --preview"}},
		"update":    {Usage: "用法：skillhub update [identity] [--preview|--dry-run]", Description: "只更新 skillhub 託管副本，不會修改 Codex、Claude、Gemini 或 Hermes 執行時目錄。", Examples: []string{"  skillhub update --preview", "  skillhub update --dry-run", "  skillhub update official/git-commit-cn", "  skillhub update"}},
		"tui":       {Usage: "用法：skillhub tui", Description: "開啟終端圖形化管理介面。Managed 表示 skillhub 可更新和回滾；Runtime 表示 Agent 實際載入的副本。", Examples: []string{"  skillhub tui"}},
	}
}
