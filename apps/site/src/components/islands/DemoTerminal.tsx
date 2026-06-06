import { useEffect, useRef, useState } from "react";
import { commands, tokenize, type TokenKind } from "../../lib/commands";

interface Props {
  probeHtml: string;
  doctorHtml: string;
  ansiHtml: string;
}

type TabKey = "play" | "probe" | "export" | "doctor";

const RECIPES = [
  { key: "default", src: "/assets/recipes/default.mp4", poster: "/assets/recipes/default-poster.webp", cmd: commands.play },
  { key: "mono", src: "/assets/recipes/mono.mp4", poster: "/assets/recipes/mono-poster.webp", cmd: commands.playMono },
  { key: "ascii", src: "/assets/recipes/ascii.mp4", poster: "/assets/recipes/ascii-poster.webp", cmd: commands.playAscii },
  { key: "blocks", src: "/assets/recipes/blocks.mp4", poster: "/assets/recipes/blocks-poster.webp", cmd: commands.playBlocks },
];

const FORMATS = [
  { key: "video", label: ".mp4", kind: "video", note: "keeps source audio", src: "/assets/recipes/default.mp4", poster: "/assets/recipes/default-poster.webp", cmd: commands.exportVideo },
  { key: "gif", label: ".gif", kind: "video", note: "no audio", src: "/assets/recipes/default.mp4", poster: "/assets/recipes/default-poster.webp", cmd: commands.exportGif },
  { key: "image", label: ".png", kind: "image", note: "one frame", src: "/assets/recipes/default-poster.webp", cmd: commands.exportImage },
  { key: "text", label: ".ansi", kind: "text", note: "real, selectable text", cmd: commands.exportText },
];

const TABS: { key: TabKey; desc: string }[] = [
  { key: "play", desc: "plays a local video or a yt-dlp link in your terminal, with color and audio." },
  { key: "export", desc: "writes the output to a file. the extension sets the format. video keeps audio; gif and apng don't. no webp." },
  { key: "probe", desc: "prints what a file is before you convert it: size, fps, frames, audio, and the text grid." },
  { key: "doctor", desc: "checks for ffmpeg, ffprobe, ffplay, and yt-dlp." },
];

const KIND: Record<TokenKind, string> = {
  program: "text-mint font-bold",
  subcommand: "text-mint",
  flag: "text-offwhite/80",
  recipe: "text-mint",
  string: "text-prompt",
  arg: "text-offwhite/70",
};

function usePrefersReducedMotion() {
  const [reduce, setReduce] = useState(false);
  useEffect(() => {
    const mq = window.matchMedia("(prefers-reduced-motion: reduce)");
    const sync = () => setReduce(mq.matches);
    sync();
    mq.addEventListener("change", sync);
    return () => mq.removeEventListener("change", sync);
  }, []);
  return reduce;
}

function CmdLine({ raw }: { raw: string }) {
  return (
    <code className="t-mono-cmd block min-w-0 overflow-x-auto whitespace-nowrap [scrollbar-width:none]">
      <span className="mr-2 select-none text-prompt">$</span>
      {tokenize(raw).map((t, i) => (
        <span key={i} className={KIND[t.kind]}>
          {i > 0 ? " " : ""}
          {t.text}
        </span>
      ))}
    </code>
  );
}

