# Design for live resize but do not promise it in v1

Mojify v1 guarantees auto-fit at playback start, not live terminal resize during playback. The core should still keep decoded frame size separate from render grid size so future frames can adapt to resize events without redesigning the decode/render boundary.
