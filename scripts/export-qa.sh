#!/usr/bin/env bash
set -euo pipefail

export_dir="dist/qa/export"
synthetic_source="dist/qa/low-motion-bars.mp4"
synthetic_mp4="${export_dir}/low-motion-bars-export.mp4"
synthetic_gif="${export_dir}/low-motion-bars-export.gif"
synthetic_apng="${export_dir}/low-motion-bars-export.apng"
synthetic_png="${export_dir}/low-motion-bars-frame.png"
synthetic_jpg="${export_dir}/low-motion-bars-frame.jpg"
synthetic_txt="${export_dir}/low-motion-bars-frame.txt"
synthetic_ansi="${export_dir}/low-motion-bars-frame.ansi"

require_nonempty_file() {
  local path="$1"

  if [[ ! -s "${path}" ]]; then
    printf 'Expected non-empty output at %s.\n' "${path}" >&2
    exit 1
  fi
}

check_video_width() {
  local path="$1"
  local expected_width="$2"
  local video_stream video_width

  printf '\nVideo/image stream metadata for %s:\n' "${path}"
  video_stream="$(
    ffprobe -hide_banner -v error \
      -select_streams v:0 \
      -show_entries stream=codec_name,width,height,avg_frame_rate,duration \
      -of default=noprint_wrappers=1 \
      "${path}"
  )"

  if [[ -z "${video_stream}" ]]; then
    printf 'No video/image stream found in %s.\n' "${path}" >&2
    exit 1
  fi

  printf '%s\n' "${video_stream}"

  video_width="$(
    ffprobe -hide_banner -v error \
      -select_streams v:0 \
      -show_entries stream=width \
      -of csv=p=0 \
      "${path}"
  )"

  if [[ "${video_width}" != "${expected_width}" ]]; then
    printf 'Expected exported width %s, got %s for %s.\n' "${expected_width}" "${video_width}" "${path}" >&2
    exit 1
  fi
}

check_audio_stream() {
  local output="$1"
  local source="$2"
  local audio_stream

  printf '\nAudio stream metadata for %s:\n' "${output}"
  audio_stream="$(
    ffprobe -hide_banner -v error \
      -select_streams a:0 \
      -show_entries stream=codec_name,sample_rate,channels,duration \
      -of default=noprint_wrappers=1 \
      "${output}"
  )"

  if [[ -z "${audio_stream}" ]]; then
    printf 'Expected an audio stream in %s because %s has source audio.\n' "${output}" "${source}" >&2
    exit 1
  fi

  printf '%s\n' "${audio_stream}"
}

if [[ ! -x ./bin/mojify ]]; then
  printf 'Missing ./bin/mojify. Run `bun run build` first.\n' >&2
  exit 1
fi

if [[ ! -f "${synthetic_source}" ]]; then
  printf 'Missing %s. Run `bun run qa:clips` first.\n' "${synthetic_source}" >&2
  exit 1
fi

mkdir -p "${export_dir}"

printf 'Exporting synthetic QA clip across representative formats...\n'
./bin/mojify export --overwrite --width 320 "${synthetic_source}" "${synthetic_mp4}"
./bin/mojify export --overwrite --width 320 --at 0s --duration 2s "${synthetic_source}" "${synthetic_gif}"
./bin/mojify export --overwrite --width 320 --at 0s --duration 2s "${synthetic_source}" "${synthetic_apng}"
./bin/mojify export --overwrite --width 320 --at 0s "${synthetic_source}" "${synthetic_png}"
./bin/mojify export --overwrite --width 320 --at 0s "${synthetic_source}" "${synthetic_jpg}"
./bin/mojify export --overwrite --width 80 --at 0s "${synthetic_source}" "${synthetic_txt}"
./bin/mojify export --overwrite --width 80 --at 0s "${synthetic_source}" "${synthetic_ansi}"

check_video_width "${synthetic_mp4}" "320"
check_video_width "${synthetic_gif}" "320"
check_video_width "${synthetic_apng}" "320"
check_video_width "${synthetic_png}" "320"
check_video_width "${synthetic_jpg}" "320"
require_nonempty_file "${synthetic_txt}"
require_nonempty_file "${synthetic_ansi}"

real_source=""
while IFS= read -r -d '' candidate; do
  if ffprobe -hide_banner -v error \
    -select_streams a:0 \
    -show_entries stream=index \
    -of csv=p=0 \
    "${candidate}" 2>/dev/null | grep -q .; then
    real_source="${candidate}"
    break
  fi
done < <(find dist -maxdepth 1 -type f \
  \( -iname '*.mp4' -o -iname '*.m4v' -o -iname '*.mov' -o -iname '*.mkv' -o -iname '*.webm' \) \
  -print0 2>/dev/null || true)

if [[ -z "${real_source}" ]]; then
  printf '\nSkipping optional audio QA: no top-level dist media sample with audio was found.\n'
  printf '\nExport QA complete.\n'
  exit 0
fi

printf '\nExporting optional real sample with source audio: %s\n' "${real_source}"
for ext in mp4 webm mov; do
  real_output="${export_dir}/real-sample-export.${ext}"
  ./bin/mojify export --overwrite --width 320 --duration 2s "${real_source}" "${real_output}"
  check_audio_stream "${real_output}" "${real_source}"
done

printf '\nExport QA complete.\n'
