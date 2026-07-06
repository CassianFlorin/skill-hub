class Skillhub < Formula
  desc "Skill package manager for AI agents"
  homepage "https://github.com/CassianFlorin/skill-hub"
  version "1.4.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/CassianFlorin/skill-hub/releases/download/v1.4.0/skillhub_v1.4.0_darwin_arm64.tar.gz"
      sha256 "e6a760106f49b290796f4eecac1faf7a7bab8ca26027c15cd24a9d6019a87b8e"
    else
      url "https://github.com/CassianFlorin/skill-hub/releases/download/v1.4.0/skillhub_v1.4.0_darwin_amd64.tar.gz"
      sha256 "69fdf346c7b64ebd644346e96144dfda831d85a0817b4888bbdf7c30d52f6644"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/CassianFlorin/skill-hub/releases/download/v1.4.0/skillhub_v1.4.0_linux_arm64.tar.gz"
      sha256 "3d64012f0282ed8874796badcfc089415568274ae4ee3ce3cef0a965c1c6702b"
    else
      url "https://github.com/CassianFlorin/skill-hub/releases/download/v1.4.0/skillhub_v1.4.0_linux_amd64.tar.gz"
      sha256 "f2cdd60704d8f1fe4569c4041b50328fe9460149718ffc234cfbc648e6f680f1"
    end
  end

  def install
    bin.install "skillhub"
  end

  test do
    assert_match "skillhub v#{version}", shell_output("#{bin}/skillhub version")
  end
end