export default function DemoTerminal({ probeHtml, doctorHtml, ansiHtml }: Props) {
  const [tab, setTab] = useState<TabKey>("play");
  const [recipe, setRecipe] = useState(0);
  const [format, setFormat] = useState(0);
  const reduce = usePrefersReducedMotion();
  const videoRef = useRef<HTMLVideoElement>(null);

  const active =
    tab === "play"
      ? RECIPES[recipe].cmd
      : tab === "export"
        ? FORMATS[format].cmd
        : tab === "probe"
          ? commands.probe
          : commands.doctor;
  const desc = TABS.find((t) => t.key === tab)!.desc;

  useEffect(() => {
    const v = videoRef.current;
    if (v && !reduce) {
      v.load();
      v.play().catch(() => {});
    }
  }, [tab, recipe, format, reduce]);

  function Video({ src, poster, label }: { src: string; poster: string; label: string }) {
    if (reduce)
      return <img src={poster} alt={label} className="h-full w-full object-cover" />;
    return (
      <video ref={videoRef} key={src} poster={poster} muted loop playsInline autoPlay aria-label={label} className="h-full w-full object-cover">
        <source src={src} type="video/mp4" />
      </video>
    );
  }

  function output() {
    if (tab === "play") {
      const r = RECIPES[recipe];
      return (
        <Video src={r.src} poster={r.poster} label={`A clip played with the ${r.key} recipe.`} />
      );
    }
    if (tab === "export") {
      const f = FORMATS[format];
      return (
        <>
          {f.kind === "video" && <Video src={f.src!} poster={f.poster!} label={`Output exported to ${f.label}.`} />}
          {f.kind === "image" && <img src={f.src!} alt={`Output exported to ${f.label}.`} className="h-full w-full object-cover" />}
          {f.kind === "text" && (
            <div className="grid h-full w-full place-items-center overflow-hidden p-3">
              <pre className="t-mono-grid leading-none" aria-hidden="true" dangerouslySetInnerHTML={{ __html: ansiHtml }} />
            </div>
          )}
          <span className="t-mono absolute right-3 top-3 rounded-md bg-field/70 px-2 py-1 text-prompt backdrop-blur-sm">
            {f.label} · {f.note}
          </span>
        </>
      );
    }
    const html = tab === "probe" ? probeHtml : doctorHtml;
    return (
      <div className="h-full w-full overflow-auto p-6 sm:p-8">
        <pre className="t-mono-cmd whitespace-pre-wrap break-words leading-[1.85] text-offwhite/90" dangerouslySetInnerHTML={{ __html: html }} />
      </div>
    );
  }

  const subChips =
    tab === "play"
      ? RECIPES.map((r, i) => ({ label: r.key, on: i === recipe, set: () => setRecipe(i) }))
      : tab === "export"
        ? FORMATS.map((f, i) => ({ label: f.label, on: i === format, set: () => setFormat(i) }))
        : [];

  return (
    <div>
      {/* command menu */}
      <div role="tablist" aria-label="Mojify commands" className="flex flex-wrap gap-2">
        {TABS.map((t) => {
          const on = t.key === tab;
          return (
            <button
              key={t.key}
              type="button"
              role="tab"
              aria-selected={on}
              onClick={() => setTab(t.key)}
              className={
                "t-mono-cmd rounded-lg border px-4 py-2 transition-colors duration-150 " +
                (on ? "border-mint/70 bg-mint/[0.06] text-mint" : "border-hairline text-prompt hover:border-prompt/50 hover:text-offwhite")
              }
            >
              {t.key}
            </button>
          );
        })}
      </div>

      {/* terminal */}
      <div className="term mt-4 overflow-hidden">
        <div className="term__bar">
          <span className="term__dot" />
          <span className="term__dot" />
          <span className="term__dot" />
          <span className="t-mono ml-2 text-prompt">~/mojify ❯ {tab}</span>
          <div className="ml-auto flex items-center">
            <button
              type="button"
              data-copy={active.raw}
              aria-label={`Copy: ${active.raw}`}
              className="copy-btn inline-flex shrink-0 items-center gap-1.5 rounded-md border border-hairline px-2 py-1 text-prompt transition-colors hover:border-mint/50 hover:text-mint"
            >
              <span className="copy-idle">
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                  <rect x="9" y="9" width="11" height="11" rx="2" />
                  <path d="M5 15V5a2 2 0 0 1 2-2h10" />
                </svg>
              </span>
              <span className="copy-done text-mint">
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                  <path d="M20 6 9 17l-5-5" />
                </svg>
              </span>
              <span data-live className="sr-only" aria-live="polite" />
            </button>
          </div>
        </div>
        <div className="p-4 sm:p-5">
          <CmdLine raw={active.raw} />
          <div className="relative mt-4 aspect-video overflow-hidden rounded-lg border border-hairline bg-field">
            {output()}
          </div>
        </div>
      </div>

      {/* description + contextual sub-options */}
      <div className="mt-5 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <p className="t-body max-w-[46ch] text-offwhite/75">{desc}</p>
        {subChips.length > 0 && (
          <div className="flex flex-wrap gap-2">
            {subChips.map((c) => (
              <button
                key={c.label}
                type="button"
                onMouseEnter={c.set}
                onFocus={c.set}
                onClick={c.set}
                aria-pressed={c.on}
                className={
                  "t-mono rounded-md border px-3 py-1.5 transition-colors duration-150 " +
                  (c.on ? "border-mint/70 text-mint" : "border-hairline text-prompt hover:border-prompt/50 hover:text-offwhite")
                }
              >
                {c.label}
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
