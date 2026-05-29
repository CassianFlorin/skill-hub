const crypto = require("node:crypto");
const fs = require("node:fs");

function checksumFor(checksums, archiveName) {
  for (const line of checksums.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed) {
      continue;
    }
    const [hash, file] = trimmed.split(/\s+/, 2);
    if (file === archiveName) {
      return hash;
    }
  }
  throw new Error(`Missing checksum for ${archiveName}`);
}

function sha256File(filePath) {
  const hash = crypto.createHash("sha256");
  hash.update(fs.readFileSync(filePath));
  return hash.digest("hex");
}

module.exports = {
  checksumFor,
  sha256File,
};
