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

## Future Renderer Pipeline

The current presets are intentionally small. They prove that Mojify can route a renderer recipe through playback, text export, image export, animation export, and video export without committing to a full custom recipe language.

The future renderer pipeline should take more inspiration from tools like Chafa and mpv:

- Chafa is the stronger model for recipe design. It treats rendering as a configurable canvas pipeline: symbol maps, cell geometry, color extraction, color space, preprocessing, dithering, and output optimization can all affect how pixels become terminal cells.
- mpv is the stronger model for playback and terminal backend design. Its terminal video outputs show that cell geometry, truecolor versus palette color, half-block rendering, buffering, and terminal image protocols are output-backend decisions, not just character-ramp choices.

That means future custom recipes should not be limited to "pick a string of characters". They should eventually describe explicit renderer axes:

- symbol map or density ramp
- cell geometry, such as full-cell characters, half-block cells, Braille-style subcells, or future terminal image backends
- color strategy, such as source color, no color, palette color, foreground/background extraction, or future quantization
- preprocessing, such as edge handling, contrast shaping, or blur/sharpen steps
- dithering and output optimization choices

Those are future architecture notes, not current CLI promises. `--recipe default|mono|ascii|blocks` remains the supported user surface for this stage.
