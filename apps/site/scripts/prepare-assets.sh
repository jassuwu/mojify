#!/usr/bin/env bash
# Rebuild the site's web assets from Mojify's own output.
#
# Two sets, both committed (their sources are gitignored and not reproducible in CI):
#   public/assets/recipes/*  - the four recipe clips shown in the demo terminal
#   public/assets/bg/*       - the ambient character-art background
#
# Idempotent. Needs: the built ./bin/mojify, ffmpeg, and ImageMagick (magick).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
OUT="$ROOT/apps/site/public/assets"
DATA="$ROOT/apps/site/src/data"
DIST="$ROOT/dist"
MOJIFY="$ROOT/bin/mojify"

mkdir -p "$OUT/recipes" "$OUT/bg"

# h264 mp4 for autoplay-muted-inline looping <video>
mp4() { # in out maxw
  ffmpeg -y -loglevel error -i "$1" \
    -vf "scale='min($3,iw)':-2:flags=lanczos,format=yuv420p" \
    -an -c:v libx264 -crf 25 -preset slow -movflags +faststart "$2"
}
# first-frame webp poster (ffmpeg here lacks libwebp; use ImageMagick)
poster() { magick "$1[0]" -resize "$3>" -quality 80 "$2"; }

echo "→ recipes (one clip, four looks)"
recipe_src() { # name -> source gif
  case "$1" in
    default) echo "$DIST/redeyes.gif" ;;
    mono) echo "$DIST/redeyes-mono.gif" ;;
    ascii) echo "$DIST/redeyes-ascii.gif" ;;
    blocks) echo "$DIST/redeyes-blocks.gif" ;;
  esac
}
for name in default mono ascii blocks; do
  src="$(recipe_src "$name")"
  if [[ -f "$src" ]]; then
    mp4 "$src" "$OUT/recipes/$name.mp4" 900
    poster "$src" "$OUT/recipes/$name-poster.webp" 900
  else
    echo "  ! missing $src (skipped $name)"
  fi
done

echo "→ export demo (one bright, full-frame still through every output format)"
# The export tab needs a source that fills the frame so all four formats read well,
# the .ansi/.txt one especially (the redeyes recipe art is mostly black, which
# re-converts to near-empty text). A single colorful still drives all four:
#   .mp4/.gif  a gentle zoom of the still, run through Mojify as char-art
#   .png       the still's char-art, one frame
#   .ansi      the same frame as truecolor, selectable text
ES="$DIST/spirited.png"
if [[ -f "$ES" && -x "$MOJIFY" ]]; then
  mkdir -p "$OUT/export"
  # gentle Ken Burns zoom (upscaled first so it stays crisp) -> char-art mp4 -> web mp4
  magick "$ES" -resize 2560x -filter Lanczos "$DIST/spirited-big.png"
  ffmpeg -y -loglevel error -loop 1 -i "$DIST/spirited-big.png" \
    -vf "zoompan=z='min(zoom+0.0006,1.18)':d=144:x='iw/2-(iw/zoom/2)':y='ih/2-(ih/zoom/2)':s=1280x720:fps=24,format=yuv420p" \
    -t 6 -c:v libx264 -crf 18 "$DIST/spirited-zoom.mp4"
  "$MOJIFY" export --overwrite --recipe default --width 900 "$DIST/spirited-zoom.mp4" "$DIST/spirited-charart.mp4"
  mp4 "$DIST/spirited-charart.mp4" "$OUT/export/spirited.mp4" 900
  # poster = the char-art video's first frame (zoom 1.0), so it matches the .ansi framing
  ffmpeg -y -loglevel error -i "$OUT/export/spirited.mp4" -frames:v 1 "$DIST/spirited-poster.png"
  poster "$DIST/spirited-poster.png" "$OUT/export/spirited-poster.webp" 900
  # .ansi: native-ish 130 cols, then round truecolor values to a step of 40 so the
  # build-time ANSI->HTML collapses neighbouring cells into shared spans (the raw
  # output sets a color before every glyph, which would be one span per cell).
  "$MOJIFY" export --overwrite --recipe default --width 130 "$ES" "$DIST/spirited.ansi"
  python3 - "$DIST/spirited.ansi" "$DATA/export-frame.ansi" 40 <<'PY'
import re, sys
src, dst, step = sys.argv[1], sys.argv[2], int(sys.argv[3])
data = open(src, encoding="utf-8", errors="replace").read()
def q(m):
    f = lambda v: min(255, (int(v) + step // 2) // step * step)
    return f"\x1b[38;2;{f(m[1])};{f(m[2])};{f(m[3])}m"
open(dst, "w").write(re.sub(r"\x1b\[38;2;(\d+);(\d+);(\d+)m", q, data))
PY
else
  echo "  ! missing $ES or $MOJIFY (skipped export demo)"
fi

echo "→ ambient background (an abstract clip, run through Mojify as mono char-art)"
SRC="$DIST/transitions1.mp4"
CHAR="$DIST/ambient-charart.mp4"
if [[ -f "$SRC" && -x "$MOJIFY" ]]; then
  # high pixel width keeps the characters crisp (not blurry pixels)
  "$MOJIFY" export --overwrite --recipe mono --width 1600 --fps 20 "$SRC" "$CHAR"
  # no blur; trim a ~11s loop; tuned to ~2 MB while staying legible
  ffmpeg -y -loglevel error -ss 0 -t 11 -r 14 -i "$CHAR" \
    -vf "scale=1100:-2:flags=bicubic,format=yuv420p" \
    -an -c:v libx264 -crf 35 -preset veryslow -movflags +faststart "$OUT/bg/ambient.mp4"
  poster "$CHAR" "$OUT/bg/ambient-poster.webp" 1100
else
  echo "  ! missing $SRC or $MOJIFY (skipped background)"
fi

echo "✓ assets prepared:"
find "$OUT" -type f -printf '%10s  %p\n' 2>/dev/null | sort -k2 || ls -laR "$OUT"
