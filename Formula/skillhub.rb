class Skillhub < Formula
  desc "Skill package manager for AI agents"
  homepage "https://github.com/CassianFlorin/skill-hub"
  version "1.3.11"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/CassianFlorin/skill-hub/releases/download/v1.3.11/skillhub_v1.3.11_darwin_arm64.tar.gz"
      sha256 "9f17a1a83a03007784961705232d57370edfbc54c8634c5d16f99de45bb7d5f2"
    else
      url "https://github.com/CassianFlorin/skill-hub/releases/download/v1.3.11/skillhub_v1.3.11_darwin_amd64.tar.gz"
      sha256 "92d4b57b1f8068373564d1691c9ace8a19c23884794d35c9a405dbc0eca09842"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/CassianFlorin/skill-hub/releases/download/v1.3.11/skillhub_v1.3.11_linux_arm64.tar.gz"
      sha256 "17ba652737f53cf4c395f54696d134a548cf956097446336893e4e45c339c6a0"
    else
      url "https://github.com/CassianFlorin/skill-hub/releases/download/v1.3.11/skillhub_v1.3.11_linux_amd64.tar.gz"
      sha256 "0c141539dc7b8f0827b8db4e7b9a201a935456abd8ffe746c16e0a09d07f8e9a"
    end
  end

  def install
    bin.install "skillhub"
  end

  test do
    assert_match "skillhub v#{version}", shell_output("#{bin}/skillhub version")
  end
end
