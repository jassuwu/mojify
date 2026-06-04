import React from 'react';

// Brand mint. Bright enough (luma ~209) that Mojify maps it to its densest glyph,
// so the converted character word is solid and legible.
export const WORD_MINT = '#7cffc6';

// Near-black field. Matches Mojify's own output background (#080808) exactly, so the
// clean word and the revealed character image meet with no visible seam at the wipe.
export const FIELD = '#080808';

const WORD_FONT = 'Arial Black, "Arial Bold", Arial, system-ui, sans-serif';

/**
 * The settled `mojify` wordmark, rendered identically in two places:
 *   - the `ReadmeHeaderSource` still that Mojify converts into character art, and
 *   - the clean layer shown above the rising wipe in `ReadmeHeader`.
 * Because both use this exact component (same font, size, position, color), the
 * character reveal registers pixel-for-pixel with the clean word and each letter
 * appears to morph clean -> character in place as the wipe passes.
 *
 * NOTE: keep this purely static. Any animation belongs in HeaderDemo, never here —
 * the source still must equal the clean layer at the moment of the reveal.
 */
export const Wordmark: React.FC<{color?: string}> = ({color = WORD_MINT}) => (
  <div
    style={{
      position: 'absolute',
      inset: 0,
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
    }}
  >
    <div
      style={{
        transform: 'translateY(-22px)',
        fontFamily: WORD_FONT,
        fontWeight: 900,
        fontSize: 236,
        lineHeight: 1,
        letterSpacing: 1.5,
        color,
        whiteSpace: 'nowrap',
      }}
    >
      mojify
    </div>
  </div>
);
