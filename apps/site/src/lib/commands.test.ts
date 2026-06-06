import { test, expect } from "bun:test";
import { commands, ALL_RAW, splitArgs, tokenize } from "./commands";

// Accuracy enforcement (spec §6): the tokenizer must never alter a command.
test("token text re-joins to the exact raw command", () => {
  for (const c of Object.values(commands)) {
    expect(c.tokens.map((t) => t.text).join(" ")).toBe(c.raw);
    expect(splitArgs(c.raw).join(" ")).toBe(c.raw);
  }
});

test("only the four real subcommands appear, on real programs", () => {
  const realSubs = new Set(["play", "probe", "export", "doctor", "install"]);
  const realPrograms = new Set(["mojify", "brew"]);
  for (const c of Object.values(commands)) {
    const t = c.tokens;
    expect(realPrograms.has(t[0].text)).toBe(true);
    for (const tok of t) {
      if (tok.kind === "subcommand") expect(realSubs.has(tok.text)).toBe(true);
    }
  }
});

test("no invented export flags slip in", () => {
  const realFlags = new Set([
    "--width",
    "--fps",
    "--bitrate",
    "--at",
    "--duration",
    "--overwrite",
    "--stats",
    "--workers",
    "--recipe",
    "--no-audio",
  ]);
  for (const raw of ALL_RAW) {
    for (const tok of tokenize(raw)) {
      if (tok.kind === "flag") expect(realFlags.has(tok.text)).toBe(true);
    }
  }
});

test("recipe names are limited to the four built-in presets", () => {
  const presets = new Set(["default", "mono", "ascii", "blocks"]);
  for (const raw of ALL_RAW) {
    for (const tok of tokenize(raw)) {
      if (tok.kind === "recipe") expect(presets.has(tok.text)).toBe(true);
    }
  }
});
