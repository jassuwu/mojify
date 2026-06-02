# Release Runbook

Mojify publishes CLI binary releases from stable tags in the shape `v0.YYYYMMDD.BUILD`, for example `v0.20260602.145`.

This document is public-safe. It names repositories, GitHub Actions secret names, and token scope requirements, but never records token values.

## Supported Install Paths

- Homebrew: `brew install jassuwu/tap/mojify`
- GitHub Release tarballs from `jassuwu/mojify`

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

The built-in `GITHUB_TOKEN` publishes releases in `jassuwu/mojify`. `TAP_GITHUB_TOKEN` is only for pushing formula updates to `jassuwu/homebrew-tap`.

## Local Snapshot QA

Install local release tooling:

```bash
brew install goreleaser ffmpeg yt-dlp
```

Run the normal checks:

```bash
go test ./...
goreleaser check
goreleaser release --snapshot --clean
```

Inspect the generated artifacts:

```bash
ls dist
cat dist/checksums.txt
tar -tzf dist/mojify_*_darwin_arm64.tar.gz
```

Smoke-test a generated binary:

```bash
tar -xzf dist/mojify_*_darwin_arm64.tar.gz -C /tmp mojify
/tmp/mojify --version
```

Expected shape:

```text
mojify 0.0.0-SNAPSHOT-...
```

Snapshot QA must not require GitHub tokens and must not publish a GitHub Release or update the Homebrew tap.

## Stable Release

Choose a calendar build version:

```bash
git tag v0.YYYYMMDD.BUILD
```

Example:

```bash
git tag v0.20260602.145
git push origin v0.20260602.145
```

The tag workflow publishes:

- macOS arm64 tarball
- macOS amd64 tarball
- Linux arm64 tarball
- Linux amd64 tarball
- `checksums.txt`
- generated GitHub Release notes
- updated Homebrew formula in `jassuwu/homebrew-tap`

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
mojify 0.YYYYMMDD.BUILD
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
