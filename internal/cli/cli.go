package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cassian/skill-hub/internal/config"
	"github.com/cassian/skill-hub/internal/deploy"
	"github.com/cassian/skill-hub/internal/install"
	projectskills "github.com/cassian/skill-hub/internal/project"
	"github.com/cassian/skill-hub/internal/registry"
	"github.com/cassian/skill-hub/internal/tui"
)

var Version = "dev"

func Run(args []string, stdout io.Writer, stderr io.Writer, workDir string) error {
	if len(args) == 0 {
		return usage(stderr)
	}
	switch args[0] {
	case "help":
		return runHelp(args[1:], stdout)
	case "version":
		return runVersion(stdout)
	case "doctor":
		return runDoctor(stdout, workDir)
	case "init":
		return runInit(stdout, workDir)
	case "registry":
		return runRegistry(args[1:], stdout, workDir)
	case "catalog":
		return runCatalog(args[1:], stdout, workDir)
	case "search":
		return runSearch(args[1:], stdout, workDir)
	case "info":
		return runInfo(args[1:], stdout, workDir)
	case "install":
		if len(args) != 2 {
			return fmt.Errorf("usage: skillhub install <path|registry/skill>")
		}
		locked, err := install.Install(workDir, args[1])
		if err != nil {
			return withCLIHint(err)
		}
		_, _ = fmt.Fprintf(stdout, "installed %s@%s\n", locked.DisplayIdentity(), locked.Version)
		return nil
	case "rollback":
		return runRollback(args[1:], stdout)
	case "uninstall":
		return runUninstall(args[1:], stdout)
	case "list":
		return runList(args[1:], stdout, workDir)
	case "update":
		return runUpdate(stdout)
	case "deploy":
		return runDeploy(args[1:], stdout)
	case "tui":
		if len(args) != 1 {
			return fmt.Errorf("usage: skillhub tui")
		}
		return tui.Run(workDir)
	default:
		return usage(stderr)
	}
}

func runVersion(stdout io.Writer) error {
	_, _ = fmt.Fprintf(stdout, "skillhub %s\n", Version)
	return nil
}

const rootUsage = "usage: skillhub <command>"
const registryUsage = "usage: skillhub registry <add|list|sync|index>"
const listUsage = "usage: skillhub list [--scope all|global|project]"
const catalogUsage = "usage: skillhub catalog <list|featured|tags|targets|namespaces|trust|export>"

func deployUsage() string {
	return fmt.Sprintf("usage: skillhub deploy <%s> [identity] [--dry-run] [--force]", strings.Join(deploy.RuntimeNames(), "|"))
}

