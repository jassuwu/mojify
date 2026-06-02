# Use frame-diffed presentation

Synchronized presentation alone did not materially improve Ghostty playback quality because Mojify still clears and rewrites the whole terminal frame on every tick. The next playback quality hardening step is frame-diffed presentation: preserve the renderer output and playback scheduler, but make steady-state terminal presentation update only changed regions between consecutive character frames.

The first frame and dimension changes should still use a full redraw. This trades presenter simplicity for lower terminal write volume and better perceived smoothness, with Ghostty-visible improvement as the acceptance gate and playback metrics as regression guards.
