<h1 align="center">mojify</h1>

<p align="center">
  <strong>Turn media into text.</strong>
</p>

<p align="center">
  Play videos live or export colored, edge-aware Mojify output as video, animated, still-image, or text files.
</p>

<p align="center">
  <a href="https://github.com/jassuwu/mojify/releases"><img alt="GitHub release" src="https://img.shields.io/github/v/release/jassuwu/mojify?style=flat-square"></a>
  <a href="LICENSE"><img alt="License: MIT" src="https://img.shields.io/badge/license-MIT-blue?style=flat-square"></a>
  <a href="https://github.com/jassuwu/homebrew-tap"><img alt="Homebrew tap" src="https://img.shields.io/badge/homebrew-jassuwu%2Ftap%2Fmojify-FBB040?style=flat-square&logo=homebrew&logoColor=111"></a>
  <img alt="Platforms: macOS and Linux" src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux-555?style=flat-square">
</p>

<p align="center">
  <picture>
    <source media="(prefers-reduced-motion: reduce)" srcset="docs/assets/readme/mojify-header-poster.png">
    <img alt="Mojify transforms a polished mojify source animation into colored text video output." src="docs/assets/readme/mojify-header.gif" width="960">
  </picture>
</p>

## Installation

```bash
brew install jassuwu/tap/mojify
```

Homebrew installs a Mojify bottle when one is available for your host, builds from the tagged source archive otherwise, and installs the formula-declared runtime dependencies.

You can also download a macOS or Linux tarball from [GitHub Releases](https://github.com/jassuwu/mojify/releases) and place `mojify` on your `PATH`.

Check runtime tools:

```bash
mojify doctor
```

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

Export Mojify output as video, animation, still image, or text:

```bash
mojify export --overwrite --width 320 ./demo.mp4 ./dist/demo-mojify.mp4
mojify export --overwrite --width 320 --at 10s --duration 3s ./demo.mp4 ./dist/demo-mojify.gif
mojify export --overwrite --width 320 --at 10s ./demo.mp4 ./dist/demo-frame.png
mojify export --overwrite --width 80 --at 10s ./demo.mp4 ./dist/demo-frame.ansi
```

Export a local still image as Mojify image or text output:

```bash
mojify export --overwrite --width 320 ./poster.png ./dist/poster-mojify.png
mojify export --overwrite --width 80 ./poster.png ./dist/poster-mojify.ansi
```

Choose a built-in recipe preset:

```bash
mojify export --overwrite --recipe blocks --width 320 ./poster.png ./dist/poster-blocks.png
```

Inspect what Mojify will derive from a source:

```bash
mojify probe ./demo.mp4
```

## What It Does

Mojify accepts local video files, local still images, and yt-dlp-compatible platform URLs as source media. It turns those sources into colored character frames that can be played live or exported through a curated set of media, image, and text formats.

Current capabilities:

- Local video playback
- Local PNG and JPEG still-image input for `probe` and single-frame `export`
- yt-dlp-compatible URL input
- Live terminal audio playback
- Curated export formats: MP4, WebM, MOV, GIF, APNG, PNG, JPEG, plain text, and ANSI text
- Source audio preservation for supported video exports when audio is available
- Truecolor ANSI output
- Edge-aware character rendering
- Built-in recipe presets: `default`, `mono`, `ascii`, and `blocks`
- Runtime dependency check with `mojify doctor`
- `play`, `probe`, `export`, and `doctor` commands

## Why Mojify

Most media-to-ASCII experiments stop at the renderer. Mojify is an attempt to make that idea complete: playable, exportable, installable, and extensible, while leaning on FFmpeg and yt-dlp for the media plumbing they already do well.

## Renderer

The default renderer is built around a practical media-to-text recipe:

- luminance and intensity mapping choose the base character
- source color becomes terminal or exported frame color
- edge detection can override density characters with directional glyphs
- frame timing favors smooth playback over showing every decoded frame

Future renderer recipes may swap the character set, color strategy, or conversion rules entirely. Emoji output and custom character recipes are intentionally left as future product surface, not current README promises.

See [Recipes](docs/recipes.md) for built-in preset behavior and the future custom recipe direction.

## Requirements

Mojify shells out to battle-tested media tools instead of reimplementing their jobs:

- `ffmpeg` and `ffprobe` for media decoding, probing, and media/image export
- `ffplay` for live playback audio
- `yt-dlp` for platform URL inputs

Homebrew installs declare `ffmpeg` and `yt-dlp`. Tarball installs require the runtime tools to be installed separately. Run `mojify doctor` to check the tools visible to the installed binary.

## Roadmap

Planned or likely follow-up work:

- custom renderer recipes
- native Windows support beyond WSL
- a desktop app?
- a landing site?

WebP export remains deferred: `.webp` can mean either still or animated output, and WebP encoding is not guaranteed by Mojify's current FFmpeg dependency. Export PNG, GIF, APNG, or MP4 and convert externally when WebP is needed.

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
