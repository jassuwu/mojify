#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tag="v2026.06.04.4"
sha256="e818836fa40a9d448292df0594a48d22bad0d18719f65c0514480124f6da4346"
formula_path="$(mktemp)"
trap 'rm -f "$formula_path"' EXIT

"${repo_root}/scripts/render-homebrew-formula.sh" "$tag" "$sha256" > "$formula_path"
ruby -c "$formula_path" >/dev/null

if grep -q '^  version "' "$formula_path"; then
  echo "rendered Homebrew formula must not emit an explicit version line" >&2
  echo "Homebrew infers Mojify calendar-build versions from the release URL, and audit rejects redundant explicit versions." >&2
  exit 1
fi

grep -q "refs/tags/${tag}.tar.gz" "$formula_path"
grep -q "$sha256" "$formula_path"
