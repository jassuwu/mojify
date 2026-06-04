#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
app_dir="${root_dir}/apps/readme-header"
dist_dir="${root_dir}/dist/readme-header"
asset_dir="${root_dir}/docs/assets/readme"

source_mp4="${dist_dir}/source.mp4"
mojify_mp4="${dist_dir}/mojify-output.mp4"
composite_mp4="${dist_dir}/mojify-header-composite.mp4"
palette_png="${dist_dir}/mojify-header-palette.png"
header_gif="${asset_dir}/mojify-header.gif"
poster_png="${asset_dir}/mojify-header-poster.png"

require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    printf 'Missing required command: %s\n' "${name}" >&2
    exit 1
  fi
}

print_size() {
  local path="$1"
  if [[ -f "${path}" ]]; then
    wc -c <"${path}" | awk -v p="${path}" '{printf "%s: %.2f MB\n", p, $1 / 1024 / 1024}'
  fi
}

require_command bun
require_command ffmpeg
require_command ffprobe

mkdir -p "${dist_dir}" "${asset_dir}"

if [[ ! -x "${root_dir}/bin/mojify" ]]; then
  printf 'Missing ./bin/mojify; building first...\n'
  (cd "${root_dir}" && bun run build)
fi

printf 'Rendering Remotion source animation...\n'
(cd "${app_dir}" && bun run render -- "${source_mp4}" --overwrite --codec h264 --pixel-format yuv420p)

printf 'Rendering Mojify proof output...\n'
"${root_dir}/bin/mojify" export \
  --overwrite \
  --width 480 \
  --fps 12 \
  --duration 4s \
  "${source_mp4}" \
  "${mojify_mp4}"

printf 'Composing README header MP4...\n'
ffmpeg -hide_banner -loglevel error -y \
  -i "${source_mp4}" \
  -i "${mojify_mp4}" \
  -filter_complex "[0:v]trim=start=0:end=2.65,setpts=PTS-STARTPTS,scale=960:320:force_original_aspect_ratio=decrease,pad=960:320:(ow-iw)/2:(oh-ih)/2,setsar=1[source];[1:v]trim=start=2.65:end=4,setpts=PTS-STARTPTS,scale=960:320:force_original_aspect_ratio=decrease,pad=960:320:(ow-iw)/2:(oh-ih)/2,setsar=1[mojify];[source][mojify]concat=n=2:v=1:a=0,format=yuv420p[out]" \
  -map "[out]" \
  -movflags +faststart \
  "${composite_mp4}"

printf 'Generating optimized GIF palette...\n'
ffmpeg -hide_banner -loglevel error -y \
  -i "${composite_mp4}" \
  -vf "fps=12,scale=960:-1:flags=lanczos,palettegen=max_colors=96:stats_mode=diff" \
  "${palette_png}"

printf 'Generating README GIF...\n'
ffmpeg -hide_banner -loglevel error -y \
  -i "${composite_mp4}" \
  -i "${palette_png}" \
  -filter_complex "fps=12,scale=960:-1:flags=lanczos[x];[x][1:v]paletteuse=dither=bayer:bayer_scale=3" \
  -loop 0 \
  "${header_gif}"

printf 'Generating reduced-motion poster...\n'
ffmpeg -hide_banner -loglevel error -y \
  -sseof -0.2 \
  -i "${composite_mp4}" \
  -frames:v 1 \
  "${poster_png}"

gif_bytes="$(wc -c <"${header_gif}")"
soft_limit=$((2 * 1024 * 1024))
hard_limit=$((5 * 1024 * 1024))

printf '\nGenerated README header assets:\n'
print_size "${header_gif}"
print_size "${poster_png}"
print_size "${source_mp4}"
print_size "${mojify_mp4}"
print_size "${composite_mp4}"

if [[ "${gif_bytes}" -gt "${hard_limit}" ]]; then
  printf 'README GIF exceeds 5 MB hard tolerance.\n' >&2
  exit 1
fi

if [[ "${gif_bytes}" -gt "${soft_limit}" ]]; then
  printf 'Warning: README GIF exceeds 2 MB target.\n' >&2
fi
