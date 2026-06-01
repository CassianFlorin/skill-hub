const translations = {
  "en": {
    metaTitle: "skill-hub | Skill package manager for AI agents",
    metaDescription: "skill-hub is a Skill package manager for AI agents. Install, update, roll back, and deploy Skills across Codex, Claude, and Gemini.",
    "language.aria": "Language",
    "nav.aria": "Primary navigation",
    "nav.model": "Model",
    "nav.features": "Features",
    "nav.workflow": "Workflow",
    "actions.aria": "Primary actions",
    "hero.eyebrow": "Skill package manager for AI agents",
    "hero.title": "Manage Skills without guessing what will change.",
    "hero.lead": "Install, version, update, roll back, and deploy Skills across Codex, Claude, and Gemini with explicit boundaries between managed packages and runtime copies.",
    "hero.install": "Install skillhub",
    "hero.docs": "Read the docs",
    "hero.docsHref": "https://github.com/CassianFlorin/skill-hub#quick-start",
    "terminal.aria": "skillhub command preview",
    "terminal.title": "local skill state",
    "terminal.updateOutput": "would update official/git-commit-cn 0.1.0 -&gt; 0.1.1<br>updates managed store only; runtime copies will not be changed",
    "terminal.deployOutput": "official/git-commit-cn&nbsp;&nbsp;codex&nbsp;&nbsp;drifted<br>official/git-commit-cn&nbsp;&nbsp;claude&nbsp;&nbsp;missing",
    "model.title": "Three layers. No hidden overwrites.",
    "model.lead": "skill-hub makes the local Skill model explicit before you run update, deploy, or force deploy.",
    "model.managed.title": "Managed store",
    "model.managed.body": "Skills in <code>$SKILLHUB_HOME/installed</code> and <code>skillhub.lock</code> are tracked, updated, and rolled back by skill-hub.",
    "model.project.title": "Project discovered",
    "model.project.body": "Skills found under project roots are visible in list and TUI views, but are not automatically adopted into the managed store.",
    "model.runtime.title": "Runtime copy",
    "model.runtime.body": "Codex, Claude, and Gemini load these copies. Only <code>skillhub deploy</code> changes this layer.",
    "features.title": "Built for local control first.",
    "features.lead": "Use one CLI to discover packages, manage versions, inspect drift, and deploy into the agent runtime you actually use.",
    "features.registry.title": "Registry discovery",
    "features.registry.body": "Browse local and Git-backed Skill registries, then inspect metadata before installing.",
    "features.updates.title": "Safe updates",
    "features.updates.body": "Preview update plans with <code>skillhub update --dry-run</code> and roll back managed copies when needed.",
    "features.deploy.title": "Runtime deploy",
    "features.deploy.body": "Copy selected Skills into Codex, Claude, or Gemini and use status checks to spot missing or drifted copies.",
    "features.tui.title": "Terminal UI",
    "features.tui.body": "Open <code>skillhub tui</code> to browse local Skills, catalog results, deployment status, and operation logs.",
    "features.catalog.title": "Catalog export",
    "features.catalog.body": "Publish static marketplace snapshots with searchable package metadata for teams or public catalogs.",
    "features.help.title": "Multilingual help",
    "features.help.body": "Command help is available in English, Simplified Chinese, and Traditional Chinese.",
    "workflow.title": "Install to deploy in four steps.",
    "workflow.lead": "The core workflow stays terminal-native and auditable.",
    "workflow.init.title": "1. Initialize",
    "workflow.init.body": "Set up registries and local state.",
    "workflow.discover.title": "2. Discover",
    "workflow.discover.body": "Search catalog entries and inspect package metadata.",
    "workflow.install.title": "3. Install",
    "workflow.install.body": "Track versions, source refs, checksums, and history.",
    "workflow.deploy.title": "4. Deploy",
    "workflow.deploy.body": "Copy managed Skills into runtime directories.",
    "cta.aria": "Release links",
    "cta.title": "Get the latest release.",
    "cta.lead": "Install with Homebrew, npm, or download binaries from GitHub Releases.",
    "cta.release": "Latest release",
    "cta.npm": "npm package"
  },
  "zh-CN": {
    metaTitle: "skill-hub | 面向 AI Agent 的 Skill 包管理器",
    metaDescription: "skill-hub 是面向 AI Agent 的 Skill 包管理器，可安装、更新、回滚并部署 Skills 到 Codex、Claude 和 Gemini。",
    "language.aria": "语言",
    "nav.aria": "主导航",
    "nav.model": "模型",
    "nav.features": "能力",
    "nav.workflow": "流程",
    "actions.aria": "主要操作",
    "hero.eyebrow": "面向 AI Agent 的 Skill 包管理器",
    "hero.title": "管理 Skills，不再猜测哪些内容会被改变。",
    "hero.lead": "安装、版本管理、更新、回滚，并将 Skills 部署到 Codex、Claude 和 Gemini；明确区分托管包与运行时副本的边界。",
    "hero.install": "安装 skillhub",
    "hero.docs": "阅读文档",
    "hero.docsHref": "https://github.com/CassianFlorin/skill-hub/blob/master/README.zh-CN.md",
    "terminal.aria": "skillhub 命令预览",
    "terminal.title": "本地 Skill 状态",
    "terminal.updateOutput": "将更新 official/git-commit-cn 0.1.0 -&gt; 0.1.1<br>只更新托管存储；运行时副本不会被改变",
    "terminal.deployOutput": "official/git-commit-cn&nbsp;&nbsp;codex&nbsp;&nbsp;有漂移<br>official/git-commit-cn&nbsp;&nbsp;claude&nbsp;&nbsp;缺失",
    "model.title": "三层模型，没有隐藏覆盖。",
    "model.lead": "skill-hub 会在你执行 update、deploy 或 force deploy 前，把本地 Skill 模型讲清楚。",
    "model.managed.title": "托管存储",
    "model.managed.body": "<code>$SKILLHUB_HOME/installed</code> 和 <code>skillhub.lock</code> 中的 Skills 由 skill-hub 跟踪、更新和回滚。",
    "model.project.title": "项目发现",
    "model.project.body": "项目根目录下发现的 Skills 会出现在 list 和 TUI 视图中，但不会自动纳入托管存储。",
    "model.runtime.title": "运行时副本",
    "model.runtime.body": "Codex、Claude 和 Gemini 实际加载这些副本。只有 <code>skillhub deploy</code> 会修改这一层。",
    "features.title": "优先服务本地控制。",
    "features.lead": "用一个 CLI 发现包、管理版本、检查漂移，并部署到你实际使用的 Agent 运行时。",
    "features.registry.title": "Registry 发现",
    "features.registry.body": "浏览本地和 Git-backed Skill registry，并在安装前检查元数据。",
    "features.updates.title": "安全更新",
    "features.updates.body": "用 <code>skillhub update --dry-run</code> 预览更新计划，需要时回滚托管副本。",
    "features.deploy.title": "运行时部署",
    "features.deploy.body": "将选定 Skills 复制到 Codex、Claude 或 Gemini，并用状态检查发现缺失或漂移的副本。",
    "features.tui.title": "终端 UI",
    "features.tui.body": "打开 <code>skillhub tui</code> 浏览本地 Skills、目录结果、部署状态和操作日志。",
    "features.catalog.title": "目录导出",
    "features.catalog.body": "发布带可搜索包元数据的静态 marketplace 快照，供团队或公开目录使用。",
    "features.help.title": "多语言帮助",
    "features.help.body": "命令帮助支持英语、简体中文和繁体中文。",
    "workflow.title": "从安装到部署，四步完成。",
    "workflow.lead": "核心流程保持终端原生，并且可审计。",
    "workflow.init.title": "1. 初始化",
    "workflow.init.body": "设置 registry 和本地状态。",
    "workflow.discover.title": "2. 发现",
    "workflow.discover.body": "搜索目录条目并检查包元数据。",
    "workflow.install.title": "3. 安装",
    "workflow.install.body": "跟踪版本、来源 ref、校验和与历史记录。",
    "workflow.deploy.title": "4. 部署",
    "workflow.deploy.body": "把托管 Skills 复制到运行时目录。",
    "cta.aria": "发布链接",
    "cta.title": "获取最新版本。",
    "cta.lead": "使用 Homebrew、npm 安装，或从 GitHub Releases 下载二进制文件。",
    "cta.release": "最新 release",
    "cta.npm": "npm 包"
  },
  "zh-TW": {
    metaTitle: "skill-hub | 面向 AI Agent 的 Skill 套件管理器",
    metaDescription: "skill-hub 是面向 AI Agent 的 Skill 套件管理器，可安裝、更新、回滾並部署 Skills 到 Codex、Claude 和 Gemini。",
    "language.aria": "語言",
    "nav.aria": "主導覽",
    "nav.model": "模型",
    "nav.features": "能力",
    "nav.workflow": "流程",
    "actions.aria": "主要操作",
    "hero.eyebrow": "面向 AI Agent 的 Skill 套件管理器",
    "hero.title": "管理 Skills，不再猜測哪些內容會被改變。",
    "hero.lead": "安裝、版本管理、更新、回滾，並將 Skills 部署到 Codex、Claude 和 Gemini；明確區分託管套件與執行時副本的邊界。",
    "hero.install": "安裝 skillhub",
    "hero.docs": "閱讀文件",
    "hero.docsHref": "https://github.com/CassianFlorin/skill-hub/blob/master/README.zh-TW.md",
    "terminal.aria": "skillhub 指令預覽",
    "terminal.title": "本地 Skill 狀態",
    "terminal.updateOutput": "將更新 official/git-commit-cn 0.1.0 -&gt; 0.1.1<br>只更新託管儲存；執行時副本不會被改變",
    "terminal.deployOutput": "official/git-commit-cn&nbsp;&nbsp;codex&nbsp;&nbsp;有漂移<br>official/git-commit-cn&nbsp;&nbsp;claude&nbsp;&nbsp;缺失",
    "model.title": "三層模型，沒有隱藏覆蓋。",
    "model.lead": "skill-hub 會在你執行 update、deploy 或 force deploy 前，把本地 Skill 模型說清楚。",
    "model.managed.title": "託管儲存",
    "model.managed.body": "<code>$SKILLHUB_HOME/installed</code> 和 <code>skillhub.lock</code> 中的 Skills 由 skill-hub 追蹤、更新和回滾。",
    "model.project.title": "專案發現",
    "model.project.body": "專案根目錄下發現的 Skills 會出現在 list 和 TUI 視圖中，但不會自動納入託管儲存。",
    "model.runtime.title": "執行時副本",
    "model.runtime.body": "Codex、Claude 和 Gemini 實際載入這些副本。只有 <code>skillhub deploy</code> 會修改這一層。",
    "features.title": "優先服務本地控制。",
    "features.lead": "用一個 CLI 發現套件、管理版本、檢查漂移，並部署到你實際使用的 Agent 執行時。",
    "features.registry.title": "Registry 發現",
    "features.registry.body": "瀏覽本地和 Git-backed Skill registry，並在安裝前檢查中繼資料。",
    "features.updates.title": "安全更新",
    "features.updates.body": "用 <code>skillhub update --dry-run</code> 預覽更新計畫，需要時回滾託管副本。",
    "features.deploy.title": "執行時部署",
    "features.deploy.body": "將選定 Skills 複製到 Codex、Claude 或 Gemini，並用狀態檢查發現缺失或漂移的副本。",
    "features.tui.title": "終端 UI",
    "features.tui.body": "打開 <code>skillhub tui</code> 瀏覽本地 Skills、目錄結果、部署狀態和操作日誌。",
    "features.catalog.title": "目錄匯出",
    "features.catalog.body": "發布帶可搜尋套件中繼資料的靜態 marketplace 快照，供團隊或公開目錄使用。",
    "features.help.title": "多語言幫助",
    "features.help.body": "指令幫助支援英語、簡體中文和繁體中文。",
    "workflow.title": "從安裝到部署，四步完成。",
    "workflow.lead": "核心流程保持終端原生，並且可稽核。",
    "workflow.init.title": "1. 初始化",
    "workflow.init.body": "設定 registry 和本地狀態。",
    "workflow.discover.title": "2. 發現",
    "workflow.discover.body": "搜尋目錄條目並檢查套件中繼資料。",
    "workflow.install.title": "3. 安裝",
    "workflow.install.body": "追蹤版本、來源 ref、校驗和與歷史記錄。",
    "workflow.deploy.title": "4. 部署",
    "workflow.deploy.body": "把託管 Skills 複製到執行時目錄。",
    "cta.aria": "發布連結",
    "cta.title": "取得最新版本。",
    "cta.lead": "使用 Homebrew、npm 安裝，或從 GitHub Releases 下載二進位檔。",
    "cta.release": "最新 release",
    "cta.npm": "npm 套件"
  }
};

