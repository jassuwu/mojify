/**
 * Build-time ANSI -> HTML renderer.
 *
 * Mojify's `.ansi` exports and `mojify doctor` output use real truecolor SGR
 * escapes (ESC[38;2;R;G;Bm). We parse them at build time into selectable,
 * copyable DOM - colored <span>s inside a <pre> - so the page's "character frame"
 * proof IS real text, not a screenshot (spec §4.4 / §7). Runs in the Astro
 * frontmatter (Node) during `astro build`; nothing ships to the client.
 */

const ESC = "\x1b";

interface SgrState {
  fg: string | null; // "r,g,b"
  bold: boolean;
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}

/** Apply one SGR parameter list (the numbers between `[` and `m`) to state. */
function applySgr(state: SgrState, params: number[]): void {
  let i = 0;
  while (i < params.length) {
    const p = params[i];
    if (p === 0) {
      state.fg = null;
      state.bold = false;
      i += 1;
    } else if (p === 1) {
      state.bold = true;
      i += 1;
    } else if (p === 22) {
      state.bold = false;
      i += 1;
    } else if (p === 39) {
      state.fg = null;
      i += 1;
    } else if (p === 38 && params[i + 1] === 2) {
      // truecolor: 38;2;r;g;b
      const r = params[i + 2] ?? 0;
      const g = params[i + 3] ?? 0;
      const b = params[i + 4] ?? 0;
      state.fg = `${r},${g},${b}`;
      i += 5;
    } else if (p === 38 && params[i + 1] === 5) {
      // 256-color index - our assets don't use it; skip the index.
      i += 3;
    } else if (p >= 30 && p <= 37) {
      const basic = [
        "0,0,0",
        "205,49,49",
        "13,188,121",
        "229,229,16",
        "36,114,200",
        "188,63,188",
        "17,168,205",
        "229,229,229",
      ];
      state.fg = basic[p - 30];
      i += 1;
    } else if (p >= 90 && p <= 97) {
      const bright = [
        "102,102,102",
        "241,76,76",
        "35,209,139",
        "245,245,67",
        "59,142,234",
        "214,112,214",
        "41,184,219",
        "255,255,255",
      ];
      state.fg = bright[p - 90];
      i += 1;
    } else {
      i += 1;
    }
  }
}

function spanOpen(state: SgrState): string {
  const styles: string[] = [];
  if (state.fg) styles.push(`color:rgb(${state.fg})`);
  if (state.bold) styles.push("font-weight:700");
  if (!styles.length) return "<span>";
  return `<span style="${styles.join(";")}">`;
}

/**
 * Convert a string containing ANSI SGR escapes into an HTML string of
 * <span>-wrapped, HTML-escaped runs. Newlines are preserved literally (the
 * caller wraps the result in a <pre>).
 */
export function ansiToHtml(input: string): string {
  const state: SgrState = { fg: null, bold: false };
  let out = "";
  let open = false;
  let i = 0;

  const closeIfOpen = () => {
    if (open) {
      out += "</span>";
      open = false;
    }
  };

  while (i < input.length) {
    if (input[i] === ESC && input[i + 1] === "[") {
      // parse CSI ... final-byte
      let j = i + 2;
      let nums = "";
      while (j < input.length && /[0-9;]/.test(input[j])) {
        nums += input[j];
        j += 1;
      }
      const final = input[j];
      if (final === "m") {
        const params =
          nums === "" ? [0] : nums.split(";").map((n) => Number(n) || 0);
        closeIfOpen();
        applySgr(state, params);
        i = j + 1;
        continue;
      }
      // non-SGR CSI (cursor moves etc.) - drop it
      i = j + 1;
      continue;
    }

    // literal run until next ESC
    let run = "";
    while (i < input.length && input[i] !== ESC) {
      run += input[i];
      i += 1;
    }
    if (run) {
      closeIfOpen();
      out += spanOpen(state);
      open = true;
      out += escapeHtml(run);
    }
  }
  closeIfOpen();
  return out;
}
