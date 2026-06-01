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
const workflow = read(".github/workflows/pages.yml");

includes(index, "skill-hub", "site/index.html");
includes(index, "brew tap CassianFlorin/tap &amp;&amp; brew install skillhub", "site/index.html");
includes(index, "Managed store", "site/index.html");
includes(index, "Project discovered", "site/index.html");
includes(index, "Runtime copy", "site/index.html");
includes(index, "skillhub update --dry-run", "site/index.html");
includes(index, "skillhub deploy status", "site/index.html");
includes(index, "styles.css", "site/index.html");
includes(styles, ".hero", "site/styles.css");
includes(workflow, "actions/deploy-pages", ".github/workflows/pages.yml");
includes(workflow, "path: site", ".github/workflows/pages.yml");

console.log("GitHub Pages files verified.");