func supportedRuntimeList() string {
	return strings.Join(deploy.RuntimeNames(), ", ")
}

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
			{"rollback", "Restore the previous installed copy"},
			{"uninstall", "Remove an installed Skill"},
			{"list", "List global and project Skills"},
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
			{"rollback", "恢复上一个已安装副本"},
			{"uninstall", "移除已安装 Skill"},
			{"list", "列出全局和项目 Skill"},
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
			{"rollback", "還原上一個已安裝副本"},
			{"uninstall", "移除已安裝 Skill"},
			{"list", "列出全域與專案 Skill"},
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
		"rollback":  {Usage: "usage: skillhub rollback <identity>", Description: "Restore the latest previous installed copy for an installed Skill.", Examples: []string{"  skillhub rollback platform-team/java-review"}},
		"uninstall": {Usage: "usage: skillhub uninstall <identity> [--deployed]", Description: "Remove an installed Skill. Use --deployed to also remove runtime copies.", Examples: []string{"  skillhub uninstall platform-team/java-review", "  skillhub uninstall platform-team/java-review --deployed"}},
		"deploy":    {Usage: deployUsage(), Sections: []helpSection{{Title: "Runtimes:", Lines: []string{"  " + supportedRuntimeList()}}, {Title: "Options:", Lines: []string{"  --dry-run, --force"}}}, Examples: []string{"  skillhub deploy codex official/git-commit-cn", "  skillhub deploy codex official/git-commit-cn --dry-run", "  skillhub deploy codex official/git-commit-cn --force", "  skillhub deploy status"}},
		"list":      {Usage: listUsage, Sections: []helpSection{{Title: "Scopes:", Lines: []string{"  --scope all      Show global installed Skills and project-only Skills", "  --scope global   Show Skills from $SKILLHUB_HOME/skillhub.lock", "  --scope project  Show Skills found in the current project"}}, {Title: "Project roots:", Lines: []string{"  .skillhub/skills, .codex/skills, .claude/skills, .agents/skills, agent/skills"}}}, Examples: []string{"  skillhub list", "  skillhub list --scope global", "  skillhub list --scope project"}},
		"update":    {Usage: "usage: skillhub update", Description: "Update installed Skills from their recorded sources.", Examples: []string{"  skillhub update"}},
		"tui":       {Usage: "usage: skillhub tui", Description: "Open the terminal management interface for local Skills, catalog browsing, deployment status, and operation logs.", Examples: []string{"  skillhub tui"}},
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
		"rollback":  {Usage: "用法：skillhub rollback <identity>", Description: "恢复某个已安装 Skill 的最近一个历史副本。", Examples: []string{"  skillhub rollback platform-team/java-review"}},
		"uninstall": {Usage: "用法：skillhub uninstall <identity> [--deployed]", Description: "移除已安装 Skill。使用 --deployed 可同时移除运行时副本。", Examples: []string{"  skillhub uninstall platform-team/java-review", "  skillhub uninstall platform-team/java-review --deployed"}},
		"deploy":    {Usage: "用法：" + deployUsage()[len("usage: "):], Sections: []helpSection{{Title: "运行时：", Lines: []string{"  " + supportedRuntimeList()}}, {Title: "选项：", Lines: []string{"  --dry-run, --force"}}}, Examples: []string{"  skillhub deploy codex official/git-commit-cn", "  skillhub deploy codex official/git-commit-cn --dry-run", "  skillhub deploy codex official/git-commit-cn --force", "  skillhub deploy status"}},
		"list":      {Usage: "用法：" + listUsage[len("usage: "):], Sections: []helpSection{{Title: "范围：", Lines: []string{"  --scope all      显示全局已安装 Skill 和项目内 Skill", "  --scope global   显示 $SKILLHUB_HOME/skillhub.lock 中的 Skill", "  --scope project  显示当前项目中的 Skill"}}, {Title: "项目目录：", Lines: []string{"  .skillhub/skills, .codex/skills, .claude/skills, .agents/skills, agent/skills"}}}, Examples: []string{"  skillhub list", "  skillhub list --scope global", "  skillhub list --scope project"}},
		"update":    {Usage: "用法：skillhub update", Description: "从记录的来源更新已安装 Skill。", Examples: []string{"  skillhub update"}},
		"tui":       {Usage: "用法：skillhub tui", Description: "打开终端图形化管理界面，用于本机 Skill 管理、目录浏览、部署状态和操作日志。", Examples: []string{"  skillhub tui"}},
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
		"rollback":  {Usage: "用法：skillhub rollback <identity>", Description: "還原某個已安裝 Skill 的最近一個歷史副本。", Examples: []string{"  skillhub rollback platform-team/java-review"}},
		"uninstall": {Usage: "用法：skillhub uninstall <identity> [--deployed]", Description: "移除已安裝 Skill。使用 --deployed 可同時移除執行時副本。", Examples: []string{"  skillhub uninstall platform-team/java-review", "  skillhub uninstall platform-team/java-review --deployed"}},
		"deploy":    {Usage: "用法：" + deployUsage()[len("usage: "):], Sections: []helpSection{{Title: "執行時：", Lines: []string{"  " + supportedRuntimeList()}}, {Title: "選項：", Lines: []string{"  --dry-run, --force"}}}, Examples: []string{"  skillhub deploy codex official/git-commit-cn", "  skillhub deploy codex official/git-commit-cn --dry-run", "  skillhub deploy codex official/git-commit-cn --force", "  skillhub deploy status"}},
		"list":      {Usage: "用法：" + listUsage[len("usage: "):], Sections: []helpSection{{Title: "範圍：", Lines: []string{"  --scope all      顯示全域已安裝 Skill 和專案內 Skill", "  --scope global   顯示 $SKILLHUB_HOME/skillhub.lock 中的 Skill", "  --scope project  顯示目前專案中的 Skill"}}, {Title: "專案目錄：", Lines: []string{"  .skillhub/skills, .codex/skills, .claude/skills, .agents/skills, agent/skills"}}}, Examples: []string{"  skillhub list", "  skillhub list --scope global", "  skillhub list --scope project"}},
		"update":    {Usage: "用法：skillhub update", Description: "從記錄的來源更新已安裝 Skill。", Examples: []string{"  skillhub update"}},
		"tui":       {Usage: "用法：skillhub tui", Description: "開啟終端圖形化管理介面，用於本機 Skill 管理、目錄瀏覽、部署狀態和操作記錄。", Examples: []string{"  skillhub tui"}},
	}
}

