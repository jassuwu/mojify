# mojify

Mojify is a terminal-first video player that transforms local video files into colored, edge-aware character frames.

## Status

V1 is source-build only. The first milestone is local visual playback in the terminal.

## Requirements

- Go 1.23+
- Bun 1.3+
- FFmpeg and ffprobe on `PATH`

## Run

```bash
bun install
bun run build
./bin/mojify --help
./bin/mojify probe ./demo.mp4
./bin/mojify play ./demo.mp4
```

## Playback QA

```bash
bun run qa:clips
bun run build
./bin/mojify play --stats dist/qa/low-motion-bars.mp4
```

The repeatable playback quality checklist lives in `docs/qa/playback-quality.md`.

## Scope

Included in v1:

- Local video files
- Visual terminal playback
- Truecolor ANSI output
- Edge-aware character rendering
- `play` and `probe` commands

Deferred:

- YouTube/URL input
- Audio
- Export to GIF/MP4/PNG
- npm/npx distribution
- Plugins and custom recipes
