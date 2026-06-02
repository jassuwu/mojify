# Use ffplay for live terminal audio

Mojify will use an `ffplay` process as the first live terminal audio backend for `mojify play`. This stays consistent with the existing FFmpeg CLI media boundary, avoids native audio libraries and platform-specific audio device code during the source-build phase, and gives Mojify process-level start/stop control for default-on playback audio with `--no-audio` as the opt-out.
