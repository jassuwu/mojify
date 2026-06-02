#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: $0 <version> <output-dir>" >&2
}

if [[ $# -ne 2 ]]; then
  usage
  exit 2
fi

version="$1"
out_dir="$2"

if [[ ! "$version" =~ ^([0-9]{4}\.[0-9]{2}\.[0-9]{2}\.[0-9]+|0\.0\.0-SNAPSHOT)$ ]]; then
  echo "invalid Mojify version: $version" >&2
  echo "expected YYYY.MM.DD.BUILD or 0.0.0-SNAPSHOT" >&2
  exit 1
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"

case "$out_dir" in
  dist/?*) ;;
  *)
    echo "output-dir must be a named relative path inside dist/" >&2
    exit 1
    ;;
esac

case "$out_dir" in
  ../*|*/../*|*/..|*/./*|*/.|*//*)
    echo "output-dir must not contain parent, current, or empty directory segments" >&2
    exit 1
    ;;
esac

out_dir_abs="$repo_root/$out_dir"

commit="${GITHUB_SHA:-}"
if [[ -z "$commit" ]] && command -v git >/dev/null 2>&1; then
  commit="$(git -C "$repo_root" rev-parse HEAD 2>/dev/null || true)"
fi

build_date="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
ldflags="-s -w -X github.com/jass/mojify/packages/core/internal/cli.version=${version} -X github.com/jass/mojify/packages/core/internal/cli.commit=${commit} -X github.com/jass/mojify/packages/core/internal/cli.date=${build_date}"

targets=(
  "darwin/amd64"
  "darwin/arm64"
  "linux/amd64"
  "linux/arm64"
)

rm -rf "$out_dir_abs"
mkdir -p "$out_dir_abs"

tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

for target in "${targets[@]}"; do
  goos="${target%/*}"
  goarch="${target#*/}"
  artifact="mojify_${version}_${goos}_${goarch}"
  stage_dir="$tmp_dir/$artifact"

  mkdir -p "$stage_dir"
  (
    cd "$repo_root"
    GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -trimpath -ldflags "$ldflags" -o "$stage_dir/mojify" ./packages/core/cmd/mojify
  )
  cp "$repo_root/LICENSE" "$stage_dir/LICENSE"
  cp "$repo_root/README.md" "$stage_dir/README.md"

  (
    cd "$stage_dir"
    tar -czf "$out_dir_abs/${artifact}.tar.gz" mojify LICENSE README.md
  )
done

(
  cd "$out_dir_abs"
  shasum -a 256 *.tar.gz > checksums.txt
)

echo "Wrote release artifacts to $out_dir_abs"
