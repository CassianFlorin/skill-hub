class Skillhub < Formula
  desc "Skill package manager for AI agents"
  homepage "https://github.com/CassianFlorin/skill-hub"
  version "1.3.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/CassianFlorin/skill-hub/releases/download/v1.3.0/skillhub_v1.3.0_darwin_arm64.tar.gz"
      sha256 "63283822bb40ce157310470868710fabf6362322e65e0a7baf7f97f4e30bf41f"
    else
      url "https://github.com/CassianFlorin/skill-hub/releases/download/v1.3.0/skillhub_v1.3.0_darwin_amd64.tar.gz"
      sha256 "7246a9c05face568d687f3c93f29a74f370853ec42b50639dd16d24ff5a8d4ce"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/CassianFlorin/skill-hub/releases/download/v1.3.0/skillhub_v1.3.0_linux_arm64.tar.gz"
      sha256 "7d6c677e1f78b0a0f97dd133ddade685e633b4716557b33ebc628652ba7d7264"
    else
      url "https://github.com/CassianFlorin/skill-hub/releases/download/v1.3.0/skillhub_v1.3.0_linux_amd64.tar.gz"
      sha256 "475caddc81cc76318154fc4995cd93270e615d3095c0f245c268afeec5a7e404"
    end
  end

  def install
    bin.install "skillhub"
  end

  test do
    assert_match "skillhub v#{version}", shell_output("#{bin}/skillhub version")
  end
end
