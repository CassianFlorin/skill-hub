const assert = require("node:assert/strict");
const test = require("node:test");

const {
  archiveName,
  buildDownloadUrls,
  goPlatform,
  packageTag,
} = require("../lib/platform");
const { checksumFor } = require("../lib/checksum");

test("maps supported Node platforms to Go release targets", () => {
  assert.deepEqual(goPlatform("darwin", "arm64"), {
    goos: "darwin",
    goarch: "arm64",
    extension: ".tar.gz",
  });
  assert.deepEqual(goPlatform("linux", "x64"), {
    goos: "linux",
    goarch: "amd64",
    extension: ".tar.gz",
  });
  assert.deepEqual(goPlatform("win32", "x64"), {
    goos: "windows",
    goarch: "amd64",
    extension: ".zip",
  });
  assert.throws(() => goPlatform("freebsd", "x64"), /Unsupported platform/);
});

test("builds GitHub release artifact URLs from the npm package version", () => {
  const tag = packageTag("1.3.0");
  const name = archiveName(tag, {
    goos: "darwin",
    goarch: "arm64",
    extension: ".tar.gz",
  });

  assert.equal(tag, "v1.3.0");
  assert.equal(name, "skillhub_v1.3.0_darwin_arm64.tar.gz");
  assert.deepEqual(buildDownloadUrls("1.3.0", "darwin", "arm64"), {
    archive:
      "https://github.com/CassianFlorin/skill-hub/releases/download/v1.3.0/skillhub_v1.3.0_darwin_arm64.tar.gz",
    checksums:
      "https://github.com/CassianFlorin/skill-hub/releases/download/v1.3.0/checksums.txt",
    archiveName: "skillhub_v1.3.0_darwin_arm64.tar.gz",
    binaryName: "skillhub",
  });
});

test("reads the expected sha256 from a shasum checksums file", () => {
  const checksums = [
    "111111  skillhub_v1.3.0_darwin_amd64.tar.gz",
    "abc123  skillhub_v1.3.0_darwin_arm64.tar.gz",
    "222222  skillhub_v1.3.0_linux_amd64.tar.gz",
  ].join("\n");

  assert.equal(
    checksumFor(checksums, "skillhub_v1.3.0_darwin_arm64.tar.gz"),
    "abc123",
  );
  assert.throws(
    () => checksumFor(checksums, "skillhub_v1.3.0_windows_arm64.zip"),
    /Missing checksum/,
  );
});
