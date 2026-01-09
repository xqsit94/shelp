class Shelp < Formula
  desc "AI-powered shell assistant - convert natural language to commands"
  homepage "https://github.com/xqsit94/shelp"
  version "0.1.0-alpha"
  license "MIT"

  on_macos do
    on_intel do
      url "https://github.com/xqsit94/shelp/releases/download/v#{version}/shelp-darwin-amd64.tar.gz"
      sha256 "PLACEHOLDER"

      def install
        bin.install "shelp"
      end
    end

    on_arm do
      url "https://github.com/xqsit94/shelp/releases/download/v#{version}/shelp-darwin-arm64.tar.gz"
      sha256 "PLACEHOLDER"

      def install
        bin.install "shelp"
      end
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/xqsit94/shelp/releases/download/v#{version}/shelp-linux-amd64.tar.gz"
      sha256 "PLACEHOLDER"

      def install
        bin.install "shelp"
      end
    end

    on_arm do
      url "https://github.com/xqsit94/shelp/releases/download/v#{version}/shelp-linux-arm64.tar.gz"
      sha256 "PLACEHOLDER"

      def install
        bin.install "shelp"
      end
    end
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/shelp --version")
  end
end
