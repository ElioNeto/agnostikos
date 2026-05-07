class Agnostikos < Formula
  desc "Meta-wrapper package manager for Pacman, Nix, and Flatpak"
  homepage "https://github.com/ElioNeto/agnostikos"
  url "https://github.com/ElioNeto/agnostikos/releases/download/vVERSION/agnostikos_VERSION_linux_amd64.tar.gz"
  version "VERSION"
  sha256 "SHA256_CHECKSUM"

  def install
    bin.install "agnostic"
  end

  test do
    assert_match "AgnosticOS", shell_output("#{bin}/agnostic --version")
  end
end
