const RELEASE_BASE =
  "https://github.com/CassianFlorin/skill-hub/releases/download";

function goPlatform(platform = process.platform, arch = process.arch) {
  const goos = {
    darwin: "darwin",
    linux: "linux",
    win32: "windows",
  }[platform];
  const goarch = {
    arm64: "arm64",
    x64: "amd64",
  }[arch];

  if (!goos || !goarch) {
    throw new Error(`Unsupported platform: ${platform}/${arch}`);
  }
  return {
    goos,
    goarch,
    extension: goos === "windows" ? ".zip" : ".tar.gz",
  };
}

function packageTag(version) {
  return version.startsWith("v") ? version : `v${version}`;
}

function archiveName(tag, target) {
  return `skillhub_${tag}_${target.goos}_${target.goarch}${target.extension}`;
}

function binaryName(target) {
  return target.goos === "windows" ? "skillhub.exe" : "skillhub";
}

function buildDownloadUrls(version, platform = process.platform, arch = process.arch) {
  const tag = packageTag(version);
  const target = goPlatform(platform, arch);
  const name = archiveName(tag, target);
  return {
    archive: `${RELEASE_BASE}/${tag}/${name}`,
    checksums: `${RELEASE_BASE}/${tag}/checksums.txt`,
    archiveName: name,
    binaryName: binaryName(target),
  };
}

module.exports = {
  archiveName,
  binaryName,
  buildDownloadUrls,
  goPlatform,
  packageTag,
};
