import { readFileSync } from "node:fs";

function read(path) {
  try {
    return readFileSync(path, "utf8");
  } catch (error) {
    throw new Error(`Missing ${path}: ${error.message}`);
  }
}

function includes(content, text, path) {
  if (!content.includes(text)) {
    throw new Error(`${path} must include ${JSON.stringify(text)}`);
  }
}

const index = read("site/index.html");
const styles = read("site/styles.css");
const i18n = read("site/i18n.js");
const workflow = read(".github/workflows/pages.yml");

includes(index, "skill-hub", "site/index.html");
includes(index, "brew tap CassianFlorin/tap &amp;&amp; brew install skillhub", "site/index.html");
includes(index, "Managed store", "site/index.html");
includes(index, "Project discovered", "site/index.html");
includes(index, "Runtime copy", "site/index.html");
includes(index, "skillhub update --dry-run", "site/index.html");
includes(index, "skillhub deploy status", "site/index.html");
includes(index, "styles.css", "site/index.html");
includes(index, "i18n.js", "site/index.html");
includes(index, 'data-lang-option="en"', "site/index.html");
includes(index, 'data-lang-option="zh-CN"', "site/index.html");
includes(index, 'data-lang-option="zh-TW"', "site/index.html");
includes(styles, ".hero", "site/styles.css");
includes(styles, ".language-switcher", "site/styles.css");
includes(i18n, "const translations", "site/i18n.js");
includes(i18n, '"en"', "site/i18n.js");
includes(i18n, '"zh-CN"', "site/i18n.js");
includes(i18n, '"zh-TW"', "site/i18n.js");
includes(i18n, "管理 Skills，不再猜测哪些内容会被改变。", "site/i18n.js");
includes(i18n, "管理 Skills，不再猜測哪些內容會被改變。", "site/i18n.js");
includes(i18n, "README.zh-CN.md", "site/i18n.js");
includes(i18n, "README.zh-TW.md", "site/i18n.js");
includes(workflow, "actions/deploy-pages", ".github/workflows/pages.yml");
includes(workflow, "path: site", ".github/workflows/pages.yml");

console.log("GitHub Pages files verified.");