func runDoctor(stdout io.Writer, workDir string) error {
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	home, err := config.DefaultHome()
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(stdout, "config: ok")
	_, _ = fmt.Fprintf(stdout, "config.path: %s\n", config.Path(workDir))
	_, _ = fmt.Fprintf(stdout, "home: %s\n", home)
	_, _ = fmt.Fprintf(stdout, "install_dir: %s\n", cfg.InstallDir)
	for _, runtime := range deploy.SupportedRuntimes() {
		dir, err := deploy.RuntimeDir(runtime.Name)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "runtime %s: %s\n", runtime.Name, dir)
	}
	_, _ = fmt.Fprintf(stdout, "registries: %d\n", len(cfg.Registries))
	for _, status := range registry.ListRegistries(cfg) {
		_, _ = fmt.Fprintf(stdout, "registry %s: %s %s skills=%d generated_at=%s\n", status.Name, status.Type, status.Location, status.SkillCount, status.GeneratedAt)
	}
	lock, err := install.LoadLock()
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "installed: %d\n", len(lock.Skills))
	return nil
}

func runUninstall(args []string, stdout io.Writer) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: skillhub uninstall <identity> [--deployed]")
	}
	removeDeployed := false
	if len(args) == 2 {
		if args[1] != "--deployed" {
			return fmt.Errorf("usage: skillhub uninstall <identity> [--deployed]")
		}
		removeDeployed = true
	}
	locked, err := install.Uninstall(args[0])
	if err != nil {
		return err
	}
	if removeDeployed {
		for _, runtime := range deploy.RuntimeNames() {
			target, err := deploy.RuntimeTarget(runtime, locked.DisplayIdentity())
			if err != nil {
				return err
			}
			if err := os.RemoveAll(target); err != nil {
				return err
			}
		}
	}
	_, _ = fmt.Fprintf(stdout, "uninstalled %s\n", locked.DisplayIdentity())
	return nil
}

func runRollback(args []string, stdout io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: skillhub rollback <identity>")
	}
	locked, err := install.Rollback(args[0])
	if err != nil {
		return withCLIHint(err)
	}
	_, _ = fmt.Fprintf(stdout, "rolled back %s to %s\n", locked.DisplayIdentity(), locked.Version)
	return nil
}

func runInit(stdout io.Writer, workDir string) error {
	cfg, err := config.NewDefault()
	if err != nil {
		return err
	}
	if err := config.Save(workDir, cfg); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "initialized %s\n", config.Path(workDir))
	return nil
}

func runRegistry(args []string, stdout io.Writer, workDir string) error {
	if len(args) < 1 {
		return errors.New(registryUsage)
	}
	switch args[0] {
	case "add":
		return runRegistryAdd(args[1:], stdout, workDir)
	case "list":
		return runRegistryList(stdout, workDir)
	case "sync":
		return runRegistrySync(args[1:], stdout, workDir)
	case "index":
		return runRegistryIndex(args[1:], stdout, workDir)
	default:
		return errors.New(registryUsage)
	}
}

func runRegistryAdd(args []string, stdout io.Writer, workDir string) error {
	if len(args) != 3 {
		return fmt.Errorf("usage: skillhub registry add <local|git> <name> <path-or-url>")
	}
	registryType, name, location := args[0], args[1], args[2]
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	switch registryType {
	case "local":
		if !filepath.IsAbs(location) {
			location = filepath.Join(workDir, location)
		}
		abs, err := filepath.Abs(location)
		if err != nil {
			return err
		}
		cfg.Registries[name] = config.Registry{Type: "local", Path: abs}
	case "git":
		cfg.Registries[name] = config.Registry{Type: "git", URL: location}
	default:
		return fmt.Errorf("unsupported registry type %q", registryType)
	}
	if err := config.Save(workDir, cfg); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "registered %s\n", name)
	return nil
}

func runRegistryIndex(args []string, stdout io.Writer, workDir string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: skillhub registry index <generate|validate> <registry>")
	}
	action, name := args[0], args[1]
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	reg, ok := cfg.Registries[name]
	if !ok {
		return withCLIHint(fmt.Errorf("unknown registry %q", name))
	}
	switch action {
	case "generate":
		index, _, err := registry.GenerateIndex(name, reg)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "indexed %s with %d skills\n", name, len(index.Skills))
	case "validate":
		count, err := registry.ValidateIndex(name, reg)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "validated %s with %d skills\n", name, count)
	default:
		return fmt.Errorf("usage: skillhub registry index <generate|validate> <registry>")
	}
	return nil
}

