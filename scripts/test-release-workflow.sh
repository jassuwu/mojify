#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
workflow="${repo_root}/.github/workflows/release.yml"

ruby - "$workflow" <<'RUBY'
workflow = File.read(ARGV.fetch(0))

def assert_includes(workflow, snippet, message)
  return if workflow.include?(snippet)

  warn message
  warn "missing snippet: #{snippet}"
  exit 1
end

def assert_order(workflow, before, after, message)
  before_index = workflow.index(before)
  after_index = workflow.index(after)

  unless before_index && after_index && before_index < after_index
    warn message
    warn "expected #{before.inspect} before #{after.inspect}"
    exit 1
  end
end

def indices(workflow, snippet)
  found = []
  offset = 0

  while (index = workflow.index(snippet, offset))
    found << index
    offset = index + snippet.length
  end

  found
end

staging_commit = 'git commit -m "mojify ${{ needs.prepare.outputs.version }} formula staging"'
assert_includes(
  workflow,
  staging_commit,
  "release workflow must commit the rendered formula before tapping the local checkout",
)

bottle_tap = 'brew tap "$TAP_NAME" "$PWD/homebrew-tap"'
publish_tap = 'brew tap "$TAP_NAME" "$PWD"'
staging_indices = indices(workflow, staging_commit)
bottle_tap_index = workflow.index(bottle_tap)
publish_tap_index = workflow.index(publish_tap)

unless staging_indices.any? { |index| bottle_tap_index && index < bottle_tap_index }
  warn "bottle jobs must tap committed rendered formula state, not uncommitted checkout changes"
  warn "expected #{staging_commit.inspect} before #{bottle_tap.inspect}"
  exit 1
end

unless staging_indices.any? { |index| bottle_tap_index && publish_tap_index && index > bottle_tap_index && index < publish_tap_index }
  warn "publish job must tap committed rendered formula state, not uncommitted checkout changes"
  warn "expected a second #{staging_commit.inspect} before #{publish_tap.inspect}"
  exit 1
end

tap_formula = 'tap_formula="$(brew --repository "$TAP_NAME")/Formula/${FORMULA_NAME}.rb"'
copy_tap_formula = 'cp "$tap_formula" "Formula/${FORMULA_NAME}.rb"'
merge_metadata = 'brew bottle --merge --write --no-commit "${json_args[@]}"'
normalize_bottle_name = 'normalized="${bottle/${FORMULA_NAME}--/${FORMULA_NAME}-}"'
bottle_artifact_glob = 'bottle_tarballs=( *.bottle*.tar.gz )'

assert_includes(
  workflow,
  tap_formula,
  "release workflow must locate the Homebrew-managed tap formula after bottle metadata merge",
)

assert_includes(
  workflow,
  copy_tap_formula,
  "release workflow must copy the merged bottle block back into the checked-out tap formula",
)

assert_order(
  workflow,
  merge_metadata,
  copy_tap_formula,
  "merged bottle metadata must be copied back before the tap formula is committed",
)

assert_includes(
  workflow,
  normalize_bottle_name,
  "release workflow must normalize Homebrew bottle tarball names before upload",
)

assert_order(
  workflow,
  'brew bottle --json --root-url "${{ needs.prepare.outputs.bottle_root_url }}" "$TAP_NAME/$FORMULA_NAME"',
  normalize_bottle_name,
  "bottle tarball normalization must happen after brew bottle creates artifacts",
)

assert_order(
  workflow,
  normalize_bottle_name,
  bottle_artifact_glob,
  "bottle tarball normalization must happen before artifact upload globs are collected",
)
RUBY
