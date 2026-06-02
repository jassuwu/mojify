# Export QA

MP4 export QA uses generated clips for a repeatable smoke test and ignored real clips under `dist/` for optional source-audio verification.

Platform URL export is covered by the cross-command checklist in `docs/qa/platform-media-input.md`.

## Canonical Smoke

```bash
bun run qa:clips
bun run build
bun run qa:export
```

Expected generated output:

- `dist/qa/export/low-motion-bars-export.mp4`
- `ffprobe` reports a video stream for the synthetic export.
- The exported synthetic video stream has width `320`.

If a top-level `dist/` media file with an audio stream is present, `bun run qa:export` also writes:

- `dist/qa/export/real-sample-export.mp4`
- `ffprobe` reports an audio stream in that exported MP4.

## Throughput QA

Use export stats to compare the current branch against the previous `main` baseline with the same source, output width, terminal, and machine load.

```bash
bun run qa:clips
bun run build
time ./bin/mojify export --overwrite --stats --width 320 dist/qa/low-motion-bars.mp4 dist/qa/export/low-motion-bars-export.mp4
```

Record ignored local notes under `dist/qa/export-throughput-after.md` when comparing a branch. Include:

- source file
- output width
- worker count from `export stats`
- elapsed wall-clock time from `time`
- `export stats` summary
- whether exported video/audio QA still passes

## Manual Synthetic Smoke

```bash
mkdir -p dist/qa/export
./bin/mojify export --overwrite --width 320 dist/qa/low-motion-bars.mp4 dist/qa/export/low-motion-bars-export.mp4
ffprobe -hide_banner -v error \
  -select_streams v:0 \
  -show_entries stream=codec_name,width,height,avg_frame_rate,duration \
  -of default=noprint_wrappers=1 \
  dist/qa/export/low-motion-bars-export.mp4
```

Expected progress output:

- Interactive stderr updates one progress line while rendering frames.
- Known-total exports show rendered export-frame progress such as `exporting video: 120/240 frames 50%`.
- Progress reaches `100%` only after visual frames have been rendered and written to the encoder.
- After `100%`, status switches to `finalizing mp4...`.
- Export ends with `export complete: <output>`.
- No ETA or time-remaining text is printed.

## Optional Real Sample Audio QA

Use an ignored real local media file under `dist/` that has source audio:

```bash
REAL_SAMPLE="dist/<real-sample-with-audio>.webm"
mkdir -p dist/qa/export
./bin/mojify export --overwrite --width 320 "$REAL_SAMPLE" dist/qa/export/real-sample-export.mp4
ffprobe -hide_banner -v error \
  -select_streams a:0 \
  -show_entries stream=codec_name,sample_rate,channels,duration \
  -of default=noprint_wrappers=1 \
  dist/qa/export/real-sample-export.mp4
```

The audio QA passes when the exported MP4 contains an audio stream. If the source file has no audio stream, choose a different ignored real sample or skip this optional check.

## Checklist

- Synthetic export completes without prompting because `--overwrite` is set.
- Synthetic export writes `dist/qa/export/low-motion-bars-export.mp4`.
- `ffprobe` finds video stream `v:0`.
- The exported video width is `320`.
- Optional real-sample export writes `dist/qa/export/real-sample-export.mp4`.
- Optional real-sample export preserves source audio when the source has audio.
- Known-total export progress reaches `100%` before `finalizing mp4...`.
- Export progress does not print an ETA or time remaining.
- Non-TTY export logs remain sparse and readable.
- Export stats print when `--stats` is passed.
- Export stats do not print by default.
- Parallel export preserves ordered video frames and source audio behavior.
- Throughput comparisons use the same source file, width, machine, and terminal conditions.
