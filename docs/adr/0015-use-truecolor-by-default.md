# Use truecolor by default

Mojify v1 renders character frames with 24-bit ANSI foreground color by default. The prototype's visual identity depends on color, and modern terminals generally support truecolor; performance work should focus first on efficient escape sequence output rather than weakening the default renderer.
