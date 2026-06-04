#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: $0 <tag> <source-sha256>" >&2
}

if [[ $# -ne 2 ]]; then
  usage
  exit 2
fi

tag="$1"
sha256="$2"
version="${tag#v}"

if [[ ! "$tag" =~ ^v[0-9]{4}\.[0-9]{2}\.[0-9]{2}\.[0-9]+$ ]]; then
  echo "invalid Mojify release tag: $tag" >&2
  echo "expected vYYYY.MM.DD.BUILD, for example v2026.06.02.145" >&2
  exit 1
fi

if [[ ! "$sha256" =~ ^[0-9a-f]{64}$ ]]; then
  echo "invalid source archive sha256: $sha256" >&2
  exit 1
fi

cat <<EOF
class Mojify < Formula
  desc "Terminal-first video player that renders media as colored character frames"
  homepage "https://github.com/jassuwu/mojify"
  url "https://github.com/jassuwu/mojify/archive/refs/tags/${tag}.tar.gz"
  version "${version}"
  sha256 "${sha256}"
  license "MIT"

  head "https://github.com/jassuwu/mojify.git", branch: "main"

  depends_on "go" => :build
  depends_on "ffmpeg"
  depends_on "yt-dlp"

  def install
    version_text = build.head? ? "0.0.0-dev" : version.to_s
    ldflags = "-s -w -X github.com/jass/mojify/packages/core/internal/cli.version=#{version_text}"
    system "go", "build", *std_go_args(output: bin/"mojify", ldflags: ldflags), "./packages/core/cmd/mojify"
  end

  test do
    expected = build.head? ? "mojify 0.0.0-dev" : "mojify #{version}"
    assert_match expected, shell_output("#{bin}/mojify --version")
  end
end
EOF
