#!/usr/bin/env bash
set -euo pipefail

mkdir -p dist/qa

ffmpeg -hide_banner -loglevel error -y \
  -f lavfi -i "smptebars=size=320x180:rate=24:duration=5" \
  -c:v mpeg4 -q:v 3 -pix_fmt yuv420p \
  dist/qa/low-motion-bars.mp4

ffmpeg -hide_banner -loglevel error -y \
  -f lavfi -i "testsrc2=size=320x180:rate=60:duration=5" \
  -c:v mpeg4 -q:v 3 -pix_fmt yuv420p \
  dist/qa/high-motion-testsrc.mp4

ffmpeg -hide_banner -loglevel error -y \
  -f lavfi -i "color=c=black:size=320x180:rate=24:duration=5,drawgrid=width=32:height=18:thickness=2:color=white" \
  -c:v mpeg4 -q:v 3 -pix_fmt yuv420p \
  dist/qa/high-contrast-grid.mp4

ffmpeg -hide_banner -loglevel error -y \
  -f lavfi -i "testsrc2=size=320x180:rate=1:duration=1" \
  -frames:v 1 \
  dist/qa/still-source.png

printf 'Generated QA clips:\n'
printf '  dist/qa/low-motion-bars.mp4\n'
printf '  dist/qa/high-motion-testsrc.mp4\n'
printf '  dist/qa/high-contrast-grid.mp4\n'
printf '  dist/qa/still-source.png\n'