const supportedLanguages = Object.keys(translations);
const defaultLanguage = "en";

function normalizeLanguage(language) {
  const normalized = String(language || "").replace("_", "-").toLowerCase();
  if (normalized.startsWith("zh-tw") || normalized.startsWith("zh-hk") || normalized.startsWith("zh-hant")) {
    return "zh-TW";
  }
  if (normalized.startsWith("zh-cn") || normalized.startsWith("zh-sg") || normalized.startsWith("zh-hans") || normalized === "zh") {
    return "zh-CN";
  }
  if (normalized.startsWith("en")) {
    return "en";
  }
  return "";
}

function storedLanguage() {
  try {
    return localStorage.getItem("skillhub-page-language") || "";
  } catch {
    return "";
  }
}

function saveLanguage(language) {
  try {
    localStorage.setItem("skillhub-page-language", language);
  } catch {
    // Browsers can disable storage in privacy modes. URL language still works.
  }
}

function languageFromLocation() {
  const params = new URLSearchParams(window.location.search);
  return normalizeLanguage(params.get("lang"));
}

function preferredLanguage() {
  return languageFromLocation()
    || normalizeLanguage(storedLanguage())
    || normalizeLanguage(navigator.language)
    || defaultLanguage;
}

function setText(selector, applyValue) {
  document.querySelectorAll(selector).forEach((element) => {
    const key = element.getAttribute(selector.slice(1, -1));
    const value = translations[currentLanguage][key];
    if (value !== undefined) {
      applyValue(element, value);
    }
  });
}

