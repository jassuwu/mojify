# Resolve platform media input in the CLI

Mojify will accept HTTP(S), yt-dlp-compatible platform URLs by resolving each URL into a temporary local media file before running probe, playback, or export. Source resolution belongs at the CLI boundary, not in FFmpeg media primitives or `main`, so `media`, `player`, and `exporter` can keep operating on resolved local paths while user-facing status, errors, and cleanup stay with command orchestration.

The first platform-input stage uses download-first resolution rather than streaming, creates a fresh temp directory per command, asks yt-dlp for one finite non-playlist video, prefers a merged playable MP4 result, reports only simple resolution/download phases on stderr, and cleans the downloaded source after the command exits. Local file inputs stay silent and bypass yt-dlp entirely; yt-dlp is required only when the user passes a platform URL.
