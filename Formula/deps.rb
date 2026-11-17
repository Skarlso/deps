# deps' homebrew manager moves the urls, sha256 sums and version to each new release (one PR).
class Deps < Formula
  desc "Renovate-style dependency update pull requests from one static binary"
  homepage "https://git.jakob-moeller.cloud/jakobmoellerdev/deps"
  version "0.18.0"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "https://git.jakob-moeller.cloud/jakobmoellerdev/deps/releases/download/v0.18.0/deps_darwin_arm64.tar.xz"
      sha256 "cd5ed208b46cef887f3131b5c591726c3bdad7316f2d3f3b6a2bcd71ce1c6d77"
    end
    on_intel do
      url "https://git.jakob-moeller.cloud/jakobmoellerdev/deps/releases/download/v0.18.0/deps_darwin_amd64.tar.xz"
      sha256 "2b5a28ecdeb896767edaf98a489bd637a5a8829d739d0f600376d7628b542d45"
    end
  end

  on_linux do
    on_arm do
      url "https://git.jakob-moeller.cloud/jakobmoellerdev/deps/releases/download/v0.18.0/deps_linux_arm64.tar.xz"
      sha256 "932535cf5dae42695156abb88ce416ec6b70cdbbea3b1b9db68fbebb18d54d7b"
    end
    on_intel do
      url "https://git.jakob-moeller.cloud/jakobmoellerdev/deps/releases/download/v0.18.0/deps_linux_amd64.tar.xz"
      sha256 "0f326ae65004065d8a7943061f3afc30e51a6f0b99d74a4f18fcee49e93b9ecf"
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
