# wallpapers

Floyd-Steinberg dithering with the Nord palette. Each image is scaled to 50% of the target screen and centered on a `nord0` background.

## Layout

```
raw/           # source images (.jpg, .jpeg, .png, .heic)
framework13/   # 2256 x 1504 outputs, prefix fm13_
mbp14/         # 3024 x 1964 outputs, prefix mbp14_
```

Outputs are skipped if a file with the same name already exists in the target directory — safe to re-run.

## Run

Both targets:

```sh
docker compose up --build
```

Just one:

```sh
docker compose up --build fm13    # framework13
docker compose up --build mbp14   # mbp14
```

## Add new wallpapers

1. Drop new images into `raw/`.
2. Re-run the command above. Existing outputs are skipped; only the new files are processed.

## Custom resolution

Override at the CLI without editing `docker-compose.yml`:

```sh
SCREEN_W=2880 SCREEN_H=1800 PREFIX=mbp15 docker compose run --rm fm13
```

`SCREEN_W`, `SCREEN_H`, `PREFIX` are read from the environment by `main.go`.

## Run without Docker

Requires Go 1.23+ and `libheif` (for `.heic` decoding). The binary expects `/input` and `/output` to exist:

```sh
mkdir -p /input /output
# bind-mount or symlink raw/ -> /input and your target dir -> /output
SCREEN_W=2256 SCREEN_H=1504 PREFIX=fm13 go run main.go
```
