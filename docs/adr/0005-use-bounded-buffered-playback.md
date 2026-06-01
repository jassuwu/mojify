# Use bounded buffered playback

Mojify v1 starts playback after preparing a short bounded buffer of character frames, then keeps producing frames while a clocked presenter consumes them. This avoids the startup and memory cost of full pre-rendering while still smoothing over decode and render spikes better than immediate frame-by-frame playback.
