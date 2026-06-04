# Release Runbook

Mojify publishes installable CLI releases from stable tags in the shape `vYYYY.MM.DD.BUILD`, for example `v2026.06.02.145`.

This document is public-safe. It names repositories, GitHub Actions secret names, and token scope requirements, but never records token values.

## Supported Install Paths

- Homebrew: `brew install jassuwu/tap/mojify`
- GitHub Release tarballs from `jassuwu/mojify`

Homebrew uses a source-building formula with Homebrew bottles for supported hosts. GitHub Releases in `jassuwu/mojify` provide prebuilt macOS and Linux tarballs plus checksums for non-Homebrew installs.

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
- Homebrew bottle artifacts in `jassuwu/homebrew-tap` GitHub Releases
- formula `bottle do` metadata for supported hosts

The workflow publishes the source-repo GitHub Release first, then renders the source formula, builds and tests Homebrew bottles on native runners, uploads bottle artifacts to `jassuwu/homebrew-tap` GitHub Releases, merges bottle metadata into the tap formula, audits it, and pushes the tap update. If the tap update fails after the GitHub Release succeeds, do not delete the release automatically. Fix the tap/token problem and rerun the workflow when possible; release publishing is idempotent and re-uploads assets with `--clobber` when the release already exists. If rerun is not available, manually render and commit `Formula/mojify.rb` with `scripts/render-homebrew-formula.sh`.

## Release Smoke Test

After the workflow passes, test Homebrew installation:

```bash
brew update
brew reinstall jassuwu/tap/mojify
brew info jassuwu/tap/mojify
mojify --version
bun run qa:clips
mojify probe ./dist/qa/low-motion-bars.mp4
brew reinstall --build-from-source jassuwu/tap/mojify
mojify --version
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

Homebrew bottle installs do not require end users to install Go. Source fallback and bottle generation still use Go as the formula's build dependency.

Tarball users must install:

- `ffmpeg`
- `ffprobe`
- `ffplay`
- `yt-dlp` for platform URL inputs

Mojify does not bundle these tools and does not download them at runtime.
