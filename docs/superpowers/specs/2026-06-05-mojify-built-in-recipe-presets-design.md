# Mojify Built-In Recipe Presets Design

## Status

Approved design for the first renderer recipe stage. This spec does not implement the feature.

## Goal

Add built-in recipe presets so users can choose a small set of alternate Mojify character-frame looks for playback and export without changing the default renderer or introducing custom recipe files.

## Product Surface

Recipe presets are selected with `--recipe <name>` on commands that render frames:

```bash
mojify play --recipe blocks ./demo.mp4
mojify export --recipe mono ./poster.png ./dist/poster-mono.ansi
```

The first built-in presets are:

- `default`: the current Mojify renderer behavior.
- `mono`: the default recipe shape without source/ANSI color.
- `ascii`: a classic ASCII luminance ramp without source color or edge glyphs.
- `blocks`: a colored Unicode shade/block ramp without edge glyphs.

No `--recipe` flag preserves today's behavior. `--recipe default` is valid for explicit baseline commands.

`probe` does not accept `--recipe` and does not print recipe information. Recipes do not affect source metadata, render grid sizing, FPS, export dimensions, audio, or platform source resolution.

## Preset Semantics

The presets are named combinations of renderer choices:

| Preset | Character ramp | Color mode | Edge mode |
| --- | --- | --- | --- |
| `default` | existing Mojify ramp: ` .;coPO?#@` | source color | default edge override |
| `mono` | existing Mojify ramp: ` .;coPO?#@` | no color | default edge override |
| `ascii` | classic ASCII ramp: ` .:-=+*#%@` | no color | no edge override |
| `blocks` | Unicode shade/block ramp: ` ░▒▓█` | source color | no edge override |

For text output, no-color recipes write plain characters. For ANSI output, no-color recipes should not reintroduce ANSI foreground colors. For raster media and image exports, no-color recipes use white glyphs over the existing black background.

## Internal Shape

Built-in presets should be represented as recipe definitions, not scattered CLI string checks. A recipe definition should contain at least:

- a character ramp mode: `default`, `ascii`, or `blocks`
- a color mode: `source` or `none`
- an edge mode: `default` or `none`

The renderer should consume a resolved recipe definition. CLI parsing should validate recipe preset names before source resolution so typos do not trigger yt-dlp downloads or media probing.

Rendered cells need explicit color presence, such as `HasColor bool`, so serializers can distinguish "no color" from a real black source pixel.

This internal shape should keep a future `--recipe-file <path>` possible. Custom recipe loading is not part of this stage.

## Documentation

Update:

- `README.md` with one compact recipe example and a capability bullet listing the presets.
- `docs/recipes.md` as the durable recipe preset page.
- `docs/qa/export.md` with recipe preset QA expectations.

`docs/recipes.md` may mention that custom recipe files are planned as a future stage, likely through a separate explicit flag such as `--recipe-file`, but it must not publish a schema promise.

## QA

Automated tests should cover:

- CLI parse support for `--recipe` on `play` and `export`.
- CLI parse rejection for unknown recipes before command execution.
- `probe` rejection of `--recipe`.
- Renderer golden behavior for each preset.
- ANSI/text serialization behavior when `HasColor` is false.
- Rasterizer white-on-black behavior for no-color cells.
- Exporter handoff of selected recipe through video, image, and text export paths.
- Playback renderer selection.

`bun run qa:export` should generate a still-source preset matrix using `dist/qa/still-source.png`, exporting at least image and ANSI outputs for every preset. The script should verify exported image widths and non-empty text/ANSI files.

## Non-Goals

- Custom recipe files or scripts.
- `--recipe-file`.
- Emoji recipe.
- Braille or subpixel Unicode renderer.
- User-composable flags such as `--color mono --edges off`.
- Recipe-specific layout, aspect, FPS, or export dimension changes.
- `probe --recipe`.
- Changing the default look.
- Terminal font detection.
- New export formats.

## Acceptance Criteria

- `mojify play` and `mojify export` accept `--recipe default|mono|ascii|blocks`.
- Invalid recipe names fail during CLI parsing with a supported-name list.
- Existing commands without `--recipe` retain the current default visual behavior.
- `probe` stays recipe-free.
- Playback and every export family use the same selected recipe.
- `.txt` and `.ansi` reflect no-color recipes instead of forcing color back in.
- Raster exports for no-color recipes render white glyphs on black.
- Recipe docs and QA docs are updated.
- `bun run qa:export` covers all built-in presets with the still-source matrix.
