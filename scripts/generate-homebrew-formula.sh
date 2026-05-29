#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $0 <tag> <checksums-file> <output-formula>" >&2
  exit 2
fi

tag="$1"
checksums_file="$2"
output="$3"
version="${tag#v}"
repo="CassianFlorin/skill-hub"

checksum_for() {
  local artifact="$1"
  awk -v artifact="${artifact}" '$2 == artifact { print $1 }' "${checksums_file}"
}

artifact_url() {
  local artifact="$1"
  printf "https://github.com/%s/releases/download/%s/%s" "${repo}" "${tag}" "${artifact}"
}

darwin_amd64="skillhub_${tag}_darwin_amd64.tar.gz"
darwin_arm64="skillhub_${tag}_darwin_arm64.tar.gz"
linux_amd64="skillhub_${tag}_linux_amd64.tar.gz"
linux_arm64="skillhub_${tag}_linux_arm64.tar.gz"

darwin_amd64_sha="$(checksum_for "${darwin_amd64}")"
darwin_arm64_sha="$(checksum_for "${darwin_arm64}")"
linux_amd64_sha="$(checksum_for "${linux_amd64}")"
linux_arm64_sha="$(checksum_for "${linux_arm64}")"

for value in darwin_amd64_sha darwin_arm64_sha linux_amd64_sha linux_arm64_sha; do
  if [[ -z "${!value}" ]]; then
    echo "missing checksum: ${value}" >&2
    exit 1
  fi
done

mkdir -p "$(dirname "${output}")"
cat >"${output}" <<FORMULA
class Skillhub < Formula
  desc "Skill package manager for AI agents"
  homepage "https://github.com/${repo}"
  version "${version}"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "$(artifact_url "${darwin_arm64}")"
      sha256 "${darwin_arm64_sha}"
    else
      url "$(artifact_url "${darwin_amd64}")"
      sha256 "${darwin_amd64_sha}"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "$(artifact_url "${linux_arm64}")"
      sha256 "${linux_arm64_sha}"
    else
      url "$(artifact_url "${linux_amd64}")"
      sha256 "${linux_amd64_sha}"
    end
  end

  def install
    bin.install "skillhub"
  end

  test do
    assert_match "skillhub v#{version}", shell_output("#{bin}/skillhub version")
  end
end
FORMULA
