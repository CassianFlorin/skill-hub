#!/usr/bin/env node

const fs = require("node:fs");
const https = require("node:https");
const os = require("node:os");
const path = require("node:path");
const { execFileSync } = require("node:child_process");

const { checksumFor, sha256File } = require("../lib/checksum");
const { buildDownloadUrls } = require("../lib/platform");

const rootDir = path.resolve(__dirname, "..");
const vendorDir = path.join(rootDir, "vendor");
const packageJson = require("../package.json");

if (process.env.SKILLHUB_SKIP_DOWNLOAD === "1") {
  process.exit(0);
}

async function download(url, destination) {
  await fs.promises.mkdir(path.dirname(destination), { recursive: true });
  try {
    await downloadWithNode(url, destination);
  } catch (error) {
    try {
      execFileSync("curl", ["-fsSL", "-o", destination, url], {
        stdio: "ignore",
      });
    } catch {
      throw error;
    }
  }
}

async function downloadWithNode(url, destination) {
  await new Promise((resolve, reject) => {
    const request = https.get(url, (response) => {
      if (
        response.statusCode >= 300 &&
        response.statusCode < 400 &&
        response.headers.location
      ) {
        response.resume();
        download(new URL(response.headers.location, url).toString(), destination)
          .then(resolve)
          .catch(reject);
        return;
      }
      if (response.statusCode !== 200) {
        response.resume();
        reject(new Error(`Download failed: ${url} returned ${response.statusCode}`));
        return;
      }
      const file = fs.createWriteStream(destination);
      response.pipe(file);
      file.on("finish", () => file.close(resolve));
      file.on("error", reject);
    });
    request.setTimeout(30000, () => {
      request.destroy(new Error(`Download timed out: ${url}`));
    });
    request.on("error", reject);
  });
}

function extractArchive(archivePath, destination, binaryName) {
  fs.rmSync(destination, { recursive: true, force: true });
  fs.mkdirSync(destination, { recursive: true });

  if (archivePath.endsWith(".zip")) {
    const expanded = path.join(os.tmpdir(), `skillhub-${Date.now()}`);
    fs.rmSync(expanded, { recursive: true, force: true });
    fs.mkdirSync(expanded, { recursive: true });
    execFileSync("powershell.exe", [
      "-NoProfile",
      "-Command",
      "Expand-Archive",
      "-Path",
      archivePath,
      "-DestinationPath",
      expanded,
      "-Force",
    ]);
    copyExtractedBinary(expanded, destination, binaryName);
    fs.rmSync(expanded, { recursive: true, force: true });
    return;
  }

  execFileSync("tar", ["-xzf", archivePath, "-C", destination, "--strip-components=1"]);
}

function copyExtractedBinary(fromDir, destination, binaryName) {
  const found = findFile(fromDir, binaryName);
  if (!found) {
    throw new Error(`Archive did not contain ${binaryName}`);
  }
  fs.copyFileSync(found, path.join(destination, binaryName));
}

function findFile(dir, fileName) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      const nested = findFile(fullPath, fileName);
      if (nested) {
        return nested;
      }
    } else if (entry.name === fileName) {
      return fullPath;
    }
  }
  return undefined;
}

async function main() {
  const urls = buildDownloadUrls(packageJson.version);
  const tmpDir = await fs.promises.mkdtemp(path.join(os.tmpdir(), "skillhub-"));
  const archivePath = path.join(tmpDir, urls.archiveName);
  const checksumsPath = path.join(tmpDir, "checksums.txt");

  try {
    await download(urls.checksums, checksumsPath);
    await download(urls.archive, archivePath);

    const expected = checksumFor(
      fs.readFileSync(checksumsPath, "utf8"),
      urls.archiveName,
    );
    const actual = sha256File(archivePath);
    if (actual !== expected) {
      throw new Error(
        `Checksum mismatch for ${urls.archiveName}: expected ${expected}, got ${actual}`,
      );
    }

    extractArchive(archivePath, vendorDir, urls.binaryName);
    const binaryPath = path.join(vendorDir, urls.binaryName);
    fs.chmodSync(binaryPath, 0o755);
  } finally {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  }
}

main().catch((error) => {
  console.error(error.message);
  process.exit(1);
});
