#!/usr/bin/env bash
set -euo pipefail

export_dir="dist/qa/export"
synthetic_source="dist/qa/low-motion-bars.mp4"
synthetic_output="${export_dir}/low-motion-bars-export.mp4"
real_output="${export_dir}/real-sample-export.mp4"

if [[ ! -x ./bin/mojify ]]; then
  printf 'Missing ./bin/mojify. Run `bun run build` first.\n' >&2
  exit 1
fi

if [[ ! -f "${synthetic_source}" ]]; then
  printf 'Missing %s. Run `bun run qa:clips` first.\n' "${synthetic_source}" >&2
  exit 1
fi

mkdir -p "${export_dir}"

printf 'Exporting synthetic QA clip...\n'
./bin/mojify export --overwrite --width 320 \
  dist/qa/low-motion-bars.mp4 \
  dist/qa/export/low-motion-bars-export.mp4

printf '\nVideo stream metadata for %s:\n' "${synthetic_output}"
video_stream="$(
  ffprobe -hide_banner -v error \
    -select_streams v:0 \
    -show_entries stream=codec_name,width,height,avg_frame_rate,duration \
    -of default=noprint_wrappers=1 \
    "${synthetic_output}"
)"

if [[ -z "${video_stream}" ]]; then
  printf 'No video stream found in %s.\n' "${synthetic_output}" >&2
  exit 1
fi

printf '%s\n' "${video_stream}"

video_width="$(
  ffprobe -hide_banner -v error \
    -select_streams v:0 \
    -show_entries stream=width \
    -of csv=p=0 \
    "${synthetic_output}"
)"

if [[ "${video_width}" != "320" ]]; then
  printf 'Expected exported width 320, got %s.\n' "${video_width}" >&2
  exit 1
fi

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
./bin/mojify export --overwrite --width 320 "${real_source}" "${real_output}"

printf '\nAudio stream metadata for %s:\n' "${real_output}"
audio_stream="$(
  ffprobe -hide_banner -v error \
    -select_streams a:0 \
    -show_entries stream=codec_name,sample_rate,channels,duration \
    -of default=noprint_wrappers=1 \
    "${real_output}"
)"

if [[ -z "${audio_stream}" ]]; then
  printf 'Expected an audio stream in %s because %s has source audio.\n' "${real_output}" "${real_source}" >&2
  exit 1
fi

printf '%s\n' "${audio_stream}"

printf '\nExport QA complete.\n'
