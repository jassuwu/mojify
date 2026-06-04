# Mojify README Header Demo Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a reproducible README header demo pipeline that renders a Remotion source animation, converts it through Mojify, composes an optimized GIF/poster pair, and embeds the result in the README.

**Architecture:** Remotion owns only the polished source animation under `apps/readme-header/`. Mojify owns the proof output by exporting that source video into colored character video. FFmpeg owns the final CLI tunnel composition, GIF palette optimization, and poster extraction; final assets live under `docs/assets/readme/`.

**Tech Stack:** Bun workspaces, Remotion scaffold, React/TypeScript, Mojify CLI, FFmpeg/FFprobe, Bash 3-compatible scripting.

---

## Current Context

- Design spec: `docs/superpowers/specs/2026-06-04-mojify-readme-header-demo-design.md`
- Current branch: `feat/readme-header-demo`
- Remotion current docs say new projects can be scaffolded with `npx create-video@latest --yes --blank my-video`, and render commands use `npx remotion render <entry-point> <composition-id> <output-location>`.
- Root workspaces already include `apps/*`, `packages/*`, and `scripts`.
- `dist/` is ignored and should hold intermediates.
- `.superpowers/` is ignored and should not be committed.

## File Map

- Create/modify: `apps/readme-header/`
  - Official Remotion scaffold output.
  - Custom source composition for the `ReadmeHeader` video.
- Modify: `package.json`
  - Add manual root script `readme:header`.
- Create: `scripts/render-readme-header.sh`
  - Manual render pipeline.
- Create: `docs/assets/readme/mojify-header.gif`
  - Final committed README animation.
- Create: `docs/assets/readme/mojify-header-poster.png`
  - Reduced-motion poster fallback.
- Modify: `README.md`
  - Embed header using `<picture>`.
- Optional modify: `CONTEXT.md`
  - Add term for README header demo if implementation adds durable project language.

## Parallelization Plan

Run these as parallel workers after the plan is committed:

- **Worker A: Remotion app** owns `apps/readme-header/**`.
- **Worker B: Render pipeline** owns `scripts/render-readme-header.sh` and root `package.json`.
- **Worker C: README/docs integration** owns `README.md`, `docs/assets/readme/.gitkeep` if needed, and optional `CONTEXT.md`.

Final asset generation depends on Worker A and B, so it should be done by the controller after integration.

---

### Task 1: Scaffold the Remotion App

**Files:**
- Create: `apps/readme-header/**`

- [ ] **Step 1: Run the official scaffold**

Run from repo root:

```bash
mkdir -p apps
npx create-video@latest --yes --blank apps/readme-header
```

Expected:
- `apps/readme-header/` exists.
- The generated app contains a Remotion entry point under `src/`.
- The generated app has a `package.json`.

If the scaffold asks for package-manager choices despite `--yes`, choose the minimal blank React/TypeScript Remotion template and do not enable extra templates or cloud/lambda features.

- [ ] **Step 2: Remove nested VCS or package-manager noise if generated**

Run:

```bash
find apps/readme-header -maxdepth 2 -name .git -type d -print
find apps/readme-header -maxdepth 1 \( -name package-lock.json -o -name yarn.lock -o -name pnpm-lock.yaml \) -print
```

Expected:
- If a nested `.git` directory exists, remove only `apps/readme-header/.git`.
- If non-Bun lockfiles exist, remove only the nested app lockfile. Keep root `bun.lock`.

- [ ] **Step 3: Normalize the app package metadata**

Modify `apps/readme-header/package.json`:

- Set `"name"` to `"@mojify/readme-header"`.
- Set `"private"` to `true`.
- Keep scaffold-generated Remotion dependency versions.
- Ensure scripts include:

```json
{
  "dev": "remotion studio src/index.ts",
  "render": "remotion render src/index.ts ReadmeHeader",
  "typecheck": "tsc --noEmit"
}
```

Do not add a `"build"` script to this package. Root `bun run build` must not render or typecheck the README header app.

- [ ] **Step 4: Install workspace dependencies**

Run from repo root:

```bash
bun install
```

Expected:
- Root `bun.lock` updates with Remotion app dependencies.
- No app-local lockfile remains.

- [ ] **Step 5: Commit scaffold**