func runRegistryList(stdout io.Writer, workDir string) error {
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	for _, status := range registry.ListRegistries(cfg) {
		_, _ = fmt.Fprintf(stdout, "%s\t%s\t%s\t%d\t%s\n", status.Name, status.Type, status.Location, status.SkillCount, status.GeneratedAt)
	}
	return nil
}

func runRegistrySync(args []string, stdout io.Writer, workDir string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: skillhub registry sync <registry|--all>")
	}
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	if args[0] == "--all" {
		for _, status := range registry.ListRegistries(cfg) {
			reg := cfg.Registries[status.Name]
			count, err := registry.SyncRegistry(status.Name, reg)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(stdout, "synced %s with %d skills\n", status.Name, count)
		}
		return nil
	}
	reg, ok := cfg.Registries[args[0]]
	if !ok {
		return withCLIHint(fmt.Errorf("unknown registry %q", args[0]))
	}
	count, err := registry.SyncRegistry(args[0], reg)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "synced %s with %d skills\n", args[0], count)
	return nil
}

func runCatalog(args []string, stdout io.Writer, workDir string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: skillhub catalog <list|featured|tags|targets|namespaces|trust|export>")
	}
	switch args[0] {
	case "list":
		return runCatalogList(args[1:], stdout, workDir, false)
	case "featured":
		return runCatalogList(args[1:], stdout, workDir, true)
	case "export":
		return runCatalogExport(args[1:], stdout, workDir)
	case "tags":
		return runCatalogAggregate(args[1:], stdout, workDir, "tags")
	case "targets":
		return runCatalogAggregate(args[1:], stdout, workDir, "targets")
	case "namespaces":
		return runCatalogAggregate(args[1:], stdout, workDir, "namespaces")
	case "trust":
		return runCatalogAggregate(args[1:], stdout, workDir, "trust")
	default:
		return fmt.Errorf("usage: skillhub catalog <list|featured|tags|targets|namespaces|trust|export>")
	}
}

func runCatalogList(args []string, stdout io.Writer, workDir string, featuredOnly bool) error {
	filter := registry.CatalogFilter{}
	jsonOutput := false
	if featuredOnly {
		featured := true
		filter.Featured = &featured
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOutput = true
		case "--registry":
			i++
			if i >= len(args) {
				return fmt.Errorf("--registry requires a value")
			}
			filter.Registry = args[i]
		case "--target":
			i++
			if i >= len(args) {
				return fmt.Errorf("--target requires a value")
			}
			filter.Target = args[i]
		case "--tag":
			i++
			if i >= len(args) {
				return fmt.Errorf("--tag requires a value")
			}
			filter.Tag = args[i]
		case "--namespace":
			i++
			if i >= len(args) {
				return fmt.Errorf("--namespace requires a value")
			}
			filter.Namespace = args[i]
		case "--trust":
			i++
			if i >= len(args) {
				return fmt.Errorf("--trust requires a value")
			}
			filter.Trust = args[i]
		case "--featured":
			featured := true
			filter.Featured = &featured
		case "--official":
			filter.Official = true
		default:
			return fmt.Errorf("unknown catalog option %q", args[i])
		}
	}
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	results, err := registry.ListCatalog(cfg, filter)
	if err != nil {
		return err
	}
	if jsonOutput {
		return writeJSON(stdout, catalogJSONResults(results))
	}
	if len(results) == 0 {
		_, _ = fmt.Fprintln(stdout, "no catalog skills found")
		return nil
	}
	for _, result := range results {
		_, _ = fmt.Fprintln(stdout, formatCatalogResult(result))
	}
	return nil
}

func runCatalogAggregate(args []string, stdout io.Writer, workDir string, kind string) error {
	filter := registry.CatalogFilter{}
	jsonOutput := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOutput = true
		case "--registry":
			i++
			if i >= len(args) {
				return fmt.Errorf("--registry requires a value")
			}
			filter.Registry = args[i]
		default:
			return fmt.Errorf("unknown catalog option %q", args[i])
		}
	}
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	results, err := registry.ListCatalog(cfg, filter)
	if err != nil {
		return err
	}
	counts := aggregateCatalog(results, kind)
	if jsonOutput {
		return writeJSON(stdout, counts)
	}
	if len(counts) == 0 {
		_, _ = fmt.Fprintf(stdout, "no catalog %s found\n", kind)
		return nil
	}
	for _, count := range counts {
		_, _ = fmt.Fprintf(stdout, "%s\t%d\n", count.Name, count.Count)
	}
	return nil
}

