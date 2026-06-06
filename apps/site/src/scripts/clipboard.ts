/**
 * Event-delegated clipboard for any `[data-copy]` element. Flips the element to
 * a `[data-copied]` state for ~1.7s (CSS swaps the label) and announces it.
 * Vanilla DOM - copy buttons don't need React, so the above-the-fold hero ships
 * zero framework JS.
 */
async function copyText(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text);
  } catch {
    const ta = document.createElement("textarea");
    ta.value = text;
    ta.style.position = "fixed";
    ta.style.opacity = "0";
    document.body.appendChild(ta);
    ta.select();
    try {
      document.execCommand("copy");
    } catch {
      /* no-op */
    }
    ta.remove();
  }
}

const timers = new WeakMap<HTMLElement, ReturnType<typeof setTimeout>>();

document.addEventListener("click", (e) => {
  const btn = (e.target as HTMLElement).closest<HTMLElement>("[data-copy]");
  if (!btn) return;
  const text = btn.getAttribute("data-copy") ?? "";
  void copyText(text).then(() => {
    btn.setAttribute("data-copied", "");
    const live = btn.querySelector<HTMLElement>("[data-live]");
    if (live) live.textContent = "Copied to clipboard";
    const prev = timers.get(btn);
    if (prev) clearTimeout(prev);
    timers.set(
      btn,
      setTimeout(() => {
        btn.removeAttribute("data-copied");
        if (live) live.textContent = "";
      }, 1700),
    );
  });
});

export {};
