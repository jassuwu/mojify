# mojify

Mojify is a terminal-first video player that transforms local video files and yt-dlp-compatible platform URLs into colored, edge-aware character frames.

## Status

Mojify is source-build only while the product capabilities are being built.

## Requirements

- Go 1.23+
- Bun 1.3+
- FFmpeg and ffprobe on `PATH`
- yt-dlp on `PATH` for platform URL inputs
- ffplay on `PATH` for live playback audio

## Run

```bash
bun install
bun run build
./bin/mojify --help
./bin/mojify probe ./demo.mp4
./bin/mojify play ./demo.mp4
./bin/mojify probe "https://www.youtube.com/watch?v=<id>"
./bin/mojify play "https://www.youtube.com/watch?v=<id>"
./bin/mojify export --overwrite --width 320 ./demo.mp4 dist/demo-export.mp4
./bin/mojify export --overwrite --width 320 "https://www.youtube.com/watch?v=<id>" dist/demo-url-export.mp4
```

## Playback QA

```bash
bun run qa:clips
bun run build
./bin/mojify play --stats dist/qa/low-motion-bars.mp4
```

The repeatable playback quality checklist lives in `docs/qa/playback-quality.md`.

## Export QA

```bash
bun run qa:clips
bun run build
bun run qa:export
```

MP4 export writes colored character-frame video and includes source audio content when the input file has audio. The repeatable export checklist lives in `docs/qa/export.md`.

## Scope

Included now:

- Local video files
- yt-dlp-compatible HTTP(S) platform URLs
- Visual terminal playback
- Live terminal audio playback
- MP4 export with source audio content when available
- Truecolor ANSI output
- Edge-aware character rendering
- `play`, `probe`, and `export` commands

Deferred:

- Export to GIF/PNG
- npm/npx distribution
- Plugins
- Custom recipes
- Playlist workflow
- Live streams
