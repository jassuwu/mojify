# Export MP4 with source audio before live terminal audio

Mojify will make MP4 export the next product utility capability and preserve source audio content in exported media when available. Source media without audio should still export successfully as a silent MP4. For MP4 compatibility, export may transcode source audio to AAC rather than attempting a bit-for-bit stream copy. This separates audio muxing for exported files from live terminal audio playback, which has harder runtime sync, pause/resume, and cleanup concerns and should be solved later.
