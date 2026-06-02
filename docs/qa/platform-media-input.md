# Platform Media Input QA

Platform media input lets Mojify accept HTTP(S), yt-dlp-compatible URLs anywhere the `Source` CLI argument accepts source media, then resolves each URL into temporary local source media before running `probe`, `play`, or `export`.

## Automated Contract

Automated tests should use a fake `yt-dlp` executable instead of real network access.

Expected contract coverage:

- Local file sources bypass yt-dlp and print no source-resolution status.
- HTTP(S) sources invoke yt-dlp with Mojify's explicit arguments.
- yt-dlp receives `--ignore-config`, `--no-playlist`, `--match-filters !is_live`, `--no-progress`, `--paths`, `--output`, `--print after_move:filepath`, `--merge-output-format mp4`, and `-f bv*[ext=mp4]+ba[ext=m4a]/b[ext=mp4]/b`.
- The final resolved source path comes from yt-dlp's `after_move:filepath` output.
- Temporary source directories are cleaned after command completion, command failure, and cancellation.
- Missing yt-dlp produces a concise platform URL dependency error.
- yt-dlp failures surface concise stderr context.
- Local export output paths still reject protocol outputs before any source download starts.

## Optional Real URL Smoke

Use a single finite public video URL that yt-dlp can resolve without cookies.

```bash
bun run build
URL="<yt-dlp-compatible-http-url>"
./bin/mojify probe "$URL"
./bin/mojify play --stats "$URL"
./bin/mojify export --overwrite --width 320 "$URL" dist/qa/export/platform-url-export.mp4
```

Expected:

- URL resolution prints simple stderr phases before normal command output.
- `probe` prints `input: <original-url>` and `resolved-source: <downloaded-basename>`.
- `play` starts only after source resolution completes.
- Playback audio, pause/resume, `q`, and Ctrl-C behave the same as local resolved source media.
- Export writes a valid MP4 and preserves source audio when the resolved source has audio.
- No downloaded source media is left behind by default.

## Out Of Scope

- Playlist workflow.
- Live stream playback or export.
- yt-dlp config files, cookies, auth, proxies, and custom downloader arguments.
- Persistent download cache.
- Streaming URL data directly into FFmpeg.
