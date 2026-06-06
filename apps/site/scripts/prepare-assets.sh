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
