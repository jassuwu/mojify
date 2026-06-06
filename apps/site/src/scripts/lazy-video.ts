/**
 * Reduced-motion-aware, in-view lazy video.
 *
 * Markup: <video data-lazy preload="none" muted loop playsinline poster=…>
 *           <source data-src="…mp4" type="video/mp4" />
 *         </video>
 *
 * - prefers-reduced-motion: reduce  → never loads/plays; the poster stays.
 * - otherwise                       → loads + plays only while in view, pauses
 *                                     when scrolled away (saves decode + power).
 *
 * Plain DOM, no framework - the hero/play/recipe clips don't need React.
 */
const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

function activate(video: HTMLVideoElement) {
  if (video.dataset.loaded) return;
  const source = video.querySelector<HTMLSourceElement>("source[data-src]");
  if (source && !source.src) {
    source.src = source.dataset.src ?? "";
    video.load();
  }
  video.dataset.loaded = "1";
}

function init() {
  const videos = Array.from(
    document.querySelectorAll<HTMLVideoElement>("video[data-lazy]"),
  );
  if (reduce || !("IntersectionObserver" in window)) return; // poster only

  const io = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        const video = entry.target as HTMLVideoElement;
        if (entry.isIntersecting) {
          activate(video);
          video.play().catch(() => {
            /* autoplay may be blocked; poster remains */
          });
        } else if (video.dataset.loaded) {
          video.pause();
        }
      }
    },
    { rootMargin: "200px 0px", threshold: 0.1 },
  );

  for (const v of videos) io.observe(v);
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", init);
} else {
  init();
}

export {};
