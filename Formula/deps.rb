# deps' homebrew manager moves the urls, sha256 sums and version to each new release (one PR).
class Deps < Formula
  desc "Renovate-style dependency update pull requests from one static binary"
  homepage "https://git.jakob-moeller.cloud/jakobmoellerdev/deps"
  version "0.17.0"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "https://git.jakob-moeller.cloud/jakobmoellerdev/deps/releases/download/v0.17.0/deps_darwin_arm64.tar.xz"
      sha256 "fe18260cf7e21bc8c71a627202461287d005bcc713d4d38fc7af7b17e9054aa2"
    end
    on_intel do
      url "https://git.jakob-moeller.cloud/jakobmoellerdev/deps/releases/download/v0.17.0/deps_darwin_amd64.tar.xz"
      sha256 "abe95be77bf79110cf78fefe9dbeff0fe7113ed5a1e9fc7b385c46f1cd694091"
    end
  end

  on_linux do
    on_arm do
      url "https://git.jakob-moeller.cloud/jakobmoellerdev/deps/releases/download/v0.17.0/deps_linux_arm64.tar.xz"
      sha256 "76e6622ebcb597e1fad3a95f3a776d92981ebe50bb756eb830e90cb8d1d19f95"
    end
    on_intel do
      url "https://git.jakob-moeller.cloud/jakobmoellerdev/deps/releases/download/v0.17.0/deps_linux_amd64.tar.xz"
      sha256 "a4751fa7b4df463007a055063b8621f2bef824d675de4989cab179ae830b21dd"
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
