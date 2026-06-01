# Use best-effort synchronized presentation

The next playback quality hardening step should target visible repainting before changing fidelity or adding new product surface. Mojify will enable synchronized presentation by default as a best-effort terminal capability: terminals that support the synchronization escape sequences can apply frame updates atomically, while terminals that ignore the sequences continue to display the same frame content through the existing presenter behavior.

This preserves the rendered frame and CLI shape while testing whether terminal-level atomic updates are enough before introducing frame diffing or other byte-reduction strategies.

Success for this stage is visual QA improvement first, with playback metrics as regression guards. A small increase in average bytes per frame is acceptable if synchronized updates reduce distracting repainting without materially reducing effective FPS or presented frames.

No user-facing flag should be added in this stage. If QA finds a terminal where synchronized presentation is harmful, Mojify can add an escape hatch later with evidence from the affected terminal.
