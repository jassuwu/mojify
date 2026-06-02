# Mojify

Mojify is a terminal-first video experience. It exists to transform source media into colored, edge-aware character frames for smooth playback in a terminal.

## Language

**Mojify**:
The project and tool for playing videos in a terminal as colored, edge-aware character frames.
_Avoid_: Prototype video to ASCII, media-to-ASCII clone

**Terminal-first video player**:
The v1 product shape for Mojify: terminal playback is the primary use case, and other outputs are secondary.
_Avoid_: General media converter, ASCII toolkit, recipe platform

**Source media**:
A local video file accepted as input by Mojify v1.
_Avoid_: YouTube URL, remote URL, stream

**Decoder**:
The FFmpeg CLI process that turns source media into raw video frames for Mojify v1.
_Avoid_: FFmpeg bindings, codec engine

**Playback**:
Visual terminal playback of character frames from source media. In v1, playback does not include audio.
_Avoid_: Audio playback, export

**Live terminal audio**:
Audio played while Mojify presents character frames in the terminal.
_Avoid_: Exported media audio, audio muxing

**MP4 export**:
A product utility output that renders Mojify visuals into an MP4 file and preserves source audio content when available.
_Avoid_: Terminal playback, silent export by default

**Export progress**:
User-facing export status on stderr that reports rendered frame progress against the export frame count when the total is knowable, and an indeterminate rendered-frame count when it is not. `100%` means Mojify has rendered and written all visual frames to the MP4 encoder, after which the status should move to MP4 finalization; export progress should be terminal-friendly when interactive, log-friendly otherwise, and should not claim an ETA or time remaining by default.
_Avoid_: Export ETA, FFmpeg progress, fake completion estimate

**Export throughput hardening**:
The product utility stage that improves MP4 export speed while preserving ordered frames, output correctness, source audio behavior, and honest export progress.
_Avoid_: Distribution polish, live terminal audio, faster-looking progress

**Export font**:
The bundled monospace font used to rasterize Mojify character frames into exported media. The preferred default is `Mx437_IBM_BIOS`.
_Avoid_: User terminal font, arbitrary system font

**Playback controls**:
The minimal interactive controls available during playback: quit and pause/resume.
_Avoid_: Seeking, speed control, zoom

**Playable local video**:
The first milestone: a local video file plays in the terminal with auto-fit truecolor edge-aware character frames and minimal playback controls.
_Avoid_: Scaffold complete, renderer-only prototype

**Playback quality hardening**:
The stage after playable local video: improving perceived smoothness and repeatable evaluation of terminal playback before expanding the product surface.
_Avoid_: Audio, export, URL input, plugins, packaged distribution

**Product utility expansion**:
The roadmap phase after playable, visually acceptable local playback: adding capabilities that make Mojify useful as a product before investing in package distribution or release polish.
_Avoid_: Repository prettiness, packaged distribution, release automation

**Smooth playback**:
Playback that maintains stable frame timing in the terminal, even if late frames must be skipped.
_Avoid_: Showing every frame, frame-perfect playback

**Perceived smoothness**:
The user's experience that playback updates feel continuous in a real terminal, with minimal visible repainting, flicker, or janky timing.
_Avoid_: Raw FPS alone, benchmark-only performance

**Practical terminal smoothness**:
The acceptance bar for playback quality hardening: sample clips should play continuously in a normal terminal size without distracting full-screen flashing or obvious repaint waves, supported by playback metrics.
_Avoid_: Native-video smoothness, metric-only success

**Playback metrics**:
Runtime measurements used during playback quality hardening: rendered frames, skipped frames, effective FPS, average frame render time, average present/write time, output bytes per frame, and render grid size.
_Avoid_: Full profiler trace, benchmark-only report

**Terminal output optimization**:
The playback quality hardening work that reduces visible repainting and terminal write volume during playback while preserving the renderer, scheduler, controls, and CLI shape.
_Avoid_: New renderer recipe, new product surface, audio, export

**Synchronized presentation**:
A best-effort terminal presentation mode where each character frame update is bracketed so capable terminals apply the update as a single visual refresh.
_Avoid_: Frame diffing, lower fidelity rendering, required terminal feature

**Frame-diffed presentation**:
A terminal presentation mode where Mojify updates only changed regions between consecutive character frames instead of repainting the whole frame every tick.
_Avoid_: Renderer change, lower fidelity rendering, full-screen repaint

**Sample clip QA set**:
The repeatable clips used to evaluate playback quality hardening. The canonical set is generated synthetic clips for low-motion, high-motion, and high-contrast edge cases; ignored local real clips can supplement manual QA.
_Avoid_: Checked-in copyrighted videos, one-off user-only demos

**Balanced fidelity**:
The default visual target: enough detail for the edge-aware renderer to be legible while preserving stable playback timing.
_Avoid_: Maximum detail, maximum FPS

**Bounded buffer**:
A small queue of ready character frames prepared before and during playback. It gives playback a short lead without pre-rendering the entire source media.
_Avoid_: Full pre-render, frame cache

**Character frame**:
A single rendered terminal frame made from text characters, optional ANSI color, and optional edge glyphs.
_Avoid_: ASCII image, text bitmap

**Truecolor**:
The default terminal color mode for v1, using 24-bit ANSI foreground color for character frames.
_Avoid_: 256-color palette, monochrome default

**Render grid**:
The terminal-sized character dimensions used to produce character frames for playback.
_Avoid_: Output resolution, video size

**Auto-fit**:
The default sizing behavior where Mojify chooses a render grid that fits the current terminal.
_Avoid_: Fixed width, required width

**Live resize**:
Adapting the render grid while playback is already running.
_Avoid_: Startup auto-fit, video zoom

**Zoom**:
An interactive or configured source viewport adjustment applied during playback.
_Avoid_: Auto-fit, live resize, terminal font-size changes

**Renderer recipe**:
The rules that turn pixels into characters, colors, and edge glyphs.
_Avoid_: Formula, filter, converter

**Golden renderer test**:
A test fixture that locks expected character, color, and edge output for a small input frame.
_Avoid_: Screenshot test, demo clip

**Default renderer**:
Mojify's built-in renderer recipe: luminance chooses character density, source color becomes terminal color, and detected edge direction can override the density character with an edge glyph.
_Avoid_: Basic ASCII renderer, color filter

**Edge glyph**:
A character chosen to preserve directional edges in a character frame, such as `/`, `\`, `|`, or `-`.
_Avoid_: Line art, border

**Core**:
The native Go implementation that decodes, renders, buffers, and presents character frames for v1.
_Avoid_: Engine, backend

**Monorepo**:
The project repository shape that houses the Go core, future TypeScript package surfaces, site, docs, assets, and release tooling.
_Avoid_: Polyrepo, Go-only repo

**CLI surface**:
The user-facing command shape for Mojify. In v1, bare `mojify` shows help, and playback is invoked through an explicit subcommand.
_Avoid_: Implicit command, flags-only interface

**Probe**:
A support command that reports source media metadata and Mojify's derived playback/render dimensions without playing the video.
_Avoid_: Export, inspect mode
