import {
  AbsoluteFill,
  Easing,
  Img,
  interpolate,
  spring,
  staticFile,
  useCurrentFrame,
  useVideoConfig,
} from 'remotion';
import {FIELD, Wordmark, WORD_MINT} from './Wordmark';

/**
 * Mojify README header — "CLI Tunnel" (command-line wipe).
 *
 * Story: a clean `mojify` wordmark holds on near-black; a terminal command types in
 * at the bottom; then THAT command line lifts off and travels straight up to the top,
 * and in its wake the wordmark resolves into the REAL Mojify character output (the
 * pre-rendered `charword.png`, which is the converted clean wordmark, pixel-registered
 * so each letter morphs clean -> character in place). The command line parks at the
 * top as a thin header above the held character word.
 *
 * The whole composition is encoded straight to the GIF, so the only thing Mojify
 * converts is the isolated `ReadmeHeaderSource` still — there is no per-frame
 * re-conversion of this scene, hence the command line / scan edge are free styling.
 */

const PROMPT = '#8a99ad';
const CMD_TEXT = '#e8f3ee';
const MONO_FONT =
  'ui-monospace, "SF Mono", SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace';

// The command line, split so segments keep their colors while it types in.
const CMD_SEGMENTS: ReadonlyArray<{text: string; color: string}> = [
  {text: '$ ', color: PROMPT},
  {text: 'mojify', color: WORD_MINT},
  {text: ' export source.mp4 header.gif', color: CMD_TEXT},
];
const CMD_LENGTH = CMD_SEGMENTS.reduce((n, s) => n + s.text.length, 0);

// Wipe-edge travel: rests low while the command types, rises to a parked top header.
// (Coordinates are in the 1440x480 composition space.)
const LINE_REST = 440;
const LINE_PARK = 48;

// Beat boundaries (48 frames @ 12fps).
const TYPE_START = 6;
const TYPE_END = 19;
const SWEEP_START = 24;
const SWEEP_END = 33;

export const HeaderDemo: React.FC = () => {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();

  // Word establishes once, then holds dead-still (it must match charword.png exactly).
  const intro = spring({
    frame,
    fps,
    config: {damping: 200, mass: 0.7, stiffness: 110},
    durationInFrames: 10,
  });
  const wordScale = interpolate(intro, [0, 1], [0.96, 1]);
  const wordOpacity = interpolate(frame, [0, 6], [0, 1], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
  });
  const introFade = interpolate(frame, [0, 3], [0, 1], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
  });

  // Command types in, then becomes the rising wipe edge.
  const typed = Math.round(
    interpolate(frame, [TYPE_START, TYPE_END], [0, CMD_LENGTH], {
      extrapolateLeft: 'clamp',
      extrapolateRight: 'clamp',
    }),
  );
  const typing = frame >= TYPE_START && frame < TYPE_END;
  const sweeping = frame >= SWEEP_START;
  const caretOn = sweeping
    ? 0
    : typing
      ? 1
      : Math.floor(frame / 3) % 2 === 0
        ? 1
        : 0.2;

  // The wipe edge (and the command line riding it) rises bottom -> top, then parks.
  const lineY = interpolate(frame, [SWEEP_START, SWEEP_END], [LINE_REST, LINE_PARK], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
    easing: Easing.inOut(Easing.cubic),
  });

  return (
    <AbsoluteFill style={{backgroundColor: FIELD, overflow: 'hidden', opacity: introFade}}>
      {/* Clean wordmark — the layer shown ABOVE the wipe edge. */}
      <div
        style={{
          position: 'absolute',
          inset: 0,
          transform: `scale(${wordScale})`,
          opacity: wordOpacity,
        }}
      >
        <Wordmark />
      </div>

      {/* Transformed output — real Mojify characters, revealed BELOW the wipe edge.
          The image is held full-frame and clipped to the band under lineY, so it
          registers exactly with the clean word above. */}
      <div
        style={{
          position: 'absolute',
          left: 0,
          top: lineY,
          width: 1440,
          height: 480 - lineY,
          overflow: 'hidden',
        }}
      >
        <Img
          src={staticFile('charword.png')}
          style={{position: 'absolute', left: 0, top: -lineY, width: 1440, height: 480}}
        />
      </div>

      {/* The command line — types at the bottom, then is the rising scan edge. */}
      <div
        style={{
          position: 'absolute',
          left: 0,
          right: 0,
          top: lineY - 54,
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'flex-end',
        }}
      >
        <div
          style={{
            fontFamily: MONO_FONT,
            fontSize: 37,
            letterSpacing: 0.5,
            whiteSpace: 'pre',
            textShadow: `0 0 24px ${FIELD}, 0 0 9px ${FIELD}`,
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
              width: 18,
              height: 36,
              marginLeft: 6,
              transform: 'translateY(6px)',
              background: WORD_MINT,
              opacity: caretOn,
            }}
          />
        </div>
      </div>

      {/* The hard wipe edge: a thin mint scan line the command rides on. */}
      <div
        style={{
          position: 'absolute',
          left: 0,
          right: 0,
          top: lineY,
          height: 3,
          background: `linear-gradient(90deg, transparent, ${WORD_MINT} 22%, ${WORD_MINT} 78%, transparent)`,
          opacity: 0.55,
        }}
      />
    </AbsoluteFill>
  );
};