func runCatalogExport(args []string, stdout io.Writer, workDir string) error {
	filter := registry.CatalogFilter{}
	outputDir := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--registry":
			i++
			if i >= len(args) {
				return fmt.Errorf("--registry requires a value")
			}
			filter.Registry = args[i]
		case "--target":
			i++
			if i >= len(args) {
				return fmt.Errorf("--target requires a value")
			}
			filter.Target = args[i]
		case "--tag":
			i++
			if i >= len(args) {
				return fmt.Errorf("--tag requires a value")
			}
			filter.Tag = args[i]
		case "--namespace":
			i++
			if i >= len(args) {
				return fmt.Errorf("--namespace requires a value")
			}
			filter.Namespace = args[i]
		case "--trust":
			i++
			if i >= len(args) {
				return fmt.Errorf("--trust requires a value")
			}
			filter.Trust = args[i]
		case "--featured":
			featured := true
			filter.Featured = &featured
		case "--official":
			filter.Official = true
		case "--output":
			i++
			if i >= len(args) {
				return fmt.Errorf("--output requires a value")
			}
			outputDir = args[i]
		default:
			return fmt.Errorf("unknown catalog export option %q", args[i])
		}
	}
	if outputDir == "" {
		return fmt.Errorf("--output requires a value")
	}
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	results, err := registry.ListCatalog(cfg, filter)
	if err != nil {
		return err
	}
	if err := writeCatalogExport(outputDir, results); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "exported %d catalog skills to %s\n", len(results), outputDir)
	return nil
}

func runSearch(args []string, stdout io.Writer, workDir string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: skillhub search <query> [--json]")
	}
	query := args[0]
	jsonOutput := false
	if len(args) == 2 {
		if args[1] != "--json" {
			return fmt.Errorf("usage: skillhub search <query> [--json]")
		}
		jsonOutput = true
	}
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	results, err := registry.SearchIndexes(cfg, query)
	if err != nil {
		return err
	}
	if jsonOutput {
		return writeJSON(stdout, catalogJSONResults(results))
	}
	if len(results) == 0 {
		_, _ = fmt.Fprintln(stdout, "no skills found")
		return nil
	}
	for _, result := range results {
		_, _ = fmt.Fprintln(stdout, formatCatalogResult(result))
	}
	return nil
}

func runInfo(args []string, stdout io.Writer, workDir string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: skillhub info <registry/identity|identity> [--json]")
	}
	spec := args[0]
	jsonOutput := false
	if len(args) == 2 {
		if args[1] != "--json" {
			return fmt.Errorf("usage: skillhub info <registry/identity|identity> [--json]")
		}
		jsonOutput = true
	}
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	result, ok, err := registry.FindIndexedSkill(cfg, spec)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("skill %q not found", spec)
	}
	indexed := result.Skill
	installCommand := fmt.Sprintf("skillhub install %s/%s", result.Registry, indexed.Identity)
	if jsonOutput {
		return writeJSON(stdout, infoJSONResult{
			Registry:       result.Registry,
			Skill:          indexed,
			InstallCommand: installCommand,
		})
	}
	_, _ = fmt.Fprintf(stdout, "identity: %s\n", indexed.Identity)
	_, _ = fmt.Fprintf(stdout, "registry: %s\n", result.Registry)
	_, _ = fmt.Fprintf(stdout, "name: %s\n", indexed.Name)
	_, _ = fmt.Fprintf(stdout, "namespace: %s\n", indexed.Namespace)
	_, _ = fmt.Fprintf(stdout, "version: %s\n", indexed.Version)
	_, _ = fmt.Fprintf(stdout, "description: %s\n", indexed.Description)
	_, _ = fmt.Fprintf(stdout, "targets: %s\n", strings.Join(indexed.Targets, ", "))
	_, _ = fmt.Fprintf(stdout, "tags: %s\n", strings.Join(indexed.Tags, ", "))
	_, _ = fmt.Fprintf(stdout, "source.type: %s\n", indexed.Source.Type)
	_, _ = fmt.Fprintf(stdout, "source.url: %s\n", indexed.Source.URL)
	_, _ = fmt.Fprintf(stdout, "source.path: %s\n", indexed.Source.Path)
	_, _ = fmt.Fprintf(stdout, "source.ref: %s\n", indexed.Source.Ref)
	_, _ = fmt.Fprintf(stdout, "maintainers: %s\n", strings.Join(indexed.Maintainers, ", "))
	_, _ = fmt.Fprintf(stdout, "license: %s\n", indexed.License)
	_, _ = fmt.Fprintf(stdout, "trust: %s\n", indexed.Trust.Level)
	_, _ = fmt.Fprintf(stdout, "trust.reviewed_at: %s\n", indexed.Trust.ReviewedAt)
	_, _ = fmt.Fprintf(stdout, "trust.reviewer: %s\n", indexed.Trust.Reviewer)
	_, _ = fmt.Fprintf(stdout, "featured: %t\n", indexed.Featured)
	_, _ = fmt.Fprintf(stdout, "updated_at: %s\n", indexed.UpdatedAt)
	_, _ = fmt.Fprintf(stdout, "checksum: %s\n", indexed.Checksum)
	_, _ = fmt.Fprintf(stdout, "install: %s\n", installCommand)
	return nil
}

