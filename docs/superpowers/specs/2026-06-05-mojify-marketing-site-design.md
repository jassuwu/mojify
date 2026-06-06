# Mojify Marketing Site - Design Spec

> **Status: superseded in part.** This was the original, larger "premium cinema"
> design. The shipped site (`apps/site`) was pared back to a more minimal page:
> a plain hero over an animated character-art background, one interactive demo
> terminal in place of the per-capability sections, JetBrains Mono throughout,
> and plain non-marketing copy. Treat the sections below as design history and
> rationale; `apps/site` is the source of truth for what actually shipped.

> Canonical, implementation-ready. Single source of truth for `apps/site`. Angle: **Premium Terminal Cinema** - artifact-maximal backbone grafted with conversion sequencing, command-as-connective-tissue, and build-time-text rendering.
>
> Locked scope: **static marketing site, not a web converter.** No cloud conversion, uploads, queues, storage, accounts, or browser-side converter. All interactivity is presentational. The site sells the CLI and drives `brew install jassuwu/tap/mojify`.

---

## 1. Overview & goals

**Positioning.** Mojify turns media into text - specifically into *colored, edge-aware character frames*. Category framing is **MEDIA-TO-TEXT**, terminal-first and truecolor. It is not an ASCII clone, not a general media converter, not an "ASCII toolkit," not a recipe platform, and not a web converter. There is no upload UI anywhere on the site.

**Design thesis.** The product is a transformation, so **the page *is* the transformation.** We render the proof at cinema scale on Mojify's own output background (`#080808`) and let T3-Code restraint carry the premium signal: one confident hero, one tagline, one obvious primary CTA. The hero artifact (source media → CLI command → character-frame output) bleeds past the fold with no bottom border, so the visitor scrolls *into* the output. Every feature section below is one large real artifact captioned by one line. The page is composited from the product's own real output - real `.ansi`/`.txt`/`probe`/`doctor` ship as selectable DOM text, not screenshots - which is simultaneously the aesthetic, the proof, and the accessibility/perf win.

**Primary conversion goal.** Get the visitor to copy `brew install jassuwu/tap/mojify`. The command *is* the primary CTA (a copy-button chip), above the fold and repeated as the page's final beat. Nothing to sign up for; `mojify doctor` is the trust closer.

