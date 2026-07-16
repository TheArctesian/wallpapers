#!/usr/bin/env bash
# Generate dithered Nord wallpapers sized for each connected monitor.
#
# Reads monitor geometry from xrandr, then runs the Go ditherer (in Docker)
# once per unique resolution, writing PNGs into screens/<WxH>/.
# Re-runnable: existing outputs are skipped by main.go.
set -euo pipefail
cd "$(dirname "$(readlink -f "$0")")"

IMAGE=wallpapers-dither
echo "==> Building $IMAGE ..."
docker build -t "$IMAGE" . >/dev/null

# Unique WxH logical sizes of connected outputs (xrandr reports post-rotation size).
sizes=$(xrandr --query | awk '/ connected/{
  for (i=1;i<=NF;i++) if ($i ~ /^[0-9]+x[0-9]+\+[0-9]+\+[0-9]+$/) { split($i,a,"+"); print a[1]; break }
}' | sort -u)

if [ -z "$sizes" ]; then echo "No connected monitors found via xrandr." >&2; exit 1; fi

for wh in $sizes; do
  w=${wh%x*}; h=${wh#*x}
  out="screens/$wh"
  mkdir -p "$out"
  echo "==> Generating ${wh} into ${out}/ ..."
  docker run --rm \
    -e SCREEN_W="$w" -e SCREEN_H="$h" -e PREFIX="wp" \
    -v "$PWD/raw:/input:ro" \
    -v "$PWD/$out:/output" \
    "$IMAGE"
done

echo "==> Done. Per-monitor wallpapers:"
for wh in $sizes; do printf '  %-12s %s files\n' "$wh" "$(ls "screens/$wh" 2>/dev/null | wc -l)"; done
