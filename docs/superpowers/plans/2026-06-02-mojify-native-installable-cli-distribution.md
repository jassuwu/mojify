# Mojify Native Installable CLI Distribution Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the GoReleaser-based distribution path with native GitHub Actions and checked-in scripts that publish GitHub Release tarballs and update a source-building Homebrew formula.

**Architecture:** GitHub Actions is the release driver, while scripts in `scripts/` provide the repeatable artifact builder and Homebrew formula renderer used by both CI and release workflows. GitHub Releases publish prebuilt macOS/Linux CLI tarballs and checksums; Homebrew installs from a formula that downloads Mojify's tagged source archive and runs `go build`.

**Tech Stack:** Go 1.23, Bun/Turbo, bash, GitHub Actions, GitHub CLI, Homebrew Ruby formula, FFmpeg/yt-dlp runtime dependencies.

---

## File Structure

- `packages/core/internal/cli/cli_test.go`: update version-output tests to the new calendar + build version shape.
- `.goreleaser.yaml`: delete; GoReleaser is no longer part of release orchestration.
- `scripts/package-release.sh`: new native release packaging script for macOS/Linux target tarballs and `checksums.txt`.
- `scripts/render-homebrew-formula.sh`: new formula renderer for `jassuwu/homebrew-tap/Formula/mojify.rb`.
- `.github/workflows/ci.yml`: remove GoReleaser and add non-publishing package smoke QA using `scripts/package-release.sh`.
- `.github/workflows/release.yml`: replace GoReleaser release steps with native package, publish, source-hash, formula-render, and tap-push steps.
- `README.md`: update install wording to separate GitHub prebuilt tarballs from source-building Homebrew.
- `docs/release.md`: update the runbook for native scripts, `vYYYY.MM.DD.BUILD`, and tap recovery.
- `docs/superpowers/plans/2026-06-02-mojify-binary-release-distribution.md`: mark the old GoReleaser plan superseded by this plan.

## Task 1: Update Version Semantics

**Files:**
- Modify: `packages/core/internal/cli/cli_test.go`

- [ ] **Step 1: Update the injected version test**

In `packages/core/internal/cli/cli_test.go`, replace `TestVersionTextUsesInjectedVersion` with:

