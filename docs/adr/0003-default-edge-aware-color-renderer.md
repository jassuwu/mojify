# Use an edge-aware color renderer by default

Mojify's default renderer combines luminance-based character density, ANSI truecolor, and edge glyph overrides. This is the core visual identity of the project, so the renderer should be implemented as testable stages but shipped as one canonical default rather than as optional complexity added after a basic ASCII player.