Run:

```bash
git add apps/readme-header package.json bun.lock
git commit --no-gpg-sign -m "feat: scaffold readme header app"
```

---

### Task 2: Implement the CLI Tunnel Source Animation

**Files:**
- Modify/Create: `apps/readme-header/src/index.ts`
- Modify/Create: `apps/readme-header/src/Root.tsx`
- Create: `apps/readme-header/src/HeaderDemo.tsx`

- [ ] **Step 1: Register the Remotion root**

Ensure `apps/readme-header/src/index.ts` contains:

```ts
import {registerRoot} from 'remotion';
import {RemotionRoot} from './Root';

registerRoot(RemotionRoot);
```

- [ ] **Step 2: Define the `ReadmeHeader` composition**

Ensure `apps/readme-header/src/Root.tsx` contains:

```tsx
import {Composition} from 'remotion';
import {HeaderDemo} from './HeaderDemo';

export const RemotionRoot: React.FC = () => {
  return (
    <Composition
      id="ReadmeHeader"
      component={HeaderDemo}
      durationInFrames={48}
      fps={12}
      width={960}
      height={320}
    />
  );
};
```

- [ ] **Step 3: Add the source animation component**

Create `apps/readme-header/src/HeaderDemo.tsx` with:

```tsx
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
```

- [ ] **Step 4: Typecheck the app**

Run:

```bash
cd apps/readme-header
bun run typecheck
```

Expected: TypeScript exits 0.

- [ ] **Step 5: Render a source smoke MP4**

Run from repo root:

```bash
mkdir -p dist/readme-header
cd apps/readme-header
bun run render -- ../../dist/readme-header/source.mp4 --overwrite --codec h264 --pixel-format yuv420p
```

Expected:
- `dist/readme-header/source.mp4` exists.
- The render is `960x320`, 12fps, and roughly 4 seconds.

- [ ] **Step 6: Commit source animation**

Run:

```bash
git add apps/readme-header
git commit --no-gpg-sign -m "feat: add readme header source animation"
```

---

### Task 3: Add the Manual Header Render Pipeline

**Files:**
- Create: `scripts/render-readme-header.sh`
- Modify: `package.json`

- [ ] **Step 1: Add the render script**

Create `scripts/render-readme-header.sh`:

```bash
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
```

- [ ] **Step 2: Make the script executable**

Run:

```bash
chmod +x scripts/render-readme-header.sh
```

- [ ] **Step 3: Add the root package script**

Modify root `package.json` scripts so it includes:

```json
"readme:header": "bash scripts/render-readme-header.sh"
```

Keep existing scripts unchanged.

- [ ] **Step 4: Validate script syntax**

Run:

```bash
bash -n scripts/render-readme-header.sh
git diff --check
```

Expected: both commands exit 0.

- [ ] **Step 5: Commit render pipeline**

Run:

```bash
git add package.json scripts/render-readme-header.sh
git commit --no-gpg-sign -m "feat: add readme header render pipeline"
```

---

### Task 4: Embed README Header and Document the Asset

**Files:**
- Modify: `README.md`
- Optional Modify: `CONTEXT.md`

- [ ] **Step 1: Add README embed**

Add this block after the badges paragraph and before `## Installation` in `README.md`:

```html
<p align="center">
  <picture>
    <source media="(prefers-reduced-motion: reduce)" srcset="docs/assets/readme/mojify-header-poster.png">
    <img alt="Mojify transforms a polished mojify source animation into colored text video output." src="docs/assets/readme/mojify-header.gif" width="960">
  </picture>
</p>
```

- [ ] **Step 2: Remove the completed roadmap item**

In `README.md`, remove this Roadmap bullet after assets are committed:

```markdown
- a Mojify-generated README header demo GIF
```

Do not remove unrelated roadmap bullets.

- [ ] **Step 3: Add project language to context**

If `CONTEXT.md` does not already define a term for README asset generation, add:

```markdown
**README header demo**:
The checked-in animated proof asset near the top of the README. It is generated from an original Remotion source animation, transformed through Mojify output, and composed into a GitHub-friendly GIF with a reduced-motion poster.
_Avoid_: Copyrighted sample footage, generic logo loop, required CI render
```

