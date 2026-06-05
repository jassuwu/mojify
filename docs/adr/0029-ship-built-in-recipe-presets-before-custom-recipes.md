# Ship built-in recipe presets before custom recipes

Mojify will expose renderer variation first as named built-in recipe presets through `--recipe`, rather than loading user-provided recipe files in the same stage. The first presets are `default`, `mono`, `ascii`, and `blocks`; they are intentionally small named combinations of character ramp, color mode, and edge mode so users can change the rendered look without Mojify committing to a custom recipe file schema too early.

Custom recipe files remain a future product surface and should use an explicit path-oriented flag such as `--recipe-file` instead of overloading `--recipe` with both preset names and file paths. This preserves typo-safe parse-time validation for preset names, avoids accidental yt-dlp or media work for invalid recipes, and lets the implementation still move toward a shared internal recipe-definition model.
