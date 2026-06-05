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
The media accepted as input by Mojify: a local time-based media file, a local still image, or a platform URL that Mojify resolves into a temporary local media file before processing.
_Avoid_: Live stream, stdin stream, unresolved URL stream

**Still source**:
A local still image accepted as source media for probing and single-frame export workflows, but not for terminal playback.
_Avoid_: Image playback, animated image source, direct HTTP image URL

**Still source export**:
A single-frame export workflow where a local still source renders to `.txt`, `.ansi`, `.png`, `.jpg`, or `.jpeg` output.
_Avoid_: Still-to-video export, still-to-animation export, synthetic duration

**Still source timestamp rejection**:
The rule that `--at` and `--duration` do not apply to local still sources because still sources have no timeline.
_Avoid_: Ignored timestamp flags, synthetic image timeline, frame-zero alias

**Source**:
The CLI argument name for source media accepted by `probe`, `play`, and `export`.
_Avoid_: Video argument, URL-only argument, input stream

**Decoder**:
The FFmpeg CLI process that turns source media into raw video frames for Mojify v1.
_Avoid_: FFmpeg bindings, codec engine

**Platform media input**:
The product utility stage that lets Mojify accept yt-dlp-compatible URLs by resolving them to temporary local source media before running probe, playback, or export.
_Avoid_: URL streaming, browser extraction, playlist workflow

**Playback**:
Terminal playback of Mojify character frames from source media. Playback may include live terminal audio when the source has audio and audio has not been explicitly disabled.
_Avoid_: Export, MP4 audio muxing

**Live terminal audio**:
Source audio played while Mojify presents character frames in the terminal.
_Avoid_: Exported media audio, audio muxing

**Playback audio**:
The product utility stage that adds default-on live terminal audio to `mojify play` while preserving visual playback fallback, pause/resume semantics, and reliable cleanup.
_Avoid_: Exported media audio, audio muxing, mute controls, volume controls

**MP4 export**:
A product utility output that renders Mojify visuals into an MP4 file and preserves source audio content when available.
_Avoid_: Terminal playback, silent export by default

**Curated multi-format export**:
The export stage where `mojify export SOURCE OUTPUT` selects a supported output family by extension: video, animated visual, still image, or still text. Mojify owns these output contracts even when FFmpeg performs the encoding.
_Avoid_: Any FFmpeg-compatible output, raw format passthrough

**Timestamp export selection**:
The `--at` and `--duration` model for choosing where export starts and, for time-based outputs, how much source media to render. Selection is timestamp-based, not exact source-frame addressing.
_Avoid_: Exact frame selection, frame number export

**Still export**:
A single-frame visual export selected by an image extension such as `.png`, `.jpg`, or `.jpeg`.
_Avoid_: Animated image export, source-frame dump

**Animated export**:
A time-based visual export selected by an animated format extension such as `.gif` or `.apng`, without source audio.
_Avoid_: Video export with audio, terminal recording

**Text export**:
Single-frame `.txt` or `.ansi` output generated from a rendered Mojify character frame without rasterizing to pixels.
_Avoid_: Animated text export, terminal recording

**WebP deferral**:
The explicit decision to leave `.webp` out of the curated export set while WebP remains both semantically ambiguous for Mojify's extension-routed export contract (still image vs animated visual) and not guaranteed by the current FFmpeg runtime dependency. Users who need WebP should export PNG, GIF, APNG, or MP4 and convert externally for now.
_Avoid_: Accidental WebP support, arbitrary image conversion, assuming FFmpeg can encode WebP

**Export progress**:
User-facing export status on stderr that reports rendered frame progress against the export frame count when the total is knowable, and an indeterminate rendered-frame count when it is not. `100%` means Mojify has rendered and written all visual frames to the active output encoder or serializer, after which the status should move to format finalization when applicable; export progress should be terminal-friendly when interactive, log-friendly otherwise, and should not claim an ETA or time remaining by default.
_Avoid_: Export ETA, FFmpeg progress, fake completion estimate