let currentLanguage = defaultLanguage;

function applyLanguage(language) {
  currentLanguage = supportedLanguages.includes(language) ? language : defaultLanguage;
  const dictionary = translations[currentLanguage];

  document.documentElement.lang = currentLanguage;
  document.title = dictionary.metaTitle;

  const description = document.querySelector('meta[name="description"]');
  if (description) {
    description.setAttribute("content", dictionary.metaDescription);
  }

  setText("[data-i18n]", (element, value) => {
    element.textContent = value;
  });
  setText("[data-i18n-html]", (element, value) => {
    element.innerHTML = value;
  });
  setText("[data-i18n-aria]", (element, value) => {
    element.setAttribute("aria-label", value);
  });
  setText("[data-i18n-href]", (element, value) => {
    element.setAttribute("href", value);
  });

  document.querySelectorAll("[data-lang-option]").forEach((button) => {
    button.setAttribute("aria-pressed", String(button.dataset.langOption === currentLanguage));
  });
}

function updateUrlLanguage(language) {
  const url = new URL(window.location.href);
  if (language === defaultLanguage) {
    url.searchParams.delete("lang");
  } else {
    url.searchParams.set("lang", language);
  }
  history.replaceState({}, "", url);
}

document.querySelectorAll("[data-lang-option]").forEach((button) => {
  button.addEventListener("click", () => {
    const language = normalizeLanguage(button.dataset.langOption);
    saveLanguage(language);
    applyLanguage(language);
    updateUrlLanguage(language);
  });
});

applyLanguage(preferredLanguage());
