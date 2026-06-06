/**
 * Single source of truth for every CLI string rendered on the site.
 *
 * Why this exists (spec §6 "Enforcement"): command text is brand-critical and
 * accuracy-critical. Every command shown on the page is defined here ONCE, then
 * tokenized programmatically for coloring. A test (commands.test.ts) asserts that
 * re-joining a command's tokens reproduces its `raw` string exactly - so a
 * tokenizer/markup typo can never silently ship a wrong command, and the set of
 * commands on the page can never drift from this list.
 *
 * Only real commands and real flags appear here (verified against
 * packages/core/internal/cli). Do not add invented flags or unsupported usage.
 */

export type TokenKind =
  | "program" // `mojify`, `brew` - the executable
  | "subcommand" // `play`, `probe`, `export`, `doctor`, `install`
  | "flag" // `--recipe`, `--width`, …
  | "recipe" // a recipe preset name following `--recipe`
  | "string" // a quoted argument (e.g. a URL)
  | "arg"; // a path / value / tap reference

export interface CmdToken {
  text: string;
  kind: TokenKind;
}

export interface Command {
  /** stable id for referencing a command from a component */
  id: string;
  /** the canonical command line, verbatim */
  raw: string;
  /** programmatically derived display tokens */
  tokens: CmdToken[];
}

const PROGRAMS = new Set(["mojify", "brew"]);
const SUBCOMMANDS = new Set([
  "play",
  "probe",
  "export",
  "doctor",
  "install",
]);
const RECIPES = new Set(["default", "mono", "ascii", "blocks"]);

/** Split a command line into argv-style pieces, keeping double-quoted spans intact. */
export function splitArgs(raw: string): string[] {
  const out: string[] = [];
  let cur = "";
  let inQuote = false;
  for (const ch of raw) {
    if (ch === '"') {
      inQuote = !inQuote;
      cur += ch;
    } else if (ch === " " && !inQuote) {
      if (cur) out.push(cur);
      cur = "";
    } else {
      cur += ch;
    }
  }
  if (cur) out.push(cur);
  return out;
}

/** Classify argv pieces into colored tokens. */
export function tokenize(raw: string): CmdToken[] {
  const pieces = splitArgs(raw);
  const tokens: CmdToken[] = [];
  let prevFlag: string | null = null;
  pieces.forEach((text, i) => {
    let kind: TokenKind;
    if (i === 0 && PROGRAMS.has(text)) {
      kind = "program";
    } else if (text.startsWith("--")) {
      kind = "flag";
    } else if (text.startsWith('"') || text.endsWith('"')) {
      kind = "string";
    } else if (prevFlag === "--recipe" && RECIPES.has(text)) {
      kind = "recipe";
    } else if (i <= 1 && SUBCOMMANDS.has(text)) {
      kind = "subcommand";
    } else {
      kind = "arg";
    }
    prevFlag = text.startsWith("--") ? text : null;
    tokens.push({ text, kind });
  });
  return tokens;
}

function cmd(id: string, raw: string): Command {
  return { id, raw, tokens: tokenize(raw) };
}

export const commands = {
  install: cmd("install", "brew install jassuwu/tap/mojify"),
  doctor: cmd("doctor", "mojify doctor"),
  probe: cmd("probe", "mojify probe poster.png"),

  // play, by recipe
  play: cmd("play", "mojify play intro.mp4"),
  playMono: cmd("playMono", "mojify play --recipe mono intro.mp4"),
  playAscii: cmd("playAscii", "mojify play --recipe ascii intro.mp4"),
  playBlocks: cmd("playBlocks", "mojify play --recipe blocks intro.mp4"),

  // export, by output family (extension routes the format)
  exportVideo: cmd("exportVideo", "mojify export intro.mp4 clip.mp4"),
  exportGif: cmd("exportGif", "mojify export intro.mp4 clip.gif"),
  exportImage: cmd("exportImage", "mojify export poster.png frame.png"),
  exportText: cmd("exportText", "mojify export poster.png frame.ansi"),
} satisfies Record<string, Command>;

export type CommandId = keyof typeof commands;

/** Every raw command string on the page - used by the accuracy test. */
export const ALL_RAW = Object.values(commands).map((c) => c.raw);