```go
func TestVersionTextUsesInjectedCalendarBuildVersion(t *testing.T) {
	oldVersion := version
	t.Cleanup(func() {
		version = oldVersion
	})

	version = "v2026.06.02.145"

	got := VersionText()
	want := "mojify 2026.06.02.145\n"
	if got != want {
		t.Fatalf("VersionText() = %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run the CLI tests**

Run:

```bash
go test ./packages/core/internal/cli
```

Expected: PASS. `Version()` already strips the leading `v`, so this is a semantic test update rather than a code change.

- [ ] **Step 3: Commit version semantics**

Run:

```bash
git add packages/core/internal/cli/cli_test.go
git -c commit.gpgsign=false commit -m "test: update release version shape"
```

## Task 2: Add Native Release Packaging Script

**Files:**
- Create: `scripts/package-release.sh`

- [ ] **Step 1: Verify the packaging script is currently absent**

Run:

```bash
test -f scripts/package-release.sh
```

Expected: FAIL because the native packaging script does not exist yet.

- [ ] **Step 2: Create `scripts/package-release.sh`**

Create `scripts/package-release.sh`:

```bash
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
```

- [ ] **Step 3: Make the script executable**

Run:

```bash
chmod +x scripts/package-release.sh
```

- [ ] **Step 4: Run snapshot packaging smoke**

Run:

```bash
scripts/package-release.sh 0.0.0-SNAPSHOT dist/release-smoke
```

Expected: PASS and prints:

```text
Wrote release artifacts to /Users/jass/repos/personal/mojify/dist/release-smoke
```

- [ ] **Step 5: Verify artifact names and checksums**

Run:

```bash
test -f dist/release-smoke/mojify_0.0.0-SNAPSHOT_darwin_amd64.tar.gz
test -f dist/release-smoke/mojify_0.0.0-SNAPSHOT_darwin_arm64.tar.gz
test -f dist/release-smoke/mojify_0.0.0-SNAPSHOT_linux_amd64.tar.gz
test -f dist/release-smoke/mojify_0.0.0-SNAPSHOT_linux_arm64.tar.gz
test -f dist/release-smoke/checksums.txt
wc -l dist/release-smoke/checksums.txt
```

Expected: PASS, and `wc -l` reports `4`.

- [ ] **Step 6: Smoke-test a local-platform packaged binary**

On Apple Silicon, run:

```bash
tmp_dir="$(mktemp -d)"
tar -xzf dist/release-smoke/mojify_0.0.0-SNAPSHOT_darwin_arm64.tar.gz -C "$tmp_dir"
"$tmp_dir/mojify" --version
rm -rf "$tmp_dir"
```

Expected:

```text
mojify 0.0.0-SNAPSHOT
```

On Intel macOS, use `mojify_0.0.0-SNAPSHOT_darwin_amd64.tar.gz`. On Linux CI, use the matching Linux archive.

- [ ] **Step 7: Commit the packaging script**

Run:

```bash
git add scripts/package-release.sh
git -c commit.gpgsign=false commit -m "build: add native release packaging"
```

## Task 3: Add Homebrew Formula Renderer

**Files:**
- Create: `scripts/render-homebrew-formula.sh`

- [ ] **Step 1: Verify the formula renderer is currently absent**

Run:

```bash
test -f scripts/render-homebrew-formula.sh
```

Expected: FAIL because the formula renderer does not exist yet.

- [ ] **Step 2: Create `scripts/render-homebrew-formula.sh`**

Create `scripts/render-homebrew-formula.sh`:

```bash
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
```

- [ ] **Step 3: Make the script executable**

Run:

```bash
chmod +x scripts/render-homebrew-formula.sh
```

- [ ] **Step 4: Render a sample formula**

Run:

```bash
scripts/render-homebrew-formula.sh v2026.06.02.145 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef > /tmp/mojify.rb
```

Expected: PASS.

- [ ] **Step 5: Validate generated Ruby syntax**

Run:

```bash
ruby -c /tmp/mojify.rb
```

Expected:

```text
Syntax OK
```

- [ ] **Step 6: Verify formula content**

Run:

```bash
rg -n 'url "https://github.com/jassuwu/mojify/archive/refs/tags/v2026\.06\.02\.145\.tar\.gz"|depends_on "go" => :build|depends_on "ffmpeg"|depends_on "yt-dlp"|std_go_args|mojify --version' /tmp/mojify.rb
```

Expected: PASS and each formula concept is matched.

- [ ] **Step 7: Commit the formula renderer**

Run:

```bash
git add scripts/render-homebrew-formula.sh
git -c commit.gpgsign=false commit -m "build: render source homebrew formula"
```

## Task 4: Replace GoReleaser Workflows With Native Actions

**Files:**
- Delete: `.goreleaser.yaml`
- Modify: `.github/workflows/ci.yml`
- Modify: `.github/workflows/release.yml`

- [ ] **Step 1: Delete GoReleaser config**

Run:

```bash
git rm .goreleaser.yaml
```

Expected: PASS.

- [ ] **Step 2: Replace `.github/workflows/ci.yml`**

Replace `.github/workflows/ci.yml` with:

```yaml
name: CI

on:
  pull_request:
  push:
    branches:
      - main

