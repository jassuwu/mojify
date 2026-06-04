# Export QA

Export QA uses generated clips for a repeatable smoke test across Mojify's curated output formats and ignored real clips under `dist/` for optional source-audio verification.

Platform URL export is covered by the cross-command checklist in `docs/qa/platform-media-input.md`.

## Supported Formats

`mojify export SOURCE OUTPUT` selects the output behavior from the `OUTPUT` extension:

- Video: `.mp4`, `.webm`, `.mov`
- Animated visual: `.gif`, `.apng`
- Still image: `.png`, `.jpg`, `.jpeg`
- Single-frame text: `.txt`, `.ansi`

`.webp` is intentionally deferred. WebP is ambiguous in Mojify's extension-routed export contract because the same extension can represent a still image or an animated visual, and WebP encoding is not guaranteed by the current FFmpeg runtime dependency. Users who need WebP should export PNG, GIF, APNG, or MP4 and convert externally for now.

`--at <timestamp>` is valid for every supported export format. Accepted timestamp examples include `10`, `10s`, `1:23`, and `01:02:03.250`.

`--duration <duration>` is valid only for time-based exports: `.mp4`, `.webm`, `.mov`, `.gif`, and `.apng`. It is invalid for single-frame outputs: `.png`, `.jpg`, `.jpeg`, `.txt`, and `.ansi`.

Text exports are single rendered Mojify character frames. `.txt` writes plain text, and `.ansi` writes a colored ANSI frame. For media and image outputs, `--width` means output pixels. For text outputs, `--width` means character columns.

## Canonical Smoke

```bash
bun run qa:clips
bun run build
bun run qa:export
```

Expected generated output:

- `dist/qa/export/low-motion-bars-export.mp4`
- `dist/qa/export/low-motion-bars-export.webm`
- `dist/qa/export/low-motion-bars-export.mov`
- `dist/qa/export/low-motion-bars-export.gif`
- `dist/qa/export/low-motion-bars-export.apng`
- `dist/qa/export/low-motion-bars-frame.png`
- `dist/qa/export/low-motion-bars-frame.jpg`
- `dist/qa/export/low-motion-bars-frame.jpeg`
- `dist/qa/export/low-motion-bars-frame.txt`
- `dist/qa/export/low-motion-bars-frame.ansi`
- `ffprobe` reports a video or image stream for the synthetic media/image exports.
- The exported synthetic media/image streams have width `320`.
- The exported synthetic text files are non-empty.
- Unsupported `.webp` output is rejected.
- `--duration` is rejected for single-frame outputs.

If a top-level `dist/` media file with an audio stream is present, `bun run qa:export` also writes:

- `dist/qa/export/real-sample-export.mp4`
- `dist/qa/export/real-sample-export.webm`
- `dist/qa/export/real-sample-export.mov`
- `ffprobe` reports an audio stream in those exported video files.

## Throughput QA

Use export stats to compare the current branch against the previous `main` baseline with the same source, output width, terminal, and machine load.

```bash
bun run qa:clips
bun run build
time ./bin/mojify export --overwrite --stats --width 320 --duration 2s dist/qa/low-motion-bars.mp4 dist/qa/export/low-motion-bars-export.mp4
```

Record ignored local notes under `dist/qa/export-throughput-after.md` when comparing a branch. Include:

- source file
- output width
- worker count from `export stats`
- elapsed wall-clock time from `time`
- `export stats` summary
- whether exported media/image/text QA still passes

## Manual Synthetic Smoke

```bash
mkdir -p dist/qa/export
./bin/mojify export --overwrite --width 320 dist/qa/low-motion-bars.mp4 dist/qa/export/low-motion-bars-export.mp4
./bin/mojify export --overwrite --width 320 --at 0s --duration 2s dist/qa/low-motion-bars.mp4 dist/qa/export/low-motion-bars-export.gif
./bin/mojify export --overwrite --width 320 --at 0s dist/qa/low-motion-bars.mp4 dist/qa/export/low-motion-bars-frame.png
./bin/mojify export --overwrite --width 80 --at 0s dist/qa/low-motion-bars.mp4 dist/qa/export/low-motion-bars-frame.ansi
ffprobe -hide_banner -v error \
  -select_streams v:0 \
  -show_entries stream=codec_name,width,height,avg_frame_rate,duration \
  -of default=noprint_wrappers=1 \
  dist/qa/export/low-motion-bars-export.mp4
```

