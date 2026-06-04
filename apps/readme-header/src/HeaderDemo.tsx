import {
  AbsoluteFill,
  Easing,
  interpolate,
  spring,
  useCurrentFrame,
  useVideoConfig,
} from 'remotion';

/**
 * Mojify README header — "CLI Tunnel".
 *
 * Pipeline contract (scripts/render-readme-header.sh):
 *   - SOURCE slice = composition frames 0..31 (0-2.65s) shown as CLEAN pixels.
 *   - MOJIFY slice = composition frames 32..47 (2.65-4.0s) pushed through the REAL
 *     Mojify converter and shown as colored character art.
 *
 * How Mojify converts a frame (verified against packages/core/internal/render):
 *   - The 960x320 frame is reduced to a 60x20 character grid (~16px per cell,
 *     point-sampled, no averaging). Coarse — only large thick forms survive.
 *   - Brightness picks a glyph on ramp " .;coPO?#@": near-black -> SPACE (empty),
 *     bright -> dense glyph (@ # O P). Per-cell color is preserved.
 *   - Strong edges (Sobel > 180) become stroke glyphs | / - \ tracing outlines.
 *
 * Design rule that follows: in the conversion window the frame must be ONLY the
 * big ultra-bold mint wordmark on near-black. Any border/grid/panel/box/scan-bar
 * generates edge noise that drowns the letters (the previous baseline did exactly
 * that and was unreadable). So every "terminal" element — eyebrow, command line,
 * scan glint — lives only in the clean source slice and is fully GONE by frame 28
 * (a two-frame margin before the cut). From LOCK_FRAME on, a hard override paints
 * nothing but the word, frozen, at its exact final mint, so frames 32..47 are
 * pixel-identical and the Mojify reveal reads as a stable character "mojify".
 *
 * Reproducibility note: the wordmark assumes Arial Black is installed (true on the
 * macOS render host this asset is built on). On a host without it the stack falls
 * back to a lighter sans, thinning the strokes and weakening the converted glyph
 * density. Keep the render on a machine with Arial Black, or bundle a heavy font.
 */

// Frame from which only the frozen mint wordmark is painted (>= the 2.65s cut - 2).
const LOCK_FRAME = 30;

const BG = '#050505'; // near-black -> converts to empty space

// Wordmark: bright white during the source beat, settling to brand mint by f28.
const INK_WHITE = '#f4f8ff';
const INK_MINT = '#7cffc6'; // luma ~209 -> densest ramp glyph '@'

// Terminal chrome — clean source slice only.
const EYEBROW = '#5f8c7d';
const PROMPT = '#8a99ad';
const CMD_TEXT = '#e8f3ee';

const WORD_FONT = 'Arial Black, "Arial Bold", Arial, system-ui, sans-serif';
const MONO_FONT =
  'ui-monospace, "SF Mono", SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace';

const WORDMARK = 'mojify';

// Command, split so segments can be tinted while it types in character by character.
const CMD_SEGMENTS: ReadonlyArray<{text: string; color: string}> = [
  {text: '$ ', color: PROMPT},
  {text: 'mojify', color: INK_MINT},
  {text: ' export source.mp4 header.gif', color: CMD_TEXT},
];
const CMD_LENGTH = CMD_SEGMENTS.reduce((n, s) => n + s.text.length, 0);

// Blend two hex colors (white -> mint as the scan passes). Returns rgb().
const mix = (a: string, b: string, t: number): string => {
  const h = (c: string) => [
    parseInt(c.slice(1, 3), 16),
    parseInt(c.slice(3, 5), 16),
    parseInt(c.slice(5, 7), 16),
  ];
  const [ar, ag, ab] = h(a);
  const [br, bg, bb] = h(b);
  const c = (x: number, y: number) => Math.round(x + (y - x) * t);
  return `rgb(${c(ar, br)}, ${c(ag, bg)}, ${c(ab, bb)})`;
};

