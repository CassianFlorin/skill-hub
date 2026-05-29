#!/usr/bin/env node

const path = require("node:path");
const { spawnSync } = require("node:child_process");
const { binaryName, goPlatform } = require("../lib/platform");

const target = goPlatform();
const binaryPath = path.join(__dirname, "..", "vendor", binaryName(target));
const result = spawnSync(binaryPath, process.argv.slice(2), {
  stdio: "inherit",
});

if (result.error) {
  console.error(result.error.message);
  process.exit(1);
}
process.exit(result.status === null ? 1 : result.status);
