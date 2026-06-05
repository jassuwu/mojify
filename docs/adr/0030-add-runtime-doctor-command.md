# Add runtime doctor command

Mojify will add `mojify doctor` as a post-distribution CLI polish stage. This supersedes the earlier installable-distribution deferral of a doctor command because Mojify now has Homebrew bottles, source fallback, and GitHub Release tarballs, and users need one command that explains whether the external runtime tools are available.

The doctor command checks `ffmpeg`, `ffprobe`, `ffplay`, and `yt-dlp` on `PATH`. Missing or unhealthy `ffmpeg` and `ffprobe` are errors because local media playback, probing, and export depend on them. Missing or unhealthy `ffplay` and `yt-dlp` are warnings because visual playback can run with `--no-audio`, and local file workflows do not require platform URL resolution.

Doctor does not install dependencies, run network checks, download sample media, check audio devices, probe terminal capabilities, or produce machine-readable diagnostics in this stage.
