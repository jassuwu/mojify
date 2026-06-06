/**
 * Render captured CLI text (probe / doctor) to lightly-colored HTML.
 * `mojify doctor` emits no color when piped, so we colorize status tokens
 * ourselves - the text stays verbatim. Shared by the Astro page and the
 * interactive demo island.
 */
function esc(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

export function doctorHtml(content: string): string {
  return content
    .replace(/\n+$/, "")
    .split("\n")
    .map((line) => {
      if (line === "mojify doctor")
        return `<span style="color:var(--color-mint);font-weight:700">${esc(line)}</span>`;
      const m = line.match(/^(ok|warn|error)(\s+)(\S+)(\s+)(.*)$/);
      if (m) {
        const status =
          m[1] === "ok"
            ? "var(--color-mint)"
            : m[1] === "warn"
              ? "#e6c34a"
              : "#f2766b";
        return (
          `<span style="color:${status};font-weight:700">${esc(m[1])}</span>${esc(m[2])}` +
          `<span style="color:var(--color-offwhite)">${esc(m[3])}</span>${esc(m[4])}` +
          `<span style="color:var(--color-prompt)">${esc(m[5])}</span>`
        );
      }
      return `<span style="color:var(--color-offwhite)">${esc(line)}</span>`;
    })
    .join("\n");
}

export function probeHtml(content: string): string {
  return content
    .replace(/\n+$/, "")
    .split("\n")
    .map((line) => {
      const m = line.match(/^([\w-]+)(:\s*)(.*)$/);
      if (m) {
        return (
          `<span style="color:var(--color-prompt)">${esc(m[1])}${esc(m[2])}</span>` +
          `<span style="color:var(--color-offwhite)">${esc(m[3])}</span>`
        );
      }
      return esc(line);
    })
    .join("\n");
}