func featuredLabel(featured bool) string {
	if featured {
		return "featured"
	}
	return "-"
}

func formatCatalogResult(result registry.SearchResult) string {
	return fmt.Sprintf("%s/%s\t%s\t%s\t%s\t%s\t%s",
		result.Registry,
		result.Skill.Identity,
		result.Skill.Version,
		strings.Join(result.Skill.Targets, ","),
		result.Skill.Trust.Level,
		featuredLabel(result.Skill.Featured),
		result.Skill.Description,
	)
}

type catalogJSONResult struct {
	Registry string              `json:"registry"`
	Skill    registry.IndexSkill `json:"skill"`
}

type catalogExport struct {
	GeneratedBy string               `json:"generated_by"`
	Skills      []catalogExportSkill `json:"skills"`
	Tags        []catalogCount       `json:"tags"`
	Targets     []catalogCount       `json:"targets"`
	Namespaces  []catalogCount       `json:"namespaces"`
	Trust       []catalogCount       `json:"trust"`
}

type catalogExportSkill struct {
	Registry       string              `json:"registry"`
	Skill          registry.IndexSkill `json:"skill"`
	InstallCommand string              `json:"install_command"`
}

type infoJSONResult struct {
	Registry       string              `json:"registry"`
	Skill          registry.IndexSkill `json:"skill"`
	InstallCommand string              `json:"install_command"`
}

type catalogCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func catalogJSONResults(results []registry.SearchResult) []catalogJSONResult {
	jsonResults := make([]catalogJSONResult, 0, len(results))
	for _, result := range results {
		jsonResults = append(jsonResults, catalogJSONResult{Registry: result.Registry, Skill: result.Skill})
	}
	return jsonResults
}

func catalogExportResults(results []registry.SearchResult) catalogExport {
	skills := make([]catalogExportSkill, 0, len(results))
	for _, result := range results {
		skills = append(skills, catalogExportSkill{
			Registry:       result.Registry,
			Skill:          result.Skill,
			InstallCommand: fmt.Sprintf("skillhub install %s/%s", result.Registry, result.Skill.Identity),
		})
	}
	return catalogExport{
		GeneratedBy: "skillhub",
		Skills:      skills,
		Tags:        aggregateCatalog(results, "tags"),
		Targets:     aggregateCatalog(results, "targets"),
		Namespaces:  aggregateCatalog(results, "namespaces"),
		Trust:       aggregateCatalog(results, "trust"),
	}
}

func aggregateCatalog(results []registry.SearchResult, kind string) []catalogCount {
	countMap := map[string]int{}
	for _, result := range results {
		values := result.Skill.Tags
		switch kind {
		case "targets":
			values = result.Skill.Targets
		case "namespaces":
			values = []string{result.Skill.Namespace}
		case "trust":
			values = []string{result.Skill.Trust.Level}
		}
		for _, value := range values {
			countMap[value]++
		}
	}
	counts := make([]catalogCount, 0, len(countMap))
	for name, count := range countMap {
		counts = append(counts, catalogCount{Name: name, Count: count})
	}
	sort.Slice(counts, func(i, j int) bool {
		if counts[i].Count != counts[j].Count {
			return counts[i].Count > counts[j].Count
		}
		return counts[i].Name < counts[j].Name
	})
	return counts
}

