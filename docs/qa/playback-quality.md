# Playback Quality QA

Playback quality hardening uses generated synthetic clips as the repeatable baseline and ignored real clips as optional manual references.

## Generate Clips

```bash
bun run qa:clips
```

Expected generated files:

- `dist/qa/low-motion-bars.mp4`
- `dist/qa/high-motion-testsrc.mp4`
- `dist/qa/high-contrast-grid.mp4`

## Manual Runs

Run each clip with stats:

```bash
bun run build
./bin/mojify play --stats dist/qa/low-motion-bars.mp4
./bin/mojify play --stats dist/qa/high-motion-testsrc.mp4
./bin/mojify play --stats dist/qa/high-contrast-grid.mp4
```

Optional local real clips can also be run from ignored `dist/`:

```bash
./bin/mojify play --stats "dist/Call of The Night - Opening ｜ 4K ｜ 60FPS ｜ Creditless ｜ [L96VbQ9ytWk].webm"
./bin/mojify play --stats "dist/米津玄師  Kenshi Yonezu - IRIS OUT [LmZD-TU96q4].webm"
```

## Visual Checklist

For each clip:

- Playback starts in the alternate screen.
- `q` exits and restores the terminal.
- Space pauses and resumes playback.
- Ctrl-C restores the cursor and terminal.
- Playback does not show distracting full-screen flashing.
- Playback does not show obvious top-to-bottom repaint waves at normal terminal size.
- Synchronized presentation does not introduce visible stalling, tearing, or delayed frame bursts.
- Frame-diffed presentation does not leave stale characters or stale colors.
- Frame-diffed presentation does not show cursor-positioning artifacts, bottom-row glitches, or visible patch trails.
- In Ghostty, frame-diffed presentation is visibly less distracting than the current `main` baseline at the same terminal size.
- The stats summary appears after exit.
- The stats summary includes render grid, rendered frames, presented frames, skipped frames, effective FPS, average render time, average present time, and average bytes per frame.

## Notes To Record

Capture these observations when comparing changes:

- Terminal app and version.
- Whether the terminal appears to support synchronized updates.
- Terminal size.
- Clip name.
- Whether repainting is distracting.
- Whether timing feels continuous.
- Stats summary.
- Current `main` baseline commit used for comparison.
- Whether frame-diffed presentation visibly improves Ghostty playback against that baseline.
- Whether full-screen clears or obvious repaint waves remain noticeable.
- Average bytes per frame before and after frame-diffed presentation.

## Regression Guardrails

For synchronized presentation, visual QA is the acceptance gate. Metrics are guardrails:

- Effective FPS should not materially regress against the previous `--stats` baseline for the same clip, terminal app, and terminal size.
- Presented frames should not materially regress against the previous `--stats` baseline for the same clip, terminal app, and terminal size.
- If no prior baseline exists for that clip, terminal app, and terminal size, record the current stats as the comparison point and do not claim a metrics improvement.
- Average bytes per frame may increase slightly because synchronized-update markers add terminal control bytes.

For frame-diffed presentation, Ghostty-visible improvement is required:

- Compare against the current `main` baseline at the same Ghostty version and terminal size.
- Low-motion generated clips should show a material average-bytes-per-frame reduction.
- High-motion generated clips may show a smaller byte reduction because more cells genuinely change.
- Effective FPS and presented frames should not materially regress against the current `main` baseline.
- Real ignored `dist/` videos should be included as manual acceptance references.
- Do not call the stage successful if Ghostty playback still looks like full-screen clear/repaint, even when unit tests pass.