Expected progress output:

- Interactive stderr updates one progress line while rendering frames.
- Known-total time-based exports show rendered export-frame progress such as `exporting video: 120/240 frames 50%` or equivalent family-aware wording.
- Progress reaches `100%` only after visual frames have been rendered and written to the encoder.
- After `100%`, status switches to format finalization.
- Export ends with `export complete: <output>`.
- No ETA or time-remaining text is printed.

## Optional Real Sample Audio QA

Use an ignored real local media file under `dist/` that has source audio:

```bash
REAL_SAMPLE="dist/<real-sample-with-audio>.webm"
mkdir -p dist/qa/export
for ext in mp4 webm mov; do
  ./bin/mojify export --overwrite --width 320 --duration 2s "$REAL_SAMPLE" "dist/qa/export/real-sample-export.${ext}"
  ffprobe -hide_banner -v error \
    -select_streams a:0 \
    -show_entries stream=codec_name,sample_rate,channels,duration \
    -of default=noprint_wrappers=1 \
    "dist/qa/export/real-sample-export.${ext}"
done
```

The audio QA passes when each exported video file contains an audio stream. If the source file has no audio stream, choose a different ignored real sample or skip this optional check.

## QA Matrix

| Format | Output | Selection flags | QA check |
| --- | --- | --- | --- |
| `.mp4` | `low-motion-bars-export.mp4` | full synthetic clip | video stream, width `320` |
| `.webm` | `low-motion-bars-export.webm` | `--at 0s --duration 2s` | video stream, width `320` |
| `.mov` | `low-motion-bars-export.mov` | `--at 0s --duration 2s` | video stream, width `320` |
| `.gif` | `low-motion-bars-export.gif` | `--at 0s --duration 2s` | video stream, width `320` |
| `.apng` | `low-motion-bars-export.apng` | `--at 0s --duration 2s` | video stream, width `320` |
| `.png` | `low-motion-bars-frame.png` | `--at 0s` | image stream, width `320` |
| `.jpg` | `low-motion-bars-frame.jpg` | `--at 0s` | image stream, width `320` |
| `.jpeg` | `low-motion-bars-frame.jpeg` | `--at 0s` | image stream, width `320` |
| `.txt` | `low-motion-bars-frame.txt` | `--at 0s --width 80` | non-empty text |
| `.ansi` | `low-motion-bars-frame.ansi` | `--at 0s --width 80` | non-empty ANSI text |
| `.mp4`, `.webm`, `.mov` | `real-sample-export.<ext>` | `--duration 2s` with optional real sample | audio stream present |

## Checklist

- Synthetic export completes without prompting because `--overwrite` is set.
- Synthetic export writes the representative matrix outputs.
- `ffprobe` finds video/image stream `v:0` for media and image outputs.
- The exported media/image width is `320`.
- Text outputs are non-empty single-frame exports.
- `--at` works for video, animated, still image, and still text outputs.
- `--duration` works for video and animated outputs.
- `--duration` is rejected for still image outputs in automated QA and for all still image/text outputs by parser tests.
- `.webp` remains unsupported in this stage.
- Optional real-sample export writes `dist/qa/export/real-sample-export.mp4`, `.webm`, and `.mov`.
- Optional real-sample export preserves source audio when the source has audio.
- Known-total export progress reaches `100%` before format finalization.
- Export progress does not print an ETA or time remaining.
- Non-TTY export logs remain sparse and readable.
- Export stats print when `--stats` is passed.
- Export stats do not print by default.
- Parallel export preserves ordered video frames and source audio behavior.
- Throughput comparisons use the same source file, width, machine, and terminal conditions.
