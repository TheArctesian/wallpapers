# wallpapers

Floyd–Steinberg dithering with the [Nord](https://www.nordtheme.com/) palette.
Each source image is scaled to 50% of the target screen and centred on a
`nord0` background.

The point of the layout is that **screens are inferred, not configured**. Attach
a display — any size, any rotation — run `./wallpapers sync`, and it gets
wallpapers cut to its exact pixel geometry. The same repo drives macOS and
Linux/COSMIC.

## Layout

```
raw/                 source images (.jpg, .jpeg, .png, .heic)
screens/<WxH>/       rendered wallpapers, one directory per screen geometry
wallpapers           the CLI — detect, render, apply
lib/
  common.sh          shared helpers
  detect.sh          screen discovery: system_profiler / xrandr / wlr-randr
  apply-macos.sh     wallpaper backend: System Events
  apply-cosmic.sh    wallpaper backend: cosmic-bg
cmd/dither/          the Go renderer (runs in the container)
```

A screen's directory is named after its **post-rotation** pixel geometry, so a
monitor stood on its end lands in `screens/1080x1920` and is served portrait
wallpapers automatically. Two screens of the same size share one directory.

## Use

```sh
./wallpapers screens     # what's attached, and how many wallpapers it has
./wallpapers generate    # render raw/ for every attached screen
./wallpapers set         # apply a random wallpaper per screen
./wallpapers sync        # generate, then set
```

`generate` skips anything already rendered at that geometry, so re-running it
after dropping new images into `raw/` only processes the new ones. Writes are
atomic — an interrupted run leaves no half-written PNG behind.

To render for a screen that is not plugged in right now, name the geometries:

```sh
./wallpapers generate 3440x1440 2256x1504
```

### Add new wallpapers

Drop images into `raw/` and run `./wallpapers sync`.

### Set on login / on demand

`set` takes a non-blocking lock, so wiring it into a shell profile or a
keybinding is safe even if several terminals start at once — the losers exit
quietly rather than racing to rewrite the same config.

## How each platform is handled

|  | macOS | Linux |
| --- | --- | --- |
| Screen discovery | `system_profiler SPDisplaysDataType -json` | `xrandr`, falling back to `wlr-randr` |
| Screen identity | CoreGraphics display ID | output connector name (`eDP-1`, `DP-3`) |
| Applying | System Events `desktop` whose `id` matches | per-output entries for `cosmic-bg` |

Rotation is normalised in one place (`_emit` in `lib/detect.sh`): a screen
reported as rotated 90°/270° is always recorded portrait, whether or not the
platform already applied the rotation to the geometry it reported.

Two caveats worth knowing:

- **macOS** applies a wallpaper to the screen's *current* Space only. Other
  Spaces keep what they had until `set` runs while they are in front.
- **Mirrored** displays share one framebuffer, so only the mirror source is
  listed.

## Requirements

Rendering runs in Docker, which keeps the Go toolchain and `libheif` off the
host. Detection and wallpaper-setting use only what the OS already ships.

- **macOS:** Docker (Desktop, Colima, …) and `jq` — `jq` ships with macOS 15+,
  otherwise `brew install jq`.
- **Linux/COSMIC:** Docker, plus `xrandr` or `wlr-randr`.

## The renderer

`cmd/dither` reads every image in `INPUT_DIR` and writes `SCREEN_W`×`SCREEN_H`
PNGs to `OUTPUT_DIR`, one worker per CPU.

| env | default | |
| --- | --- | --- |
| `SCREEN_W`, `SCREEN_H` | — | required, in pixels |
| `INPUT_DIR` | `/input` | |
| `OUTPUT_DIR` | `/output` | |
| `PREFIX` | none | optional filename prefix |
| `WORKERS` | CPU count | |

`docker-compose.yml` exposes it directly for one-off renders:

```sh
SCREEN_W=3440 SCREEN_H=1440 docker compose run --rm dither
```

It also builds and runs natively (Go 1.23+; `.heic` needs `heif-convert`, or
just works on macOS via the built-in `sips`):

```sh
SCREEN_W=3024 SCREEN_H=1964 INPUT_DIR=raw OUTPUT_DIR=screens/3024x1964 \
  go run ./cmd/dither
```