jobs:
  quality:
    runs-on: ubuntu-latest
    timeout-minutes: 10
    steps:
      - uses: actions/checkout@v4

      - uses: oven-sh/setup-bun@v2
        with:
          bun-version-file: package.json

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Install FFmpeg
        run: sudo apt-get update && sudo apt-get install -y ffmpeg

      - name: Install JS dependencies
        run: bun install --frozen-lockfile

      - name: Format check
        run: bun run fmt:check

      - name: Test
        run: bun run test

      - name: Build
        run: bun run build

      - name: Package smoke
        run: scripts/package-release.sh 0.0.0-SNAPSHOT dist/release-smoke

      - name: Version smoke
        shell: bash
        run: |
          set -euo pipefail
          tmp_dir="$(mktemp -d)"
          tar -xzf dist/release-smoke/mojify_0.0.0-SNAPSHOT_linux_amd64.tar.gz -C "$tmp_dir"
          actual="$("$tmp_dir/mojify" --version)"
          rm -rf "$tmp_dir"
          if [[ "$actual" != "mojify 0.0.0-SNAPSHOT" ]]; then
            echo "Unexpected version output: $actual" >&2
            exit 1
          fi

      - name: Artifact smoke
        shell: bash
        run: |
          set -euo pipefail
          test -f dist/release-smoke/mojify_0.0.0-SNAPSHOT_darwin_amd64.tar.gz
          test -f dist/release-smoke/mojify_0.0.0-SNAPSHOT_darwin_arm64.tar.gz
          test -f dist/release-smoke/mojify_0.0.0-SNAPSHOT_linux_amd64.tar.gz
          test -f dist/release-smoke/mojify_0.0.0-SNAPSHOT_linux_arm64.tar.gz
          test -f dist/release-smoke/checksums.txt
          line_count="$(wc -l < dist/release-smoke/checksums.txt | tr -d ' ')"
          if [[ "$line_count" != "4" ]]; then
            echo "Expected 4 checksum lines, got $line_count" >&2
            exit 1
          fi
```

- [ ] **Step 3: Replace `.github/workflows/release.yml`**

Replace `.github/workflows/release.yml` with:

```yaml
name: Release

on:
  push:
    tags:
      - "v*.*.*.*"

permissions:
  contents: write

