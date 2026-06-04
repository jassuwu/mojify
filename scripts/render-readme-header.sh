#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
app_dir="${root_dir}/apps/readme-header"
dist_dir="${root_dir}/dist/readme-header"
asset_dir="${root_dir}/docs/assets/readme"
public_dir="${app_dir}/public"

sourceword_png="${dist_dir}/sourceword.png"
charword_small_png="${dist_dir}/charword-480.png"
charword_png="${public_dir}/charword.png"
source_mp4="${dist_dir}/source.mp4"
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

mkdir -p "${dist_dir}" "${asset_dir}" "${public_dir}"

if [[ ! -x "${root_dir}/bin/mojify" ]]; then
  printf 'Missing ./bin/mojify; building first...\n'
  (cd "${root_dir}" && bun run build)
fi

# 1. Render the clean wordmark still (the only thing Mojify converts).
printf 'Rendering clean wordmark still...\n'
(cd "${app_dir}" && bunx remotion still src/index.ts ReadmeHeaderSource "${sourceword_png}" --frame=0 --image-format=png)

# 2. Convert it through the real Mojify pipeline into character art.
#    --width 720 -> a 90x30 character grid (finer than the default 60x20).
printf 'Converting wordmark through Mojify...\n'
"${root_dir}/bin/mojify" export --overwrite --width 720 "${sourceword_png}" "${charword_small_png}"

# 3. Upscale the character image to the composition size (nearest = crisp glyphs).
printf 'Preparing reveal image...\n'
ffmpeg -hide_banner -loglevel error -y \
  -i "${charword_small_png}" \
  -vf "scale=1440:480:flags=neighbor" \
  "${charword_png}"

# 4. Render the full animation. It reveals the real Mojify image under the rising
#    command line, so the GIF is the Remotion output directly (no FFmpeg compositing).
printf 'Rendering README header animation...\n'
(cd "${app_dir}" && bun run render -- "${source_mp4}" --overwrite --codec h264 --pixel-format yuv420p)

printf 'Generating optimized GIF palette...\n'
ffmpeg -hide_banner -loglevel error -y \
  -i "${source_mp4}" \
  -vf "fps=12,palettegen=max_colors=128:stats_mode=diff" \
  "${palette_png}"

printf 'Generating README GIF...\n'
ffmpeg -hide_banner -loglevel error -y \
  -i "${source_mp4}" \
  -i "${palette_png}" \
  -filter_complex "fps=12[x];[x][1:v]paletteuse=dither=bayer:bayer_scale=3" \
  -loop 0 \
  "${header_gif}"

printf 'Generating reduced-motion poster...\n'
ffmpeg -hide_banner -loglevel error -y \
  -sseof -0.2 \
  -i "${source_mp4}" \
  -frames:v 1 \
  "${poster_png}"

gif_bytes="$(wc -c <"${header_gif}")"
soft_limit=$((2 * 1024 * 1024))
hard_limit=$((5 * 1024 * 1024))

printf '\nGenerated README header assets:\n'
print_size "${header_gif}"
print_size "${poster_png}"
print_size "${sourceword_png}"
print_size "${charword_png}"
print_size "${source_mp4}"

if [[ "${gif_bytes}" -gt "${hard_limit}" ]]; then
  printf 'README GIF exceeds 5 MB hard tolerance.\n' >&2
  exit 1
fi

if [[ "${gif_bytes}" -gt "${soft_limit}" ]]; then
  printf 'Warning: README GIF exceeds 2 MB target.\n' >&2
fi
