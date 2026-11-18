# deps' homebrew manager moves the urls, sha256 sums and version to each new release (one PR).
class Deps < Formula
  desc "Renovate-style dependency update pull requests from one static binary"
  homepage "https://git.jakob-moeller.cloud/jakobmoellerdev/deps"
  version "0.18.1"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "https://git.jakob-moeller.cloud/jakobmoellerdev/deps/releases/download/v0.18.1/deps_darwin_arm64.tar.xz"
      sha256 "5708b315f75e770ead79e2901aea27e6c3c6019a7c1c4198c9ae06b907a68d5a"
    end
    on_intel do
      url "https://git.jakob-moeller.cloud/jakobmoellerdev/deps/releases/download/v0.18.1/deps_darwin_amd64.tar.xz"
      sha256 "6e11376a2c16c7738c9a3823eb1a5dfc8e6863c7c3dd0cd2dd033bfde9f2700c"
    end
  end

  on_linux do
    on_arm do
      url "https://git.jakob-moeller.cloud/jakobmoellerdev/deps/releases/download/v0.18.1/deps_linux_arm64.tar.xz"
      sha256 "4375ffc6209d0abf241648fdca4664d8974f7cf313a99e9cbea76ab9e2bcbc59"
    end
    on_intel do
      url "https://git.jakob-moeller.cloud/jakobmoellerdev/deps/releases/download/v0.18.1/deps_linux_amd64.tar.xz"
      sha256 "e24294e34515c23316b4a91636e9eb0da61858c5ce32983c60def8d4cd6d2bc2"
    end
  end

  def install
    bin.install "deps"
    bash_completion.install "completions/deps.bash" => "deps"
    zsh_completion.install "completions/_deps"
    fish_completion.install "completions/deps.fish"
  end

  test do
    assert_equal "v#{version}", shell_output("#{bin}/deps version").strip
  end
end