**What success looks like.**
- A decided developer copies the install command in under ten seconds (it's above the fold).
- A skeptic scrolls and is convinced by real artifacts (live-play capture, four-recipe comparison, real truecolor ANSI as selectable text, the `doctor` summary) - not by claims.
- Zero accuracy violations: nobody leaves believing Mojify outputs emoji, exports WebP, runs native Windows, accepts uploads, or seeks/zooms playback.
- Lighthouse ~100 on the static path; initial above-the-fold transfer well under budget; the page never reads as a generic SaaS layout.

---

## 2. Locked copy

Every string below is canonical. Do not paraphrase locked strings. CLI snippets use only the approved set and real flags.

### 2.1 Hero copy decision

- **Wordmark / h1:** `mojify` (lowercase).
- **Tagline (h2) - CHOSEN:** `Turn media into text.`
  - Rationale: it is the primary canonical tagline; it states the category plainly. The punchy alternate `Media goes in. Text comes alive.` is **A/B-gated only** - kept behind a build flag, never rendered simultaneously with the primary tagline. Pick one per build.
- **Support line:** `Play videos live or export colored, edge-aware Mojify output as video, animated, still-image, or text files.` (the only place the four export families are enumerated above the fold).
- **Primary CTA (command-as-button):** `brew install jassuwu/tap/mojify` with a trailing copy glyph; copied state → `Copied`.
- **Secondary CTA:** `Watch it play` (smooth-scrolls to "Play it live").
- **Tertiary CTA:** `GitHub ↗` → `https://github.com/jassuwu/mojify`.
- **Nav:** wordmark `mojify` left; anchors `play · recipes · export · install` center/right; `GitHub ↗` ghost button far right.

### 2.2 Section headlines + body (verbatim)

| Section | Headline | Body (one line) |
|---|---|---|
| Play it live | `Play it live.` | `Colored, edge-aware character frames in your terminal - with the source audio playing, on by default.` |
| Bring your own media | `Bring your own media.` | `A local video, a still PNG or JPEG, or a yt-dlp-compatible link - Mojify resolves it, then renders.` |
| Recipes change the look | `Recipes change the look.` | `Four built-in presets - from the colorful edge-aware default to flat retro text and a colored block mosaic.` |
| Export formats | `Export the weird stuff.` | `One command, output family chosen by extension - video, animated, still image, or text.` |
| Built on the right tools | `Built on the right tools.` | `FFmpeg does the decoding. yt-dlp resolves the links. Mojify owns the look.` |
| Install + final CTA | `Install in one line.` | `Homebrew pulls Mojify plus FFmpeg and yt-dlp. Or grab a release tarball.` |

> Conflict resolution: `Export the weird stuff.` is kept (it is the single most-quoted candidate headline and a deliberate, earned playful beat) but the body line carries the substance, and the honest WebP/audio captions immediately ground it. Voice stays grounded everywhere else.

### 2.3 Honest micro-captions (prompt-grey; treat as required on-page copy, not optional polish)

- Play it live: `Controls: quit, pause/resume. Add --no-audio for silence.`
- Bring your own media: `Stills are probe + single-frame export only - they don't play. Playlists and live streams are rejected.`
- Recipes: `Built-in presets only - default · mono · ascii · blocks.`
- Export - flags line: `--width --fps --bitrate --at --duration --overwrite --stats --workers --recipe`
- Export - audio/format honesty: `Video keeps source audio. GIF and APNG carry no audio. No WebP.`
- Built on the right tools (bridge): `Run mojify doctor to see what's installed.`
- Install: `macOS and Linux. Windows is WSL-only.` · `Releases are versioned vYYYY.MM.DD.BUILD.`

### 2.4 Real CLI snippets (the only commands allowed on the page)

```
brew install jassuwu/tap/mojify
mojify doctor
mojify play intro.mp4
mojify play --recipe blocks "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
mojify export --width 120 --fps 24 --at 1:05 --duration 8 clip.mp4 demo.gif
mojify export --recipe ascii poster.png poster.ansi
mojify probe poster.png
```

---

## 3. Design system

### 3.1 Color tokens

| Token | Hex | Role |
|---|---|---|
| `--field` | `#080808` | The only background. Every section, every terminal stage. **Never `#000`** - this is Mojify's own output bg, so real character art composites with zero seam. Load-bearing. |
| `--mint` | `#7cffc6` | Brand accent, **rationed to exactly these jobs**: wordmark, highlighted command verb/token (`mojify`, `brew install`, recipe names), block caret, the scan-line wipe, primary-CTA border/accent, active tab underline, `doctor` ✓ / required status, focus rings, link hover. If everything is mint, nothing is. |
| `--prompt` | `#8a99ad` | Shell prompt (`$`, `~/mojify ❯`), inactive nav, all honest captions/micro-notes, secondary borders, optional-tool status, footer. |
| `--offwhite` | `#e8f3ee` | All body + tagline text (mint-tinted so it harmonizes on field). |

Mint-rationing is a literal token-deployment law, not a guideline.

### 3.2 Type scale + fonts

**Display / wordmark:** one self-hosted **heavy black grotesque, weight 900**, `font-display: swap`, subset to the glyphs actually used (`mojify` + headline characters). Recommendation: **Archivo Black** (Google Fonts, OFL, single 900 weight, blocky/heavy - closest free match to the Arial Black 900 wordmark the README header uses and that survives character conversion). Fallback stack: `"Archivo Black", "Arial Black", "Arial Bold", sans-serif`. The hosted face must be visually verified against the committed `mojify-header.gif`/poster wordmark that sits beside it (registration matters). ~15-25 KB subset woff2.

**Mono (the page's dominant voice - commands, captions, ANSI/probe/doctor grids, ramps, nav mark, chips):** system stack only, **zero font bytes**:
`ui-monospace, "SF Mono", SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace`

**Body prose:** system sans (`system-ui, sans-serif`) for the longest support paragraphs; mono for short terminal-voice copy.

| Token | Use | Size (desktop → mobile) | Family / weight |
|---|---|---|---|
| `display-xl` | wordmark `mojify` | `clamp(64px, 12vw, 160px)` | grotesque 900 |
| `display-l` | section headlines | `clamp(40px, 6vw, 88px)` | grotesque 900 |
| `tagline` | hero h2 | `clamp(28px, 4vw, 56px)` | grotesque 800 |
| `body` | support / body lines | `clamp(16px, 1.4vw, 20px)` | system sans 400-500, off-white |
| `mono-cmd` | command chips | `clamp(15px, 1.2vw, 18px)` | mono 400/600 |
| `mono` | snippets, captions, chips | `clamp(13px, 1.1vw, 16px)` | mono 400/600 |
| `mono-grid` | ANSI / txt / doctor / probe grids | `clamp(11px, 0.9vw, 14px)` | mono 400 |
| `caption` | honest micro-notes | `13-14px` | mono, prompt-grey |

Keep the display face blocky; never pair with a thin display face.

### 3.3 Spacing / rhythm

8px base unit. Section vertical padding `clamp(96px, 14vh, 200px)` - the generous black is the premium signal. Content max-width `1200px`; prose capped ~58ch; command/code panels max `720px` (short, copy-able lines). Intra-section stack: command → 16px → headline → 12px → body → 32px → artifact → 12px → caption. Sections separated by whitespace or a 1px `#141414` hairline - **never boxes**.

### 3.4 Motifs

1. **3px mint scan-line wipe** - `linear-gradient(90deg, transparent, #7cffc6 22%, #7cffc6 78%, transparent)` at `opacity: 0.55`. Triple duty: (a) the hero one-shot reveal sweep; (b) horizontal connectors/arrows in the input + tools sections; (c) thin section dividers and the active-recipe-tab underline. Animated only as the hero reveal and a one-shot scroll-enter sweep per section; never ambient looping. Reduced-motion: render at final state.
2. **Mint block caret** - solid mint rectangle, `blink` via `steps(2)` opacity ~1s. Parked after the nav wordmark, at the end of every command line, and as the footer signature. Reduced-motion: solid, no blink. Keep to ~3 placements max to avoid over-decoration.
3. **Character ramps as decorative texture** - the literal ramps (`" .;coPO?#@"`, `" .:-=+*#%@"`, `" ░▒▓█"`) as low-opacity (`~0.04`) field-on-field mono text behind the hero/empty margins, and as recipe-tile caption labels. Always real text, `aria-hidden`, `user-select: none`. Turn a real product detail into ornament - sparingly.

### 3.5 Dark-only

**Dark only, intentionally.** No theme toggle (one fewer chrome element). The seamless-composite premise *requires* `#080808`; a light mode would break it. Set `color-scheme: dark` and `<meta name="color-scheme" content="dark">`. Provide a print stylesheet that flattens to legible mono.

### 3.6 Responsive breakpoints

| Breakpoint | Behavior |
|---|---|
| `<640px` (mobile) | Single column. Hero stacks: text → full-bleed artifact that still overflows the fold. Three-stage transformation restacks **vertically**, scan-line connector rotated to point downward (preserves the L→R reading as top→bottom). Recipe grid → tabbed single tile. 24px gutters. |
| `640-1023px` (tablet) | Single column, larger type; artifacts ~90vw. |
| `≥1024px` (desktop) | Two-column hero (text left ≤540px, artifact stage right, taller than viewport). Recipes 2×2. Section copy/artifact alternate sides for rhythm. 64px gutters. |
| `≥1440px` (reference) | Layout as drawn in §4. Content max 1200px; artifacts full-bleed or 70vw. |

---

## 4. Page structure & sections

Global: continuous `#080808` field; no alternating backgrounds. **Every section opens with one real, copy-pasteable command line - the exact line that produced the artifact below it** (command-as-connective-tissue). Pattern per section: `command → headline → body → artifact → caption`. All interactivity is presentational (hover, tabs, copy, pre-rendered playback). No upload, no dropzone, no live conversion, no progress bars, no ETAs.

### 4.0 Hero - above the fold

**Purpose.** Render the locked transformation at cinema scale and place the primary CTA above the fold; bleed the output past the fold so scrolling = scrolling into the product.

**Exact copy.** Wordmark `mojify`; tagline `Turn media into text.`; the support line; primary CTA `brew install jassuwu/tap/mojify` (copy); secondary `▶ Watch it play`; tertiary `GitHub ↗`. (See §2.1.)

**Layout - desktop (≥1024px).**
```
┌──────────────────────────────────────────────────────────────┐
│ nav: mojify▋        play · recipes · export · install  [GitHub↗]│
├──────────────────────────────────────────────────────────────┤
│ mojify  (display 900)              ┌──────────────────────────┐ │
│                                    │ ~/mojify ❯  (terminal top) │ │
│ Turn media into text.              │  [ source still ]          │ │
│ ── support line, ≤2 lines ──       │  $ mojify play intro.mp4 ▋ │ │
│                                    │  ── mint scan-line wipe ── │ │
│ [ brew install …/mojify ⧉ ]        │  [ colored char frames ]   │ │
│ [ ▶ Watch it play ] [ GitHub ↗ ]   │  ▼ output bleeds past fold │ │
│──────────────────────────────── fold │  (no bottom border) ▼     │
└──────────────────────────────────────────────────────────────┘
```
- Left = text column (max ~540px). Right = the **hero artifact stage**, taller than the viewport, **clipped only by the page edge - never by a card border at the bottom**. Stage bg `#080808`; **no bottom border, no rounded bottom, no base shadow** so character frames look like they emerge from the page. Top of the stage gets a 1px `#1a1a1a` hairline + a faint `~/mojify ❯` prompt to read as a terminal; the bottom is open. (Load-bearing premium+identity mechanic - non-negotiable.)

**Layout - mobile.** Text stacks above; artifact goes full-bleed width and still overflows the fold; three stages stack vertically with the scan-line rotated downward.

**Artifact + how produced.** The **CLI-Tunnel** sequence (a pre-rendered loop - interactivity presentational only):
1. Frame 0: the **source** still in the terminal window.
2. The command **types in** at the prompt with the **mint block caret** blinking: `mojify play intro.mp4`; tokens `mojify`/`play` rendered mint, path off-white.
3. On "Enter," the **3px mint scan-line** sweeps top→bottom across the stage (the wipe).
4. In the wipe's wake, the source resolves into **real colored, edge-aware character frames** (truecolor glyph grid with `/ \ | -` edges) - the output that *continues below the fold*.
5. Holds on living output ~4s, soft-cuts back to Frame 0.

Produced by re-rendering the `apps/readme-header` Remotion composition at marketing resolution (2×), source = a striking clip; ship as encoded `<video muted autoplay loop playsinline>` with a poster. Stopgap before regen: the committed `docs/assets/readme/mojify-header.gif` + `mojify-header-poster.png`. (Aspect mismatch is a known gap - regen to a viewport-bleeding aspect.)

**Token usage.** Stage bg `--field`; wordmark/caret/scan-line/command-token highlights `--mint`; prompt + nav-inactive `--prompt`; tagline/body `--offwhite`. Primary CTA = field bg + 1px mint border + mint text, hover fills mint at low alpha; secondary/GitHub = prompt-grey border + off-white text (clearly subordinate).

**Presentational interactions.** Hover on command tokens brightens; copy button on the brew chip. Nothing else. Reduced-motion → static post-wipe output poster, no autoplay.

---

### 4.1 Play it live

- **Purpose.** Prove the core product shape - live terminal playback with audio - and absorb the hero's "Watch it play."
- **Command (connective):** `mojify play intro.mp4`
- **Copy.** Headline `Play it live.` · Body `Colored, edge-aware character frames in your terminal - with the source audio playing, on by default.`
- **Caption.** `Controls: quit, pause/resume. Add --no-audio for silence.`
- **Second command shown:** `mojify play --recipe blocks "https://www.youtube.com/watch?v=dQw4w9WgXcQ"`
- **Artifact + production.** A **live-play terminal capture** - screen recording of `mojify play` in a real terminal (minimal chrome: top hairline + three muted dots + prompt; frames updating; a static `AUDIO ●` *label* pill, not a control). Regenerate by screen-recording `mojify play <clip>.mp4` → trim → encode to muted looped `<video>` + poster. **This is the one section whose headline asset must be captured from scratch** (highest-effort asset; see §5). The visual loops; a static "audio plays live" annotation communicates sound - never autoplay audio. A non-functional text row `␣ pause   q quit` shows the *real* control set. **No seek/scrub/speed/zoom UI anywhere.**
- **Layout.** Desktop: copy left (~40%), terminal video right (~60%). Mobile: stacked, copy first.
- **Interactions.** Hover → subtle mint border-glow + caret resumes blink. The audio pill is a label. Reduced-motion → first-frame poster.

---

### 4.2 Bring your own media

- **Purpose.** Set accurate input expectations and turn the limits into a designed beat.
- **Command (connective):** `mojify probe poster.png`
- **Copy.** Headline `Bring your own media.` · Body `A local video, a still PNG or JPEG, or a yt-dlp-compatible link - Mojify resolves it, then renders.`
- **Caption (honest, required).** `Stills are probe + single-frame export only - they don't play. Playlists and live streams are rejected.`
- **Commands shown:** `mojify play intro.mp4` · `mojify probe poster.png` · `mojify export --recipe ascii poster.png poster.ansi`
- **Artifact + production.** A **three-chip input legend** as live DOM pills - `video · play + export` / `PNG / JPEG still · probe + export only` / `yt-dlp URL · single video only` - with the **rejected cases (`playlist`, `live`) shown struck-through** in prompt-grey (honesty as design). Beside it, a real **`mojify probe` output block** rendered as **selectable mono text** (captured verbatim from `mojify probe dist/hmc.png`), doubling as proof the command is real. Optionally a source→output cross-fade using committed pairs (`hmc.png → hmc-text.png`, `spirited.png → spirited-text.png`, `son.png → son-text.png`).
- **Layout.** Two columns desktop (input legend + probe text). Single column mobile.
- **Interactions.** Hover a source thumb → big output panel cross-fades to that pair's `-text.png`. Copy button on the probe command only. Presentational.

---

### 4.3 Recipes change the look

- **Purpose.** The visual money shot - one clip, four built-in looks. **The ONE justified multi-artifact section.**
- **Command (connective):** `mojify play --recipe blocks "https://www.youtube.com/watch?v=dQw4w9WgXcQ"`
- **Copy.** Headline `Recipes change the look.` · Body `Four built-in presets - from the colorful edge-aware default to flat retro text and a colored block mosaic.`
- **Caption.** `Built-in presets only - default · mono · ascii · blocks.`
- **Per-tile captions (exact, accurate to ramp/color/edge facts):**
  - `default - " .;coPO?#@" ramp · source color · edge glyphs` (the hero look)
  - `mono - same ramp + edges, no color (white on black)`
  - `ascii - " .:-=+*#%@" ramp, no color, no edges (flat retro)`
  - `blocks - " ░▒▓█" ramp + color, no edges (block mosaic)`
- **Artifact + production.** The committed 4-up of one clip: `dist/redeyes.gif` (default), `dist/redeyes-mono.gif`, `dist/redeyes-ascii.gif`, `dist/redeyes-blocks.gif`. **2×2 desktop / stacked mobile.** Default loaded eager; the other three lazy-load on tab activation. **Must be re-encoded to muted looped `<video>`/WebM** before shipping (raw `redeyes-blocks.gif` is 5.6 MB - see §5/§7; hard decision, not "consider").
- **Layout.** Four equal tiles, 4px field gutters so they read as one strip; tile labels mono prompt-grey, recipe-name token mint, active tile 1px mint frame. Narrow viewports: tab row `default · mono · ascii · blocks` swapping one large tile.
- **Interactions.** Tabs swap which pre-rendered video plays (src-swap of already-loaded assets). Hover reveals the tile's `--recipe NAME`. One copy button on the full `mojify play --recipe blocks …` line. Do not imply any per-recipe quality ranking.

---

### 4.4 Export formats

- **Purpose.** Demonstrate output breadth with one extension-routed mental model.
- **Command (connective):** `mojify export --width 120 --fps 24 --at 1:05 --duration 8 clip.mp4 demo.gif`
- **Copy.** Headline `Export the weird stuff.` · Body `One command, output family chosen by extension - video, animated, still image, or text.`
- **Second command shown:** `mojify export --recipe ascii poster.png poster.ansi`
- **Caption (honest).** `Video keeps source audio. GIF and APNG carry no audio. No WebP.`
- **Flags micro-caption.** `--width --fps --bitrate --at --duration --overwrite --stats --workers --recipe`
- **Artifact + production.** A **routing table** (mono, live text) mapping extension family → honest note, paired with a **format filmstrip** whose chips swap one large preview:
  - **Video** `.mp4 .webm .mov` - *preserves source audio when present* → `dist/qa/export/low-motion-bars-export.{mp4,webm,mov}`
  - **Animated** `.gif .apng` - *no audio* → `low-motion-bars-export.{gif,apng}`
  - **Still** `.png .jpg .jpeg` - single frame → `recipe-default.png`, `still-source-output.{jpg,jpeg}`
  - **Text** `.txt` (flat) · `.ansi` (truecolor) → `low-motion-bars-frame.txt`, `recipe-default.ansi`
  - The **`.txt`/`.ansi` previews render as real, selectable, copyable truecolor DOM text** parsed from the actual `.ansi` (38;2;R;G;B spans) at Astro build time - the load-bearing "it's real text, a character frame, not an ASCII image" proof.
- **Layout.** Desktop: routing table + flag chips left; one large preview + chip filmstrip right. Mobile: stacked. **WebP chip appears nowhere.** Video chips carry a tiny `audio-capable` tag; GIF/APNG carry `no audio`.
- **Interactions.** Chip selects preview; hover a table row highlights the matching chip. Copy buttons on both command lines and the raw ANSI block.

---

### 4.5 Built on the right tools

- **Purpose.** Credibility via honest plumbing; the calm typographic interstitial between heavy artifacts.
- **Command (connective):** `mojify doctor`
- **Copy.** Headline `Built on the right tools.` · Body `FFmpeg does the decoding. yt-dlp resolves the links. Mojify owns the look.`
- **Caption.** `ffmpeg + ffprobe required · ffplay + yt-dlp optional`
- **Bridge line.** `Run mojify doctor to see what's installed.`
- **Artifact + production.** A minimal typographic diagram - mono labels connected by the **scan-line gradient** as connectors: `source → ffmpeg/ffprobe → mojify → output`, side branch `URL → yt-dlp →`. No logos required (if used, monochrome text on field). Not a big render - the eye rests here.
- **Layout.** Centered, narrow (max 720px), lots of field around it.
- **Interactions.** None (or a single hover that brightens each tool node). Deliberately calm.

---

### 4.6 Install + final CTA

- **Purpose.** Convert. Frictionless primary path, discoverable fallback, doctor as the trust capstone.
- **Command (connective / climax):** `brew install jassuwu/tap/mojify`
- **Copy.** Headline `Install in one line.` · Body `Homebrew pulls Mojify plus FFmpeg and yt-dlp. Or grab a release tarball.`
- **Primary block (copy):** `brew install jassuwu/tap/mojify` - caption `macOS and Linux. Windows is WSL-only.`
- **Verify block (copy):** `mojify doctor` - caption `Checks ffmpeg + ffprobe (required) and ffplay + yt-dlp (optional), then prints a plain-English summary.`
- **Doctor artifact.** A real `mojify doctor` capture rendered as **selectable colored mono text** (✓ mint, labels off-white, `(required)`/`(optional)` prompt-grey, off-white summary line). Captured verbatim from a real run - never hand-faked.
- **Fallback (lower contrast).** `Prefer a tarball? Download from GitHub Releases - macOS and Linux, arm64 or amd64. Tarball users install ffmpeg, ffprobe, ffplay, and yt-dlp themselves.` → `https://github.com/jassuwu/mojify/releases`. Version note: `Releases are versioned vYYYY.MM.DD.BUILD.`
- **Final CTA.** Repeat the wordmark small + `Turn media into text.` + the brew copy button full-width (mint border) + `▶ Watch it play` and `GitHub ↗`. Same primary action as the hero, now with all proof behind it. A single mint scan-line underlines this closing beat on entry.
- **Layout.** Narrow centered column; primary / verify / fallback stacked; final CTA centered below - all on `#080808` so the doctor panel bg matches.
- **Interactions.** Copy buttons on every command; `Copied` toast + caret blink.

---

### 4.7 Footer

- **Purpose.** Minimal signature.
- **Copy.** `mojify · MEDIA-TO-TEXT · MIT · made with FFmpeg + yt-dlp` + `GitHub ↗ · Releases ↗` links, prompt-grey mono. Trailing blinking mint caret as signature.
- **Layout.** Single mono row, centered/left. No newsletter, no social grid.

---

## 5. Asset production plan

Budgets are per-asset transfer targets (encoded, gzipped where applicable). **Hard rule: ship encoded `<video>`/AVIF/WebP page-assets only - never raw GIF/MP4 source.** (Site-internal WebP for its own thumbnails is fine and must never be implied as a Mojify export capability.)

| Artifact | Reuse / Regenerate | Exact command or source | Target dims / budget |
|---|---|---|---|
| Hero CLI-Tunnel (stopgap) | Reuse | `docs/assets/readme/mojify-header.gif` + `mojify-header-poster.png` | as-is until regen |
| Hero CLI-Tunnel (final) | Regenerate | Re-render `apps/readme-header` Remotion comp at 2×, source = striking clip; export `<video>` + poster | ~tall stage aspect; video ≤ 1.2 MB, poster AVIF ≤ 120 KB |
| **Live-play terminal capture** | **Regenerate (from scratch)** | Screen-record `mojify play <clip>.mp4` in a real terminal → trim → muted looped `<video>` + poster | ~60vw; video ≤ 1.5 MB |
| Per-recipe marketing strip | Reuse → re-encode | `dist/redeyes.gif`, `dist/redeyes-{mono,ascii,blocks}.gif` → muted looped WebM/MP4 `<video>` | each ≤ 600 KB encoded (raw blocks.gif is 5.6 MB - must convert) |
| Recipe stills (reduced-motion) | Reuse | `dist/qa/export/recipe-{default,mono,ascii,blocks}.png` | AVIF ≤ 80 KB each |
| Source→output pairs | Reuse | `dist/{hmc,son,spirited}.png` + `-text.png`; `dist/portrait-bw.png` + `dist/bw-portrait-text.png` | AVIF, lazy |
| Export filmstrip - video | Reuse | `dist/qa/export/low-motion-bars-export.{mp4,webm,mov}` | poster + lazy `<video>` |
| Export filmstrip - animated | Reuse → re-encode | `dist/qa/export/low-motion-bars-export.{gif,apng}` → `<video>`/AVIF | ≤ 400 KB |
| Export filmstrip - still | Reuse | `dist/qa/export/recipe-default.png`, `still-source-output.{jpg,jpeg}` | AVIF ≤ 80 KB |
| Export - real ANSI (selectable text) | Reuse | `dist/qa/export/recipe-default.ansi` → parsed to colored `<pre>` at build time | text, ~0 raster |
| Export - real TXT (selectable text) | Reuse | `dist/qa/export/low-motion-bars-frame.txt` → plain `<pre>` | text |
| Probe output block | Regenerate | `mojify probe dist/hmc.png` → capture stdout verbatim → mono `<pre>` | text |
| Doctor output block | Regenerate | `mojify doctor` → capture stdout verbatim → colored `<pre>` | text |
| Optional fresh recipe clip | Regenerate | `mojify export --recipe {default,mono,ascii,blocks} --width 140 <clip> out.gif` then encode to `<video>` | ≤ 600 KB each |

All regen uses **only** real flags (`--width --fps --bitrate --at --duration --overwrite --stats --workers --recipe`) and real recipe names. The `.ansi`/`.txt`/`probe`/`doctor` blocks ship as **real text, never screenshots.**

---

## 6. Accuracy guardrails

The site must **NOT** imply any of the following. Each item below is a pre-ship checklist line.

- **No emoji output.** Despite the name, never show/imply emoji. Output is always `colored, edge-aware character frames` / `character frame` - never "ASCII image," "text bitmap," or "emoji." Decorative UI glyphs (`▋ ● ↗ · ⧉`) are chrome, never presented as product output.
- **Not an ASCII clone / converter / toolkit / recipe platform.** Category stays **MEDIA-TO-TEXT**. Recipes are described strictly as **built-in presets** (`--recipe NAME`) - never user-authored files or a platform. No `--recipe-file`, no "bring your own ramp."
- **No WebP export.** Format lists contain exactly `.mp4 .webm .mov · .gif .apng · .png .jpg .jpeg · .txt .ansi`. WebP appears nowhere.
- **No web converter / dropzone / desktop app / native Windows.** Zero upload UI. Install says **macOS + Linux; Windows is WSL-only.** No progress bars, no ETAs.
- **Stills are bounded.** PNG/JPEG only; **probe + single-frame export only; not playable.** `--at`/`--duration` never shown on still examples (no timeline). Stills shown via `mojify probe`, never `mojify play poster.png`.
- **URLs are bounded.** yt-dlp-compatible only; **playlists and live streams rejected** (stated on page).
- **Playback controls bounded.** **quit + pause/resume only - no seek/speed/zoom.** Audio default-on; `--no-audio` to disable. The audio pill is a label, not a control.
- **Audio scope is exact.** Live audio in `play`; **video exports preserve source audio when present**; **GIF/APNG carry no audio** (chips labeled accordingly). No audio for stills/text.
- **Recipe color/edge truth table.** color: `default`+`blocks` yes, `mono`+`ascii` no; edges: `default`+`mono` yes, `ascii`+`blocks` no. Ramps verbatim per §4.3. No quality ranking implied.
- **Commands are real.** Only the four commands (`play · probe · export · doctor`; bare `mojify` → help) and only the approved flag list. No invented flags. No export ETAs (`--stats`/`--workers` listed without speed claims).
- **Install honesty.** `brew install jassuwu/tap/mojify` declares ffmpeg + yt-dlp; tarball users self-install ffmpeg/ffprobe/ffplay/yt-dlp. Versioning `vYYYY.MM.DD.BUILD`.
- **Enforcement.** Store every CLI string as a single-source-of-truth constant; tokenize spans programmatically; lint rendered command text against the locked snippet list in CI to prevent tokenization typos.

---

## 7. Accessibility & performance

**Reduced motion (`prefers-reduced-motion: reduce`).** Mirror the README `<picture>`/poster pattern: hero `<video>` → post-wipe output poster (`mojify-header-poster.png` as stopgap); every recipe/play `<video>` → its first-frame poster still (all exist in `dist`); scan-line sweeps render at final state; caret solid (no blink); tab/chip swaps instant (no cross-fade); no autoplay anywhere. No information is motion-only.

**Contrast (WCAG on `#080808`).** off-white `#e8f3ee` ≈ 17:1 (AAA); mint `#7cffc6` ≈ 14:1 (safe for text + large UI); prompt-grey `#8a99ad` ≈ 6.3:1 (AA for body - reserved for *secondary* text/captions; never the sole signal for an interactive state - pair with border/underline). Focus rings 2px mint, always visible. Copy buttons/tabs are real `<button>`/`role="tablist"` with keyboard nav; copy announces via `aria-live="polite"` ("Copied").

**Image-vs-text budget.** Ship as **real selectable text**: wordmark, all headings/body, every command chip, ramps (`aria-hidden`), the truecolor `.ansi` grid, `.txt` grids, probe output, doctor output, format chips, nav. Ship as **raster/video only** the true motion/photographic proof: hero loop, play capture, recipe comparisons, source→output stills. **ANSI accessibility paradox:** decorative ANSI grids are `aria-hidden` with a concise text alternative describing the transformation (e.g. `alt="A film still rendered as colored edge-aware character frames using the default recipe"`); announced text is reserved for actual content (commands, probe/doctor).

**Weight target.** Initial above-the-fold ≤ ~600 KB transferred (HTML + CSS + subset display font ~15-25 KB + hero poster AVIF + first-paint). Hero video `preload="none"` until in view. All below-fold artifacts `loading="lazy"`, `<video preload="none">`, recipe/format alternates fetched on tab/chip activation. **Re-encode every GIF → `<video>`** (5-10× smaller). Full-scroll budget ≤ ~3.5 MB with everything lazy; LCP element is the hero poster/text, not a GIF. Lighthouse target ~100 on the static path.

---

## 8. Tech stack & file layout

**App.** `apps/site`, package `@mojify/site` (private). Build output to `dist/**` (turbo-cached `build` task - already configured for `dist/**`). Scripts: `dev` / `build` / `typecheck`. Self-contained `tsconfig.json` (no root tsconfig). Prettier 2-space, no tabs. Bun 1.3.14 monorepo.

**Bundler - Astro (static) + React 19 islands + Tailwind v4.** Rationale: the locked direction is a content/artifact page that is ~95% static HTML/CSS with three small interactive surfaces; Astro ships ~0 JS by default, builds to static `dist/**` (slots into the cached turbo task), supports React 19 islands and Tailwind v4 via `@tailwindcss/vite`, and reads `.ansi`/`.txt` at build time to emit colored `<pre>` DOM. This beats Next.js static export for this shape (less runtime, simpler static output) while keeping the only existing UI precedent (React 19 + Tailwind v4, as in `apps/readme-header`).

**React islands - exactly three interactive surfaces:** hero player, recipe tabs, format filmstrip. Everything else is static Astro HTML. Copy buttons are tiny shared client components.

**Brand tokens → Tailwind theme.** Define in a Tailwind v4 `@theme` block as CSS custom properties, consumed as utilities and in component CSS:
```css
@theme {
  --color-field:    #080808;
  --color-mint:     #7cffc6;
  --color-prompt:   #8a99ad;
  --color-offwhite: #e8f3ee;
  --font-display: "Archivo Black", "Arial Black", "Arial Bold", sans-serif;
  --font-mono: ui-monospace, "SF Mono", SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
}
```

**Proposed `src/` tree.**
```
apps/site/
  astro.config.mjs            # @astrojs/react, @tailwindcss/vite, static output
  tailwind.config / @theme    # tokens above
  tsconfig.json               # self-contained
  package.json                # @mojify/site, private, dev/build/typecheck
  public/                     # encoded posters, AVIF, <video> sources
  src/
    pages/index.astro
    styles/global.css         # @theme tokens, color-scheme:dark, print sheet
    lib/
      commands.ts             # single-source-of-truth CLI strings (CI-linted)
      ansi.ts                 # build-time .ansi -> colored <pre> parser
    components/
      Nav.astro
      Hero.astro              # stage + bleed; mounts HeroPlayer island
      islands/
        HeroPlayer.tsx        # React island #1
        RecipeTabs.tsx        # React island #2
        FormatFilmstrip.tsx   # React island #3
        CopyButton.tsx
      CommandChip.astro       # prompt + tokenized command + CopyButton
      Section.astro           # command -> headline -> body -> artifact -> caption
      AnsiBlock.astro         # build-time colored <pre> (selectable)
      TextBlock.astro         # plain .txt / probe / doctor <pre>
      RecipeGrid.astro
      FormatTable.astro
      InstallBlock.astro
      ScanLine.astro · Caret.astro · RampTexture.astro
      Footer.astro
```

---

## 9. Implementation plan

Ordered, checkable. `[R]` = needs a real `mojify` run; `[F]` = pure frontend.

**Phase 0 - Scaffold**
- [ ] `[F]` Create `apps/site` (`@mojify/site`, private); Astro static + `@astrojs/react` (React 19) + `@tailwindcss/vite` (Tailwind v4); scripts `dev`/`build`/`typecheck`; self-contained `tsconfig.json`; Prettier 2-space.
- [ ] `[F]` Wire `build` → `dist/**` into the turbo `build` task; confirm cache hit.

**Phase 1 - Design system**
- [ ] `[F]` `@theme` tokens (`--field/--mint/--prompt/--offwhite`, display/mono fonts); `color-scheme: dark`; print stylesheet.
- [ ] `[F]` Self-host + subset the heavy grotesque (Archivo Black); visually verify registration against `mojify-header.gif`.
- [ ] `[F]` Type scale, spacing rhythm, motif primitives (`ScanLine`, `Caret`, `RampTexture`); reduced-motion variants.
- [ ] `[F]` `lib/commands.ts` single-source CLI constants; `CommandChip` + `CopyButton`; `lib/ansi.ts` build-time parser.

**Phase 2 - Hero**
- [ ] `[F]` Two-column stage, no-bottom-border bleed, terminal top hairline/prompt; mobile vertical restack.
- [ ] `[F]` `HeroPlayer` island: poster-first `<video>`, reduced-motion poster swap; token-highlight hover.
- [ ] `[F]` Primary brew CTA chip + secondary/tertiary; copy + toast.

**Phase 3 - Sections (static + islands)**
- [ ] `[F]` `Section` shell (command → headline → body → artifact → caption) for 4.1-4.6.
- [ ] `[F]` Play it live; Bring your own media (input legend w/ struck-through rejects + probe `<pre>`).
- [ ] `[F]` Recipes (`RecipeTabs` island, 2×2 → tabbed, lazy-load non-default).
- [ ] `[F]` Export (`FormatFilmstrip` island, routing table, build-time `AnsiBlock`/`TextBlock`).
- [ ] `[F]` Built on the right tools (scan-line connector diagram); Install + final CTA; Footer.

**Phase 4 - Asset regeneration**
- [ ] `[R]` Re-render Remotion hero comp at 2× → hero `<video>` + poster.
- [ ] `[R]` Record live-play terminal capture → muted looped `<video>` + poster.
- [ ] `[R]` `mojify probe dist/hmc.png` and `mojify doctor` → capture stdout verbatim for text blocks.
- [ ] `[F]` Re-encode all GIFs (`redeyes-*`, `low-motion-bars-*`) → WebM/MP4 `<video>` + AVIF posters; wire reuse-as-is stills/ANSI/TXT.

**Phase 5 - Polish & responsive**
- [ ] `[F]` Mobile/tablet/desktop breakpoints; scan-line connector rotation on mobile; section rhythm/alternation.
- [ ] `[F]` Guard the one-artifact-one-line rule; trim any creeping chrome/decoration.

**Phase 6 - Accessibility**
- [ ] `[F]` Reduced-motion full pass; focus rings; `role="tablist"` keyboard nav; `aria-live` copy; ANSI `aria-hidden` + text alternatives; semantic headings.
- [ ] `[F]` Contrast audit; print sheet check.

**Phase 7 - QA**
- [ ] `[F]` CI lint: rendered command text === `lib/commands.ts` constants; assert no `webp`/emoji/forbidden strings in output HTML.
- [ ] `[F]` Run §6 guardrail checklist against the built page.
- [ ] `[F]` Lighthouse + weight-budget verification (≤600 KB above-fold, ≤3.5 MB full scroll); seam QA on the hero triptych at `#080808`.

---

## 10. Open questions / decisions for the user

1. **Deploy target.** Where does the static `dist/**` ship - Vercel, Netlify, Cloudflare Pages, or GitHub Pages? (Affects only headers/redirects config, not the build.)
2. **Hosted display font.** Approve **Archivo Black** (OFL, free, blocky 900) as the hosted grotesque, or license a specific Arial-Black-class face to register exactly against the README wordmark?
3. **Live-play terminal capture.** The one headline asset that must be recorded from scratch (highest effort/risk). Confirm clip + recipe + capture tooling/terminal.
4. **Tagline A/B.** Ship `Turn media into text.` as default (recommended). Wire the alt `Media goes in. Text comes alive.` behind a build flag, or omit?
5. **Hero clip choice.** Single-hero fragility is the top risk - which source clip best represents the product for the hero CLI-Tunnel?