jobs:
  release:
    name: Publish CLI release
    runs-on: ubuntu-latest
    timeout-minutes: 45
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Validate release tag
        shell: bash
        run: |
          set -euo pipefail
          if [[ ! "$GITHUB_REF_NAME" =~ ^v[0-9]{4}\.[0-9]{2}\.[0-9]{2}\.[0-9]+$ ]]; then
            echo "Invalid Mojify release tag: $GITHUB_REF_NAME" >&2
            echo "Expected vYYYY.MM.DD.BUILD, for example v2026.06.02.145" >&2
            exit 1
          fi

      - name: Install FFmpeg
        run: sudo apt-get update && sudo apt-get install -y ffmpeg

      - name: Test
        run: go test ./...

      - name: Package release artifacts
        shell: bash
        run: |
          set -euo pipefail
          version="${GITHUB_REF_NAME#v}"
          scripts/package-release.sh "$version" dist/release

      - name: Version smoke
        shell: bash
        run: |
          set -euo pipefail
          version="${GITHUB_REF_NAME#v}"
          tmp_dir="$(mktemp -d)"
          tar -xzf "dist/release/mojify_${version}_linux_amd64.tar.gz" -C "$tmp_dir"
          actual="$("$tmp_dir/mojify" --version)"
          rm -rf "$tmp_dir"
          expected="mojify ${version}"
          if [[ "$actual" != "$expected" ]]; then
            echo "Unexpected version output" >&2
            echo "Expected: $expected" >&2
            echo "Actual:   $actual" >&2
            exit 1
          fi

      - name: Publish GitHub Release
        shell: bash
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          set -euo pipefail
          version="${GITHUB_REF_NAME#v}"
          if gh release view "$GITHUB_REF_NAME" --repo "$GITHUB_REPOSITORY" >/dev/null 2>&1; then
            gh release upload "$GITHUB_REF_NAME" dist/release/* \
              --repo "$GITHUB_REPOSITORY" \
              --clobber
            gh release edit "$GITHUB_REF_NAME" \
              --repo "$GITHUB_REPOSITORY" \
              --title "Mojify ${version}"
          else
            gh release create "$GITHUB_REF_NAME" dist/release/* \
              --repo "$GITHUB_REPOSITORY" \
              --title "Mojify ${version}" \
              --generate-notes
          fi

      - name: Compute source archive checksum
        id: source
        shell: bash
        run: |
          set -euo pipefail
          archive_url="https://github.com/${GITHUB_REPOSITORY}/archive/refs/tags/${GITHUB_REF_NAME}.tar.gz"
          archive_path="$(mktemp)"
          curl -fsSL "$archive_url" -o "$archive_path"
          sha256="$(shasum -a 256 "$archive_path" | awk '{print $1}')"
          echo "sha256=$sha256" >> "$GITHUB_OUTPUT"

      - name: Checkout Homebrew tap
        uses: actions/checkout@v4
        with:
          repository: jassuwu/homebrew-tap
          token: ${{ secrets.TAP_GITHUB_TOKEN }}
          path: homebrew-tap

      - name: Render Homebrew formula
        shell: bash
        run: |
          set -euo pipefail
          mkdir -p homebrew-tap/Formula
          scripts/render-homebrew-formula.sh "$GITHUB_REF_NAME" "${{ steps.source.outputs.sha256 }}" > homebrew-tap/Formula/mojify.rb
          ruby -c homebrew-tap/Formula/mojify.rb

      - name: Set up Homebrew
        uses: Homebrew/actions/setup-homebrew@main

      - name: Validate Homebrew formula
        shell: bash
        run: |
          set -euo pipefail
          brew tap jassuwu/tap "$PWD/homebrew-tap"
          tap_dir="$(brew --repository jassuwu/tap)"
          mkdir -p "$tap_dir/Formula"
          cp homebrew-tap/Formula/mojify.rb "$tap_dir/Formula/mojify.rb"
          brew audit --formula jassuwu/tap/mojify
          brew install --build-from-source jassuwu/tap/mojify
          brew test jassuwu/tap/mojify

      - name: Push Homebrew tap update
        shell: bash
        run: |
          set -euo pipefail
          cd homebrew-tap
          git config user.name "github-actions[bot]"
          git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
          git add Formula/mojify.rb
          if git diff --cached --quiet; then
            echo "Homebrew formula already up to date."
            exit 0
          fi
          git commit -m "mojify ${GITHUB_REF_NAME#v}"
          git push
```

- [ ] **Step 4: Verify GoReleaser references are gone from live release files**

Run:

```bash
test ! -f .goreleaser.yaml
rg -n 'GoReleaser|goreleaser|brews|homebrew_casks' .github
```

Expected: first command PASS. Second command FAIL because `.github` no longer references GoReleaser.

- [ ] **Step 5: Commit workflow replacement**

Run:

```bash
git add .github/workflows/ci.yml .github/workflows/release.yml
git -c commit.gpgsign=false commit -m "ci: use native release packaging"
```

## Task 5: Update Release Docs

**Files:**
- Modify: `README.md`
- Modify: `docs/release.md`
- Modify: `docs/superpowers/plans/2026-06-02-mojify-binary-release-distribution.md`

- [ ] **Step 1: Update README installation wording**

In `README.md`, replace the installation section through the runtime requirements paragraph with:

````markdown
## Installation

Mojify is distributed through Homebrew and GitHub Releases for macOS and Linux.

### Homebrew

```bash
brew install jassuwu/tap/mojify
```

The Homebrew formula builds Mojify from the tagged source archive and installs runtime dependencies declared by the formula.

### GitHub Releases

Download the matching tarball for your platform from the [GitHub Releases](https://github.com/jassuwu/mojify/releases), then place `mojify` on your `PATH`.

Windows support is WSL-only for now. Native Windows binaries are deferred.

## Requirements

- FFmpeg and ffprobe on `PATH`
- yt-dlp on `PATH` for platform URL inputs
- ffplay on `PATH` for live playback audio

Homebrew installs declare `ffmpeg` and `yt-dlp`. Tarball installs require these tools to be installed separately.
````

- [ ] **Step 2: Replace `docs/release.md`**

Replace `docs/release.md` with:

````markdown
# Release Runbook

Mojify publishes installable CLI releases from stable tags in the shape `vYYYY.MM.DD.BUILD`, for example `v2026.06.02.145`.

This document is public-safe. It names repositories, GitHub Actions secret names, and token scope requirements, but never records token values.

## Supported Install Paths

- Homebrew: `brew install jassuwu/tap/mojify`
- GitHub Release tarballs from `jassuwu/mojify`

Homebrew uses a source-building formula. GitHub Releases provide prebuilt macOS and Linux tarballs plus checksums.

Not supported in this stage:

- npm or npx
- native Windows binaries
- Linux distro packages
- `go install`
- signed or notarized artifacts

Windows users should use Mojify inside WSL for this stage.

## One-Time Tap Setup

1. Create a public GitHub repository named `jassuwu/homebrew-tap`.
2. Use `main` as the default branch.
3. Do not add token values or private account material to this repository.
4. Create a fine-grained GitHub token that can write contents to `jassuwu/homebrew-tap`.
5. In `jassuwu/mojify`, add the token as a GitHub Actions secret named `TAP_GITHUB_TOKEN`.

The built-in `GITHUB_TOKEN` publishes releases in `jassuwu/mojify`. `TAP_GITHUB_TOKEN` is only for checking out and pushing formula updates to `jassuwu/homebrew-tap`.

## Local Snapshot QA

Install local release tooling:

```bash
brew install ffmpeg yt-dlp
```

Run the normal checks:

```bash
go test ./...
bun run fmt:check
bun run test
bun run build
```

Package release-shaped artifacts without publishing:

```bash
scripts/package-release.sh 0.0.0-SNAPSHOT dist/release-smoke
```

Inspect the generated artifacts:

```bash
ls dist/release-smoke
cat dist/release-smoke/checksums.txt
tar -tzf dist/release-smoke/mojify_0.0.0-SNAPSHOT_darwin_arm64.tar.gz
```

Smoke-test a generated binary:

```bash
tmp_dir="$(mktemp -d)"
tar -xzf dist/release-smoke/mojify_0.0.0-SNAPSHOT_darwin_arm64.tar.gz -C "$tmp_dir"
"$tmp_dir/mojify" --version
rm -rf "$tmp_dir"
```

Expected:

```text
mojify 0.0.0-SNAPSHOT
```

Snapshot QA must not require GitHub tokens and must not publish a GitHub Release or update the Homebrew tap.

## Stable Release

Choose a calendar + build version:

```bash
git tag vYYYY.MM.DD.BUILD
```

Example:

```bash
git tag v2026.06.02.145
git push origin v2026.06.02.145
```

The tag workflow publishes:

- macOS arm64 tarball
- macOS amd64 tarball
- Linux arm64 tarball
- Linux amd64 tarball
- `checksums.txt`
- generated GitHub Release notes
- source-building Homebrew formula in `jassuwu/homebrew-tap`

The workflow publishes the GitHub Release first, then renders, audits, source-installs, tests, and pushes the Homebrew formula. If the tap update fails after the GitHub Release succeeds, do not delete the release automatically. Fix the tap/token problem and rerun the workflow when possible; release publishing is idempotent and re-uploads assets with `--clobber` when the release already exists. If rerun is not available, manually render and commit `Formula/mojify.rb` with `scripts/render-homebrew-formula.sh`.

## Release Smoke Test

After the workflow passes, test Homebrew installation:

```bash
brew update
brew install jassuwu/tap/mojify
mojify --version
bun run qa:clips
mojify probe ./dist/qa/low-motion-bars.mp4
```

Expected version shape:

```text
mojify YYYY.MM.DD.BUILD
```

If the formula update fails, check:

- `TAP_GITHUB_TOKEN` exists in `jassuwu/mojify` repository secrets.
- The token can write contents to `jassuwu/homebrew-tap`.
- The tap repository exists and has a `main` branch.

## Runtime Dependencies

Homebrew installs declare:

- `ffmpeg`
- `yt-dlp`

Tarball users must install:

- `ffmpeg`
- `ffprobe`
- `ffplay`
- `yt-dlp` for platform URL inputs

Mojify does not bundle these tools and does not download them at runtime.
````

- [ ] **Step 3: Mark the old implementation plan superseded**

At the top of `docs/superpowers/plans/2026-06-02-mojify-binary-release-distribution.md`, immediately after the title, add:

```markdown
> **Superseded:** This GoReleaser-based plan was superseded by `docs/superpowers/plans/2026-06-02-mojify-native-installable-cli-distribution.md`. Do not use this plan for implementation.
```

- [ ] **Step 4: Verify stale release wording is gone from current docs**

Run:

```bash
rg -n 'v0\.20260602|v0\.YYYYMMDD|0\.YYYYMMDD|brews|homebrew_casks|goreleaser' README.md docs/release.md docs/adr/0027-start-binary-release-distribution-with-github-releases-and-homebrew.md CONTEXT.md .github
```

Expected: FAIL. The old superseded plan may still contain stale wording, and the ADR/glossary may mention `GoReleaser` by name to explain that it is not used, but current docs and workflows should not contain old version examples, old Homebrew config names, or lowercase release-tooling references.

- [ ] **Step 5: Commit release docs**

Run:

```bash
git add README.md docs/release.md docs/superpowers/plans/2026-06-02-mojify-binary-release-distribution.md
git -c commit.gpgsign=false commit -m "docs: update native release runbook"
```

## Task 6: Final Verification

**Files:**
- Verify all changed files.

- [ ] **Step 1: Check formatting**

Run:

```bash
bun run fmt:check
```

Expected: PASS.

- [ ] **Step 2: Run tests**

Run:

```bash
bun run test
```

Expected: PASS.

- [ ] **Step 3: Run build**

Run:

```bash
bun run build
```

Expected: PASS.

- [ ] **Step 4: Run package smoke**

Run:

```bash
scripts/package-release.sh 0.0.0-SNAPSHOT dist/release-smoke
```

Expected: PASS and writes four tarballs plus `checksums.txt`.

- [ ] **Step 5: Smoke-test packaged binary version**

On Apple Silicon, run:

```bash
tmp_dir="$(mktemp -d)"
tar -xzf dist/release-smoke/mojify_0.0.0-SNAPSHOT_darwin_arm64.tar.gz -C "$tmp_dir"
"$tmp_dir/mojify" --version
rm -rf "$tmp_dir"
```

Expected:

```text
mojify 0.0.0-SNAPSHOT
```

- [ ] **Step 6: Validate generated Homebrew formula**

Run:

```bash
scripts/render-homebrew-formula.sh v2026.06.02.145 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef > /tmp/mojify.rb
ruby -c /tmp/mojify.rb
```

Expected:

```text
Syntax OK
```

- [ ] **Step 7: Verify GoReleaser is removed from active release tooling**

Run:

```bash
test ! -f .goreleaser.yaml
rg -n 'GoReleaser|goreleaser|brews|homebrew_casks' .github
rg -n 'v0\.20260602|v0\.YYYYMMDD|0\.YYYYMMDD|brews|homebrew_casks|goreleaser' README.md docs/release.md docs/adr/0027-start-binary-release-distribution-with-github-releases-and-homebrew.md CONTEXT.md
```

Expected: first command PASS, second and third commands FAIL with no matches. Uppercase `GoReleaser` may still appear in ADR/glossary text only to record the rejected path.

- [ ] **Step 8: Commit final plan/doc glossary state if needed**

If `CONTEXT.md`, `docs/adr/0027-start-binary-release-distribution-with-github-releases-and-homebrew.md`, or this plan are still uncommitted, run:

```bash
git add CONTEXT.md docs/adr/0027-start-binary-release-distribution-with-github-releases-and-homebrew.md docs/superpowers/plans/2026-06-02-mojify-native-installable-cli-distribution.md
git -c commit.gpgsign=false commit -m "docs: plan native installable cli distribution"
```

## Self-Review

- Spec coverage: The plan covers removing GoReleaser, changing tags to `vYYYY.MM.DD.BUILD`, preserving GitHub tarballs/checksums, adding script-based packaging with guarded `dist/` output, adding source-building Homebrew formula generation and validation, updating CI/release workflows with idempotent release publishing, documenting tap recovery, updating README/runbook/ADR/glossary, and marking the old plan superseded.
- Placeholder scan: No red-flag placeholder patterns remain; every code/script/workflow change includes concrete content.
- Type and name consistency: Script names, workflow paths, tag regexes, archive names, formula dependency names, and version examples are consistent across tasks.
