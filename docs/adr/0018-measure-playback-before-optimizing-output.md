# Measure playback before optimizing output

Playback quality hardening starts with repeatable measurement before changing terminal output behavior. Mojify should expose basic playback metrics and use a small sample-clip QA checklist so presenter and renderer changes can be evaluated against a baseline.

This keeps perceived smoothness grounded in evidence instead of relying only on subjective terminal playback impressions.

The first metrics surface is `mojify play --stats <video>`, which preserves normal playback and prints a post-run summary after exit. A machine-readable `--stats-json` surface is likely useful later, but is deferred until the human QA workflow is proven.

The canonical sample clip QA set should be generated synthetic clips covering low motion, high motion, and high-contrast edge cases. Ignored local real clips, such as files under `dist/`, can supplement manual judgment but should not be required for the repo's repeatable baseline.

Generated QA clips should be written to ignored `dist/qa/` paths. The repository should keep the generator/checklist source, not the binary clip outputs.

The first implementation plan for this stage should include only the measuring stick: `mojify play --stats`, internal playback metrics, synthetic QA clip generation into `dist/qa/`, and a manual QA checklist. Terminal output optimizations should come in a follow-up plan after baseline measurements exist.
