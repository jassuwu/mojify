# Recipes

Mojify recipes control how source pixels become character frames. The current release supports built-in recipe presets selected with `--recipe <name>` on commands that render frames.

```bash
mojify play --recipe blocks ./demo.mp4
mojify export --recipe mono ./poster.png ./dist/poster-mono.ansi
```

`probe` does not accept recipes because recipes do not change source metadata or derived layout.

## Built-In Presets

| Preset | Character mapping | Color | Edges |
| --- | --- | --- | --- |
| `default` | Mojify's default ramp, ` .;coPO?#@` | source color | default edge glyphs |
| `mono` | Mojify's default ramp, ` .;coPO?#@` | none | default edge glyphs |
| `ascii` | classic ASCII ramp, ` .:-=+*#%@` | none | none |
| `blocks` | Unicode shade/block ramp, ` ░▒▓█` | source color | none |

No `--recipe` flag is the same as `--recipe default`.

For text output, no-color recipes write plain characters. For ANSI output, no-color recipes do not emit foreground color escapes. For raster image, animation, and video exports, no-color recipes render white glyphs on black.

## Custom Recipes

Custom recipe files are planned as a future stage. They should use a separate explicit surface, likely `--recipe-file <path>`, rather than overloading `--recipe` with both preset names and file paths.

Mojify does not currently publish or support a custom recipe file schema.
