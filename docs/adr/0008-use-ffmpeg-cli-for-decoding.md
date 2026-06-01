# Use the FFmpeg CLI for v1 decoding

Mojify v1 shells out to the FFmpeg CLI to decode local video files into raw frames. This keeps the Go core focused on rendering and playback while avoiding FFmpeg binding, linking, and codec distribution problems during the source-build phase.