func writeCatalogExport(outputDir string, results []registry.SearchResult) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	data := catalogExportResults(results)
	jsonFile, err := os.Create(filepath.Join(outputDir, "catalog.json"))
	if err != nil {
		return err
	}
	if err := writeJSON(jsonFile, data); err != nil {
		_ = jsonFile.Close()
		return err
	}
	if err := jsonFile.Close(); err != nil {
		return err
	}
	htmlFile, err := os.Create(filepath.Join(outputDir, "index.html"))
	if err != nil {
		return err
	}
	if err := catalogHTMLTemplate.Execute(htmlFile, data); err != nil {
		_ = htmlFile.Close()
		return err
	}
	return htmlFile.Close()
}

func writeJSON(stdout io.Writer, value any) error {
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

var catalogHTMLTemplate = template.Must(template.New("catalog").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>skill-hub catalog</title>
  <style>
    body { font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; margin: 2rem; color: #1f2937; }
    header { border-bottom: 1px solid #d1d5db; margin-bottom: 1.5rem; padding-bottom: 1rem; }
    h1 { margin: 0 0 .25rem; font-size: 1.75rem; }
    section { margin: 1.5rem 0; }
    table { border-collapse: collapse; width: 100%; }
    th, td { border-bottom: 1px solid #e5e7eb; padding: .7rem; text-align: left; vertical-align: top; }
    th { background: #f9fafb; font-weight: 600; }
    code { background: #f3f4f6; border-radius: 4px; padding: .15rem .3rem; }
    .meta { color: #4b5563; }
  </style>
</head>
<body>
  <header>
    <h1>skill-hub catalog</h1>
    <p class="meta">{{len .Skills}} skills exported by {{.GeneratedBy}}</p>
  </header>
  <section>
    <h2>Skills</h2>
    <table>
      <thead>
        <tr><th>Identity</th><th>Targets</th><th>Trust</th><th>Tags</th><th>Install</th></tr>
      </thead>
      <tbody>
      {{range .Skills}}
        <tr>
          <td><strong>{{.Registry}}/{{.Skill.Identity}}</strong><br>{{.Skill.Description}}</td>
          <td>{{range $i, $target := .Skill.Targets}}{{if $i}}, {{end}}{{$target}}{{end}}</td>
          <td>{{.Skill.Trust.Level}}</td>
          <td>{{range $i, $tag := .Skill.Tags}}{{if $i}}, {{end}}{{$tag}}{{end}}</td>
          <td><code>{{.InstallCommand}}</code></td>
        </tr>
      {{end}}
      </tbody>
    </table>
  </section>
</body>
</html>
`))

func runList(args []string, stdout io.Writer, workDir string) error {
	scope := "all"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--scope":
			i++
			if i >= len(args) {
				return fmt.Errorf("--scope requires a value")
			}
			scope = args[i]
		default:
			return fmt.Errorf(listUsage)
		}
	}
	if scope != "all" && scope != "global" && scope != "project" {
		return fmt.Errorf("unsupported scope %q", scope)
	}

	lock, err := install.LoadLock()
	if err != nil {
		return err
	}
	projectSkills, err := projectskills.DiscoverSkills(workDir)
	if err != nil {
		return err
	}

	var rows []skillListRow
	if scope == "all" || scope == "global" {
		for _, locked := range lock.Skills {
			rows = append(rows, skillListRow{
				Scope:    "global",
				Skill:    locked.DisplayIdentity(),
				Version:  locked.Version,
				Location: locked.Description,
			})
		}
	}
	if scope == "all" || scope == "project" {
		for _, discovered := range projectSkills {
			rows = append(rows, skillListRow{
				Scope:    "project",
				Skill:    discovered.Identity,
				Version:  discovered.Version,
				Location: discovered.RelPath,
			})
		}
	}
	if len(rows) == 0 {
		_, _ = fmt.Fprintln(stdout, "no skills found")
		return nil
	}
	_, _ = fmt.Fprint(stdout, formatSkillList(rows))
	return nil
}

type skillListRow struct {
	Scope    string
	Skill    string
	Version  string
	Location string
}

func formatSkillList(rows []skillListRow) string {
	headers := []string{"Scope", "Skill", "Version", "Location / Description"}
	widths := []int{len(headers[0]), len(headers[1]), len(headers[2]), len(headers[3])}
	for _, row := range rows {
		widths[0] = max(widths[0], len(row.Scope))
		widths[1] = max(widths[1], len(row.Skill))
		widths[2] = max(widths[2], len(row.Version))
		widths[3] = max(widths[3], len(row.Location))
	}

	var builder strings.Builder
	writeSkillListSeparator(&builder, widths)
	writeSkillListCells(&builder, headers, widths)
	writeSkillListSeparator(&builder, widths)
	for _, row := range rows {
		writeSkillListCells(&builder, []string{row.Scope, row.Skill, row.Version, row.Location}, widths)
	}
	writeSkillListSeparator(&builder, widths)
	return builder.String()
}

func writeSkillListSeparator(builder *strings.Builder, widths []int) {
	builder.WriteByte('+')
	for _, width := range widths {
		builder.WriteString(strings.Repeat("-", width+2))
		builder.WriteByte('+')
	}
	builder.WriteByte('\n')
}

func writeSkillListCells(builder *strings.Builder, cells []string, widths []int) {
	builder.WriteByte('|')
	for i, cell := range cells {
		_, _ = fmt.Fprintf(builder, " %-*s |", widths[i], cell)
	}
	builder.WriteByte('\n')
}

func runUpdate(stdout io.Writer) error {
	changes, err := install.UpdateAll()
	if err != nil {
		return err
	}
	if len(changes) == 0 {
		_, _ = fmt.Fprintln(stdout, "all skills are current")
		return nil
	}
	for _, change := range changes {
		_, _ = fmt.Fprintf(stdout, "updated %s %s -> %s\n", change[0], change[1], change[2])
	}
	return nil
}

func runDeploy(args []string, stdout io.Writer) error {
	if len(args) < 1 {
		return errors.New(deployUsage())
	}
	runtime := args[0]
	if runtime == "status" {
		return runDeployStatus(args[1:], stdout)
	}
	options := deploy.Options{}
	for _, arg := range args[1:] {
		switch arg {
		case "--dry-run":
			options.DryRun = true
		case "--force":
			options.Force = true
		default:
			if options.Identity != "" {
				return errors.New(deployUsage())
			}
			options.Identity = arg
		}
	}
	var deployed []deploy.Result
	var err error
	switch runtime {
	case "codex":
		deployed, err = deploy.DeployCodex(options)
	case "claude":
		deployed, err = deploy.DeployClaude(options)
	case "gemini":
		deployed, err = deploy.DeployGemini(options)
	default:
		return fmt.Errorf("unsupported runtime %q; supported runtimes: %s", runtime, supportedRuntimeList())
	}
	if err != nil {
		return withCLIHint(err)
	}
	for _, result := range deployed {
		_, _ = fmt.Fprintln(stdout, deployResultLine(result, runtime))
	}
	return nil
}

func deployResultLine(result deploy.Result, runtime string) string {
	if result.Runtime != "" {
		runtime = result.Runtime
	}
	switch result.State {
	case deploy.StateSkipped:
		return fmt.Sprintf("skipped %s to %s: %s", result.Identity, runtime, result.Reason)
	case deploy.StateConflict:
		return fmt.Sprintf("conflict %s to %s: %s", result.Identity, runtime, result.Reason)
	case deploy.StateWouldDeploy:
		return fmt.Sprintf("would deploy %s to %s", result.Identity, runtime)
	default:
		if result.DryRun {
			return fmt.Sprintf("would deploy %s to %s", result.Identity, runtime)
		}
		return fmt.Sprintf("deployed %s to %s", result.Identity, runtime)
	}
}

func runDeployStatus(args []string, stdout io.Writer) error {
	if len(args) > 1 {
		return fmt.Errorf("usage: skillhub deploy status [%s]", strings.Join(deploy.RuntimeNames(), "|"))
	}
	runtime := ""
	if len(args) == 1 {
		runtime = args[0]
	}
	statuses, err := deploy.Statuses(runtime)
	if err != nil {
		return err
	}
	for _, status := range statuses {
		_, _ = fmt.Fprintf(stdout, "%s\t%s\t%s\n", status.Identity, status.Runtime, status.State)
	}
	return nil
}

func withCLIHint(err error) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	switch {
	case strings.Contains(message, "unknown registry "):
		return fmt.Errorf("%w\nhint: run `skillhub registry list` to see configured registries", err)
	case strings.Contains(message, "target already exists"):
		return fmt.Errorf("%w\nhint: use --force to overwrite the runtime copy, or remove the target directory first", err)
	case strings.Contains(message, "no rollback history for "):
		return fmt.Errorf("%w\nhint: install a newer version or reinstall the Skill once before rolling back", err)
	default:
		return err
	}
}

func usage(stderr io.Writer) error {
	_, _ = fmt.Fprintln(stderr, rootUsage)
	_, _ = fmt.Fprintln(stderr, "run `skillhub help` for available commands")
	return fmt.Errorf("invalid command")
}
