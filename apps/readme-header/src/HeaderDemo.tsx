import {
  AbsoluteFill,
  Easing,
  interpolate,
  spring,
  useCurrentFrame,
  useVideoConfig,
} from 'remotion';

const glyphRows = [
  '██▓▒░   mojify   ░▒▓██',
  '▓▒░  turn media into text  ░▒▓',
  '░▒▓██   export header.gif   ██▓▒░',
];

export const HeaderDemo: React.FC = () => {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();

  const intro = spring({
    frame,
    fps,
    config: {
      damping: 18,
      mass: 0.7,
      stiffness: 90,
    },
  });

  const wordScale = interpolate(intro, [0, 1], [0.88, 1], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
  });
  const wordY = interpolate(frame, [0, 16, 28], [18, 0, -82], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
    easing: Easing.out(Easing.cubic),
  });
  const commandOpacity = interpolate(frame, [13, 18, 31, 36], [0, 1, 1, 0], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
  });
  const tunnelScale = interpolate(frame, [18, 32], [0.92, 1.08], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
    easing: Easing.inOut(Easing.cubic),
  });
  const reveal = interpolate(frame, [31, 42], [0, 1], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
    easing: Easing.out(Easing.cubic),
  });
  const scanX = interpolate(frame, [0, 47], [-180, 1140], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
  });

  return (
    <AbsoluteFill
      style={{
        background:
          'radial-gradient(circle at 22% 28%, rgba(124,255,198,0.18), transparent 30%), linear-gradient(135deg, #06080c 0%, #111827 52%, #05070a 100%)',
        color: '#eaf2ff',
        fontFamily:
          'Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
        overflow: 'hidden',
      }}
    >
      <div
        style={{
          position: 'absolute',
          inset: 0,
          backgroundImage:
            'linear-gradient(rgba(255,255,255,0.045) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.035) 1px, transparent 1px)',
          backgroundSize: '40px 40px',
          opacity: 0.45,
        }}
      />

      <div
        style={{
          position: 'absolute',
          left: scanX,
          top: -80,
          width: 130,
          height: 520,
          transform: 'rotate(12deg)',
          background:
            'linear-gradient(90deg, transparent, rgba(124,255,198,0.28), transparent)',
          opacity: 0.8,
        }}
      />

      <div
        style={{
          position: 'absolute',
          inset: 36,
          border: '1px solid rgba(151,166,186,0.26)',
          borderRadius: 18,
          boxShadow: '0 24px 80px rgba(0,0,0,0.42)',
        }}
      />

      <div
        style={{
          position: 'absolute',
          left: 78,
          top: 76,
          transform: `translateY(${wordY}px) scale(${wordScale})`,
          transformOrigin: 'left center',
        }}
      >
        <div
          style={{
            fontSize: 92,
            lineHeight: 1,
            fontWeight: 850,
            letterSpacing: 0,
            color: '#f8fbff',
            textShadow: '0 12px 38px rgba(0,0,0,0.38)',
          }}
        >
          mojify
        </div>
        <div
          style={{
            marginTop: 16,
            fontSize: 24,
            color: '#97a6ba',
          }}
        >
          turn media into text
        </div>
      </div>

      <div
        style={{
          position: 'absolute',
          left: 92,
          right: 92,
          top: 132,
          height: 72,
          opacity: commandOpacity,
          transform: `scale(${tunnelScale})`,
          transformOrigin: 'center',
          borderRadius: 12,
          background: 'rgba(5, 8, 12, 0.92)',
          border: '1px solid rgba(124,255,198,0.32)',
          boxShadow: '0 20px 80px rgba(124,255,198,0.08)',
          display: 'flex',
          alignItems: 'center',
          padding: '0 30px',
          fontFamily:
            'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace',
          fontSize: 24,
          color: '#7cffc6',
          whiteSpace: 'nowrap',
        }}
      >
        <span style={{color: '#97a6ba', marginRight: 14}}>$</span>
        mojify export source.mp4 header.gif
      </div>

      <div
        style={{
          position: 'absolute',
          inset: 58,
          opacity: reveal,
          transform: `translateY(${interpolate(reveal, [0, 1], [34, 0])}px)`,
          borderRadius: 16,
          background: 'rgba(3,5,8,0.94)',
          border: '1px solid rgba(124,255,198,0.32)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontFamily:
            'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace',
          color: '#7cffc6',
          textAlign: 'center',
          boxShadow: '0 24px 90px rgba(0,0,0,0.54)',
        }}
      >
        <div>
          {glyphRows.map((row) => (
            <div
              key={row}
              style={{
                fontSize: 27,
                lineHeight: 1.22,
                letterSpacing: 0,
                textShadow: '0 0 20px rgba(124,255,198,0.24)',
              }}
            >
              {row}
            </div>
          ))}
        </div>
      </div>
    </AbsoluteFill>
  );
};
