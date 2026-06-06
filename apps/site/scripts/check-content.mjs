// Build-output accuracy guardrail. Reads the built dist/index.html and asserts
// the page says what the product does - and never claims what it doesn't.
// Run after `astro build`:  bun run check
//
// The interactive demo renders most commands client-side, so static HTML only
// holds the hero + the demo's initial (play) state. We therefore (1) forward-check
// every command in the static HTML against the single source of truth, and
// (2) validate the ENTIRE command set in commands.ts is clean (covers the
// client-rendered variants too).
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { ALL_RAW } from "../src/lib/commands.ts";

const html = readFileSync(
  fileURLToPath(new URL("../dist/index.html", import.meta.url)),
  "utf8",
);

const decode = (s) =>
  s
    .replace(/&quot;|&#34;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/&gt;/g, ">")
    .replace(/&lt;/g, "<")
    .replace(/&amp;/g, "&");

const text = decode(
  html
    .replace(/<script[\s\S]*?<\/script>/gi, " ")
    .replace(/<style[\s\S]*?<\/style>/gi, " ")
    .replace(/<[^>]+>/g, " ")
    .replace(/\s+/g, " "),
).replace(/\s+([.,;])/g, "$1");

const fails = [];
const must = (s) => {
  if (!text.includes(s)) fails.push(`MISSING required copy: ${JSON.stringify(s)}`);
};

[
  "turn media into text.",
  "brew install jassuwu/tap/mojify",
  "run it.",
  "videos in your terminal",
  "macOS and linux",
].forEach(must);

// keep the copy human: no em/en dashes, no corporate buzzwords (spec: be real).
const dash = text.match(/.{0,18}[—–].{0,18}/);
if (dash) fails.push(`AI-ism (em/en dash) in copy: ${JSON.stringify(dash[0].trim())}`);
const buzz = text.match(
  /\b(leverage|seamless(?:ly)?|effortless(?:ly)?|supercharge|unleash|cutting.edge|game.?chang\w*|empower|delightful|robust|best.in.class|next.gen|elevate your|revolutioniz\w*)\b/i,
);
if (buzz) fails.push(`marketing buzzword in copy: ${JSON.stringify(buzz[0])}`);

// (1) every command in the STATIC html must be a known, canonical command.
const rendered = new Set(
  [
    ...html.matchAll(/data-command="([^"]*)"/g),
    ...html.matchAll(/data-copy="([^"]*)"/g),
  ].map((m) => decode(m[1])),
);
for (const r of rendered) {
  if (!ALL_RAW.includes(r)) fails.push(`ROGUE command in static HTML: ${r}`);
}

// (2) the WHOLE command set (incl. client-rendered demo variants) must be clean.
const badCmd = [/web ?p/i, /emoji/i, /--recipe-file/i, /--ramp/i, /\bnpx\b/i];
for (const raw of ALL_RAW) {
  for (const re of badCmd) {
    if (re.test(raw)) fails.push(`FORBIDDEN in command "${raw}": ${re}`);
  }
}

// (3) forbidden claims in visible text (static).
const scrubbed = text.replace(/No WebP\.?/g, " ");
const forbidden = [
  [/\bweb ?p\b/i, "implies WebP export"],
  [/\bemoji\b/i, "implies emoji output"],
  [/drag|drop your|dropzone/i, "implies an upload/dropzone"],
  [/\bsign ?up\b|\bcreate an account\b/i, "implies accounts"],
  [/upload your|upload a (file|video|clip)/i, "implies uploads"],
  [/\bgo install\b|\bnpm install\b|\bnpx\b/i, "implies an unsupported install path"],
  [/\.webp\b/i, "lists .webp as a format"],
];
for (const [re, why] of forbidden) {
  const m = scrubbed.match(re);
  if (m) fails.push(`FORBIDDEN (${why}): matched ${JSON.stringify(m[0])}`);
}

if (fails.length) {
  console.error("✗ content guardrail failed:\n - " + fails.join("\n - "));
  process.exit(1);
}
console.log(
  `✓ content guardrail passed: ${ALL_RAW.length} commands clean, canonical copy present, no false claims`,
);
