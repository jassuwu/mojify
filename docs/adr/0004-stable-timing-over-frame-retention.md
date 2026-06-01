# Prefer stable timing over frame retention

Mojify v1 prioritizes stable terminal playback timing over displaying every decoded frame. If rendering or terminal output falls behind schedule, the player may skip late frames rather than slowing the whole video down.