Place it near other distribution/project-shape terms.

- [ ] **Step 4: Validate Markdown references**

Run:

```bash
rg -n "mojify-header|README header demo|a Mojify-generated README header demo GIF" README.md CONTEXT.md
```

Expected:
- README has the `<picture>` block.
- README no longer has the completed roadmap bullet.
- CONTEXT has the durable term if added.

- [ ] **Step 5: Commit README/docs integration**

Run:

```bash
git add README.md CONTEXT.md
git commit --no-gpg-sign -m "docs: embed readme header demo"
```

---

### Task 5: Generate and Tune the Final Assets

**Files:**
- Create: `docs/assets/readme/mojify-header.gif`
- Create: `docs/assets/readme/mojify-header-poster.png`
- Tuning target: `apps/readme-header/src/HeaderDemo.tsx`
- Tuning target: `scripts/render-readme-header.sh`

- [ ] **Step 1: Run the full render pipeline**

Run:

```bash
bun run readme:header
```

Expected:
- `docs/assets/readme/mojify-header.gif` exists.
- `docs/assets/readme/mojify-header-poster.png` exists.
- Script exits 0.
- GIF is no larger than 5 MB.

- [ ] **Step 2: Inspect media metadata**

Run:

```bash
ffprobe -hide_banner -v error \
  -select_streams v:0 \
  -show_entries stream=codec_name,width,height,avg_frame_rate,duration \
  -of default=noprint_wrappers=1 \
  docs/assets/readme/mojify-header.gif

ffprobe -hide_banner -v error \
  -select_streams v:0 \
  -show_entries stream=codec_name,width,height \
  -of default=noprint_wrappers=1 \
  docs/assets/readme/mojify-header-poster.png
```

Expected:
- GIF width is `960`.
- Poster width is `960`.
- GIF duration is approximately 4 seconds.

- [ ] **Step 3: Inspect generated poster**

Use the local image viewer on:

```text
docs/assets/readme/mojify-header-poster.png
```

Expected:
- Poster shows a readable final Mojify output reveal.
- The image is not blank, too dark, or unreadable.

- [ ] **Step 4: Tune if readability or size fails**

If the GIF is unreadable:
- Increase `--width 480` to `--width 560` in `scripts/render-readme-header.sh`.
- Keep final GIF width at `960`.
- Re-run `bun run readme:header`.

If the GIF exceeds 5 MB:
- Reduce `max_colors=96` to `max_colors=80`.
- Keep FPS at `12`; reduce to `10` only if palette reduction is insufficient.
- Re-run `bun run readme:header`.

If the CLI command is too small:
- Increase command font size in `apps/readme-header/src/HeaderDemo.tsx` from `24` to `28`.
- Re-run `bun run readme:header`.

- [ ] **Step 5: Commit final assets**

Run:

```bash
git add apps/readme-header/src/HeaderDemo.tsx scripts/render-readme-header.sh docs/assets/readme/mojify-header.gif docs/assets/readme/mojify-header-poster.png
git commit --no-gpg-sign -m "feat: generate readme header demo assets"
```

If neither the app nor script changed during tuning, omit those paths from `git add`.

---

### Task 6: Final Verification

**Files:**
- Verify all files changed in this stage.

- [ ] **Step 1: Run static checks**

Run:

```bash
bash -n scripts/render-readme-header.sh
git diff --check
bun run fmt:check
```

Expected: all commands exit 0.

- [ ] **Step 2: Run project tests and build**

Run:

```bash
GOCACHE=/private/tmp/mojify-gocache go test -count=1 ./...
bun run test
bun run build
```

Expected: all commands exit 0.

- [ ] **Step 3: Run app typecheck**

Run:

```bash
cd apps/readme-header
bun run typecheck
```

Expected: exits 0.

- [ ] **Step 4: Re-run README header pipeline**

Run:

```bash
bun run readme:header
```

Expected:
- exits 0.
- final GIF remains under 5 MB.
- final assets are updated deterministically enough that repeated runs do not cause unexpected large visual changes.

- [ ] **Step 5: Check final git status**

Run:

```bash
git status --short --branch
git log --oneline --decorate -8
```

Expected:
- Branch is `feat/readme-header-demo`.
- Worktree is clean after final commit.