export const HeaderDemo: React.FC = () => {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  const locked = frame >= LOCK_FRAME;

  // --- Wordmark establish (f0..8), recenter for the hero hold (f16..28) ---
  const intro = spring({
    frame,
    fps,
    config: {damping: 200, mass: 0.7, stiffness: 110},
    durationInFrames: 10,
  });
  const wordScale = interpolate(intro, [0, 1], [0.96, 1]);
  const wordOpacity = interpolate(frame, [0, 7], [0, 1], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
  });
  // Sits a touch high during the command beat (room for the spine), eases to the
  // optically-centered hero position (+6 balances the lowercase descenders).
  const wordY = interpolate(frame, [16, 28], [-18, 6], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
    easing: Easing.inOut(Easing.cubic),
  });
  // White -> mint, driven by the scan, fully mint and frozen by f28.
  const tint = interpolate(frame, [16, 26], [0, 1], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
  });

  // --- Eyebrow tagline (source slice only) ---
  const eyebrowOpacity = interpolate(frame, [2, 7, 12, 16], [0, 1, 1, 0], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
  });

  // --- Beat 2: command types in, holds, fully clears by f28 ---
  const cmdOpacity = interpolate(frame, [7, 10, 22, 28], [0, 1, 1, 0], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
  });
  const cmdRise = interpolate(frame, [7, 13], [12, 0], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
    easing: Easing.out(Easing.cubic),
  });
  const typed = Math.round(
    interpolate(frame, [7, 18], [0, CMD_LENGTH], {
      extrapolateLeft: 'clamp',
      extrapolateRight: 'clamp',
    }),
  );
  const typing = frame >= 7 && frame < 18;
  const caretOn = typing ? 1 : Math.floor(frame / 3) % 2 === 0 ? 1 : 0.15;

  // --- Convert gesture: single mint scan glint sweeps the word, gone by f28 ---
  const scanX = interpolate(frame, [15, 27], [-260, 1220], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
    easing: Easing.inOut(Easing.cubic),
  });
  const scanOpacity = interpolate(frame, [15, 18, 24, 28], [0, 1, 1, 0], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
  });

  // --- Loop seam: fade up from the shared black field (flash-free wrap) ---
  const introFade = interpolate(frame, [0, 3], [0, 1], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
  });

  // The hero wordmark. When locked, everything is forced to its exact frozen final
  // state so frames 30..47 are byte-identical and convert to a stable character word.
  const heroColor = locked ? INK_MINT : mix(INK_WHITE, INK_MINT, tint);
  const heroTransform = locked
    ? 'translateY(6px) scale(1)'
    : `translateY(${wordY}px) scale(${wordScale})`;
  const heroOpacity = locked ? 1 : wordOpacity;

  return (
    <AbsoluteFill
      style={{
        backgroundColor: BG,
        overflow: 'hidden',
        opacity: locked ? 1 : introFade,
      }}
    >
      {/* ===== HERO WORDMARK — the only element alive in the conversion window ===== */}
      <AbsoluteFill style={{alignItems: 'center', justifyContent: 'center'}}>
        <div
          style={{
            transform: heroTransform,
            opacity: heroOpacity,
            fontFamily: WORD_FONT,
            fontWeight: 900,
            fontSize: 196,
            lineHeight: 1,
            letterSpacing: 1,
            color: heroColor,
            whiteSpace: 'nowrap',
          }}
        >
          {WORDMARK}
        </div>
      </AbsoluteFill>

      {/* Everything below is conditionally UNMOUNTED before the conversion window. */}
      {!locked && (
        <>
          {/* Scan glint — screen-blended, only shows where it crosses the bright word. */}
          {scanOpacity > 0 && (
            <AbsoluteFill
              style={{
                mixBlendMode: 'screen',
                pointerEvents: 'none',
                opacity: scanOpacity,
              }}
            >
              <div
                style={{
                  position: 'absolute',
                  top: -60,
                  bottom: -60,
                  left: scanX,
                  width: 150,
                  transform: 'skewX(-14deg)',
                  background:
                    'linear-gradient(90deg, transparent, rgba(214,255,240,0.85), transparent)',
                }}
              />
            </AbsoluteFill>
          )}

          {/* Eyebrow tagline, top. */}
          {eyebrowOpacity > 0 && (
            <AbsoluteFill
              style={{
                alignItems: 'center',
                justifyContent: 'flex-start',
                paddingTop: 34,
                opacity: eyebrowOpacity,
              }}
            >
              <div
                style={{
                  fontFamily: MONO_FONT,
                  fontSize: 17,
                  letterSpacing: 7,
                  textTransform: 'uppercase',
                  color: EYEBROW,
                }}
              >
                turn media into text
              </div>
            </AbsoluteFill>
          )}

          {/* CLI command line, bottom — the "tunnel" the source passes through. */}
          {cmdOpacity > 0 && (
            <AbsoluteFill
              style={{
                alignItems: 'center',
                justifyContent: 'flex-end',
                paddingBottom: 30,
                opacity: cmdOpacity,
              }}
            >
              <div
                style={{
                  transform: `translateY(${cmdRise}px)`,
                  fontFamily: MONO_FONT,
                  fontSize: 27,
                  letterSpacing: 0.5,
                  whiteSpace: 'pre',
                }}
              >
                {(() => {
                  let shown = typed;
                  return CMD_SEGMENTS.map((seg, i) => {
                    const take = Math.max(0, Math.min(seg.text.length, shown));
                    shown -= seg.text.length;
                    return (
                      <span key={i} style={{color: seg.color}}>
                        {seg.text.slice(0, take)}
                      </span>
                    );
                  });
                })()}
                <span
                  style={{
                    display: 'inline-block',
                    width: 13,
                    height: 26,
                    marginLeft: 4,
                    transform: 'translateY(4px)',
                    background: INK_MINT,
                    opacity: caretOn,
                  }}
                />
              </div>
            </AbsoluteFill>
          )}
        </>
      )}
    </AbsoluteFill>
  );
};