**Export throughput hardening**:
The product utility stage that improves MP4 export speed while preserving ordered frames, output correctness, source audio behavior, and honest export progress.
_Avoid_: Distribution polish, live terminal audio, faster-looking progress

**Export font**:
The bundled monospace font used to rasterize Mojify character frames into exported media. The preferred default is `Mx437_IBM_BIOS`.
_Avoid_: User terminal font, arbitrary system font

**Playback controls**:
The minimal interactive controls available during playback: quit and pause/resume for the active playback experience, including live terminal audio when enabled.
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

**Installable CLI distribution**:
The roadmap phase that makes the proven Mojify CLI available through normal user install paths while preserving the existing `play`, `probe`, and `export` command surface.
_Avoid_: Product utility expansion, plugin ecosystem, release polish

**README header demo**:
The checked-in animated proof asset near the top of the README. It is generated from an original Remotion source animation, transformed through Mojify output, and composed into a GitHub-friendly GIF with a reduced-motion poster.
_Avoid_: Copyrighted sample footage, generic logo loop, required CI render

**Binary release distribution**:
An installable CLI distribution path where Mojify publishes prebuilt command-line binaries as GitHub Release tarballs.
_Avoid_: Source-building Homebrew formula, desktop installer, full Linux distro packaging

**Source-building Homebrew formula**:
The Homebrew install path where the Mojify tap formula downloads a tagged source archive, builds the CLI from source, and installs the resulting `mojify` binary.
_Avoid_: GoReleaser binary formula, cask, prebuilt Homebrew artifact

**Stable tag release**:
A release flow triggered by an explicit calendar + build tag, such as `v2026.06.02.145`, that publishes Mojify install artifacts for that version.
_Avoid_: Nightly release, snapshot build, continuous deployment

**Calendar + build version**:
The stable release version shape for Mojify releases: Git tags use `vYYYY.MM.DD.BUILD`, such as `v2026.06.02.145`, while user-facing version output omits the `v`, such as `mojify 2026.06.02.145`.
_Avoid_: SemVer-shaped calendar tags, Go module major-version bump, build-metadata-only uniqueness

**Release snapshot QA**:
A local, non-publishing release dry run that verifies Mojify binary archive layout, naming, and checksums before a stable tag release.
_Avoid_: Nightly release channel, published prerelease, source-build QA

**Version output**:
The CLI surface that reports the installed Mojify binary version as `mojify VERSION` so users and release QA can confirm which build is running.
_Avoid_: Diagnostics command, full build manifest, update checker

**Runtime dependency hint**:
A command-specific error message that tells users which external tool is missing and how to install it when Mojify cannot run a requested media operation.
_Avoid_: Doctor command, dependency installer, background repair

**WSL-only Windows support**:
The first Windows distribution stance for Mojify, where Windows users run the Linux CLI inside WSL instead of a native Windows binary.
_Avoid_: Native Windows binary, PowerShell-first install, Windows audio backend

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

**Recipe preset**:
A built-in renderer recipe selected by name, giving users a controlled way to change Mojify's character-frame look by combining choices for character mapping, edge behavior, and color mode without loading custom recipe files or scripts.
_Avoid_: Custom recipe, plugin, arbitrary renderer config

**Default recipe preset**:
Mojify's existing renderer recipe: luminance density mapping, source color, and edge glyph overrides.
_Avoid_: Basic renderer, legacy mode, fallback preset

**Mono recipe preset**:
A built-in recipe preset that keeps Mojify's default luminance and edge behavior but disables source/ANSI color for copyable monochrome terminal art.
_Avoid_: Grayscale truecolor, color desaturation, classic ASCII

**ASCII recipe preset**:
A built-in recipe preset that uses a plain ASCII luminance ramp with no source color and no edge glyph override.
_Avoid_: Default recipe without color, edge-aware ASCII, Unicode art

**Blocks recipe preset**:
A built-in recipe preset that uses a Unicode shade/block luminance ramp with source color enabled and edge glyph overrides disabled.
_Avoid_: Braille renderer, emoji recipe, classic ASCII

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
