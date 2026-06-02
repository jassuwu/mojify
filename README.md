<h1 align="center">mojify</h1>

<p align="center">
  <strong>Turn media into text.</strong>
</p>

<p align="center">
  Play videos live or export them as MP4s with color, edges, and source audio when available.
</p>

<p align="center">
  <a href="https://github.com/jassuwu/mojify/releases"><img alt="GitHub release" src="https://img.shields.io/github/v/release/jassuwu/mojify?style=flat-square"></a>
  <a href="LICENSE"><img alt="License: MIT" src="https://img.shields.io/badge/license-MIT-blue?style=flat-square"></a>
  <a href="https://github.com/jassuwu/homebrew-tap"><img alt="Homebrew tap" src="https://img.shields.io/badge/homebrew-jassuwu%2Ftap%2Fmojify-FBB040?style=flat-square&logo=homebrew&logoColor=111"></a>
  <img alt="Platforms: macOS and Linux" src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux-555?style=flat-square">
</p>

<!--
Header demo asset goes here once Mojify can generate GIF/PNG output.
-->

## Installation

```bash
brew install jassuwu/tap/mojify
```

Homebrew builds Mojify from the tagged source archive and installs the formula-declared runtime dependencies.

You can also download a macOS or Linux tarball from [GitHub Releases](https://github.com/jassuwu/mojify/releases) and place `mojify` on your `PATH`.

Windows support is WSL-only for now.

## Usage

Play a local video:

```bash
mojify play ./demo.mp4
```

Play a yt-dlp-compatible URL:

```bash
mojify play "https://www.youtube.com/watch?v=<id>"
```

Export Mojify output as MP4:

```bash
mojify export --overwrite --width 320 ./demo.mp4 ./dist/demo-mojify.mp4
```

Inspect what Mojify will derive from a source:

```bash
mojify probe ./demo.mp4
```

## What It Does

Mojify accepts local video files and yt-dlp-compatible platform URLs as source media. It turns those sources into colored character frames that can be played live or exported as MP4.

Current capabilities:

- Local video playback
- yt-dlp-compatible URL input
- Live terminal audio playback
- MP4 export with source audio when available
- Truecolor ANSI output
- Edge-aware character rendering
- `play`, `probe`, and `export` commands

## Why Mojify

Most media-to-ASCII experiments stop at the renderer. Mojify is an attempt to make that idea complete: playable, exportable, installable, and extensible, while leaning on FFmpeg and yt-dlp for the media plumbing they already do well.

## Renderer

The default renderer is built around a practical media-to-text recipe:

- luminance and intensity mapping choose the base character
- source color becomes terminal or exported frame color
- edge detection can override density characters with directional glyphs
- frame timing favors smooth playback over showing every decoded frame

Future renderer recipes may swap the character set, color strategy, or conversion rules entirely. Emoji output, custom character recipes, and still-frame/image outputs are intentionally left as future product surface, not current README promises.

## Requirements

Mojify shells out to battle-tested media tools instead of reimplementing their jobs:

- `ffmpeg` and `ffprobe` for media decoding, probing, and MP4 export
- `ffplay` for live playback audio
- `yt-dlp` for platform URL inputs

Homebrew installs declare `ffmpeg` and `yt-dlp`. Tarball installs require the runtime tools to be installed separately.

## Roadmap

Planned or likely follow-up work:

- GIF, PNG, and still-image outputs
- a Mojify-generated README header demo GIF
- custom renderer recipes
- npm/npx wrapper around the native binary
- native Windows support beyond WSL
- a desktop app?
- a landing site?

## Development

Requirements:

- Go 1.23+
- Bun 1.3+

Build from source:

```bash
bun install
bun run build
./bin/mojify --help
```

Run tests:

```bash
bun run test
bun run fmt:check
```

Playback QA:

```bash
bun run qa:clips
bun run build
./bin/mojify play --stats dist/qa/low-motion-bars.mp4
```

Export QA:

```bash
bun run qa:clips
bun run build
bun run qa:export
```

The repeatable QA checklists live in:

- [Playback quality](docs/qa/playback-quality.md)
- [Platform media input](docs/qa/platform-media-input.md)
- [Export](docs/qa/export.md)

## Release

Mojify uses calendar + build tags in the shape `vYYYY.MM.DD.BUILD`.

See [the release runbook](docs/release.md) for snapshot QA, stable tag releases, and Homebrew tap publishing.

## License

[MIT](LICENSE)
