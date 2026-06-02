# Enable live playback audio by default

Mojify playback will attempt live terminal audio by default when the source media has an audio stream, with `--no-audio` as the explicit playback opt-out. Audio is best-effort so missing `ffplay`, silent sources, or audio-device failures should not block visual terminal playback; this makes source media feel complete by default while preserving a clean escape hatch for silent terminal sessions.
