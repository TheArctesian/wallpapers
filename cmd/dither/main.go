package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"io"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/image/draw"
)

var (
	screenW   = envInt("SCREEN_W", 0)
	screenH   = envInt("SCREEN_H", 0)
	prefix    = envStr("PREFIX", "")
	inputDir  = envStr("INPUT_DIR", "/input")
	outputDir = envStr("OUTPUT_DIR", "/output")
	workers   = envInt("WORKERS", runtime.NumCPU())
)

const (
	scaleFactor = 0.50 // 50% — image fills half the screen, rest is border
	noiseAmount = 0.05 // matches webapp
)

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

var nordPalette = [][3]uint8{
	// Polar Night
	{46, 52, 64},
	{59, 66, 82},
	{67, 76, 94},
	{76, 86, 106},
	// Snow Storm
	{216, 222, 233},
	{229, 233, 240},
	{236, 239, 244},
	// Frost
	{143, 188, 187},
	{136, 192, 208},
	{129, 161, 193},
	{94, 129, 172},
	// Aurora
	{191, 97, 106},
	{208, 135, 112},
	{235, 203, 139},
	{163, 190, 140},
	{180, 142, 173},
}

func clampU8(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func findClosestColor(r, g, b uint8) (uint8, uint8, uint8) {
	minDist := math.MaxFloat64
	var cr, cg, cb uint8
	for _, c := range nordPalette {
		dr := float64(r) - float64(c[0])
		dg := float64(g) - float64(c[1])
		db := float64(b) - float64(c[2])
		dist := dr*dr + dg*dg + db*db
		if dist < minDist {
			minDist = dist
			cr, cg, cb = c[0], c[1], c[2]
		}
	}
	return cr, cg, cb
}

func ditherImage(src image.Image, screenW, screenH int) *image.NRGBA {
	// Calculate image area (50% of screen) and border
	availW := int(math.Floor(float64(screenW) * scaleFactor))
	availH := int(math.Floor(float64(screenH) * scaleFactor))

	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	scaleX := float64(availW) / float64(srcW)
	scaleY := float64(availH) / float64(srcH)
	scale := scaleX
	if scaleY < scale {
		scale = scaleY
	}
	newW := int(math.Floor(float64(srcW) * scale))
	newH := int(math.Floor(float64(srcH) * scale))

	borderX := (screenW - newW) / 2
	borderY := (screenH - newH) / 2

	// Resize with Catmull-Rom
	resized := image.NewNRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(resized, resized.Bounds(), src, srcBounds, draw.Over, nil)

	// Create canvas with nord0 background
	canvas := image.NewNRGBA(image.Rect(0, 0, screenW, screenH))
	bg := color.NRGBA{R: 46, G: 52, B: 64, A: 255}
	for y := 0; y < screenH; y++ {
		for x := 0; x < screenW; x++ {
			canvas.SetNRGBA(x, y, bg)
		}
	}

	// Center the image
	draw.Copy(canvas, image.Pt(borderX, borderY), resized, resized.Bounds(), draw.Over, nil)

	// Floyd-Steinberg dithering — image area only (matches webapp behaviour)
	// Use a uint8 buffer like the webapp's Uint8ClampedArray
	buf := make([][]uint8, newH)
	for y := 0; y < newH; y++ {
		buf[y] = make([]uint8, newW*3)
		for x := 0; x < newW; x++ {
			c := canvas.NRGBAAt(borderX+x, borderY+y)
			buf[y][x*3] = c.R
			buf[y][x*3+1] = c.G
			buf[y][x*3+2] = c.B
		}
	}

	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			i := x * 3

			oldR := clampU8(float64(buf[y][i]) + (rand.Float64()-0.5)*255*noiseAmount)
			oldG := clampU8(float64(buf[y][i+1]) + (rand.Float64()-0.5)*255*noiseAmount)
			oldB := clampU8(float64(buf[y][i+2]) + (rand.Float64()-0.5)*255*noiseAmount)

			newR, newG, newB := findClosestColor(oldR, oldG, oldB)

			errR := float64(oldR) - float64(newR)
			errG := float64(oldG) - float64(newG)
			errB := float64(oldB) - float64(newB)

			buf[y][i] = newR
			buf[y][i+1] = newG
			buf[y][i+2] = newB

			if x+1 < newW {
				ri := (x + 1) * 3
				buf[y][ri] = clampU8(float64(buf[y][ri]) + errR*7/16)
				buf[y][ri+1] = clampU8(float64(buf[y][ri+1]) + errG*7/16)
				buf[y][ri+2] = clampU8(float64(buf[y][ri+2]) + errB*7/16)
			}
			if x > 0 && y+1 < newH {
				bi := (x - 1) * 3
				buf[y+1][bi] = clampU8(float64(buf[y+1][bi]) + errR*3/16)
				buf[y+1][bi+1] = clampU8(float64(buf[y+1][bi+1]) + errG*3/16)
				buf[y+1][bi+2] = clampU8(float64(buf[y+1][bi+2]) + errB*3/16)
			}
			if y+1 < newH {
				bi := x * 3
				buf[y+1][bi] = clampU8(float64(buf[y+1][bi]) + errR*5/16)
				buf[y+1][bi+1] = clampU8(float64(buf[y+1][bi+1]) + errG*5/16)
				buf[y+1][bi+2] = clampU8(float64(buf[y+1][bi+2]) + errB*5/16)
			}
			if x+1 < newW && y+1 < newH {
				bi := (x + 1) * 3
				buf[y+1][bi] = clampU8(float64(buf[y+1][bi]) + errR*1/16)
				buf[y+1][bi+1] = clampU8(float64(buf[y+1][bi+1]) + errG*1/16)
				buf[y+1][bi+2] = clampU8(float64(buf[y+1][bi+2]) + errB*1/16)
			}
		}
	}

	// Write dithered pixels back to canvas (image area only)
	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			i := x * 3
			canvas.SetNRGBA(borderX+x, borderY+y, color.NRGBA{
				R: buf[y][i], G: buf[y][i+1], B: buf[y][i+2], A: 255,
			})
		}
	}

	return canvas
}

func convertHEIC(path string) (string, error) {
	tmpFile, err := os.CreateTemp("", "heic-*.jpg")
	if err != nil {
		return "", err
	}
	tmp := tmpFile.Name()
	tmpFile.Close()

	// heif-convert (libheif, Linux/container) first; sips is the stock macOS
	// decoder and needs no extra packages, so it covers native runs there.
	converters := [][]string{
		{"heif-convert", "-q", "95", path, tmp},
		{"sips", "-s", "format", "jpeg", path, "--out", tmp},
	}

	var errs []string
	for _, argv := range converters {
		if _, err := exec.LookPath(argv[0]); err != nil {
			continue
		}
		out, err := exec.Command(argv[0], argv[1:]...).CombinedOutput()
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v: %s", argv[0], err, out))
			continue
		}
		if info, err := os.Stat(tmp); err == nil && info.Size() > 0 {
			return tmp, nil
		}
		errs = append(errs, fmt.Sprintf("%s: produced no output", argv[0]))
	}

	os.Remove(tmp)
	if len(errs) == 0 {
		return "", fmt.Errorf("no HEIC decoder found (install libheif-tools, or run on macOS for sips)")
	}
	return "", fmt.Errorf("decoding %s: %s", path, strings.Join(errs, "; "))
}

// isHEIF reports whether the first 12 bytes look like an ISO-BMFF HEIF/HEIC
// container ("....ftyp<brand>" with a HEIF-family brand).
func isHEIF(head []byte) bool {
	if len(head) < 12 || !bytes.Equal(head[4:8], []byte("ftyp")) {
		return false
	}
	switch string(head[8:12]) {
	case "heic", "heix", "heim", "heis", "hevc", "hevx", "mif1", "msf1":
		return true
	}
	return false
}

func loadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	head := make([]byte, 12)
	n, _ := io.ReadFull(f, head)
	f.Close()

	if isHEIF(head[:n]) {
		tmp, err := convertHEIC(path)
		if err != nil {
			return nil, err
		}
		defer os.Remove(tmp)
		path = tmp
	}

	g, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer g.Close()

	img, _, err := image.Decode(g)
	return img, err
}

func main() {
	if screenW <= 0 || screenH <= 0 {
		fmt.Fprintln(os.Stderr, "SCREEN_W and SCREEN_H must be set to positive integers")
		os.Exit(2)
	}
	if workers < 1 {
		workers = 1
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output dir: %v\n", err)
		os.Exit(1)
	}

	entries, err := os.ReadDir(inputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input dir: %v\n", err)
		os.Exit(1)
	}

	existing := make(map[string]bool)
	outEntries, _ := os.ReadDir(outputDir)
	for _, e := range outEntries {
		existing[e.Name()] = true
	}

	// Pending work only — anything already rendered at this resolution is left
	// alone, so a re-run costs nothing but picks up newly added sources.
	var todo []string
	skipped := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		switch strings.ToLower(filepath.Ext(e.Name())) {
		case ".jpg", ".jpeg", ".png", ".heic":
		default:
			continue
		}
		if existing[outputName(e.Name())] {
			skipped++
			continue
		}
		todo = append(todo, e.Name())
	}
	sort.Strings(todo)

	fmt.Printf("%dx%d: %d to render, %d already present (%d workers)\n",
		screenW, screenH, len(todo), skipped, workers)
	if len(todo) == 0 {
		return
	}

	var (
		mu     sync.Mutex
		failed int
		done   int
	)
	jobs := make(chan string)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for fname := range jobs {
				err := render(fname)
				mu.Lock()
				done++
				if err != nil {
					failed++
					fmt.Fprintf(os.Stderr, "  [%d/%d] %s: %v\n", done, len(todo), fname, err)
				} else {
					fmt.Printf("  [%d/%d] %s\n", done, len(todo), outputName(fname))
				}
				mu.Unlock()
			}
		}()
	}
	for _, f := range todo {
		jobs <- f
	}
	close(jobs)
	wg.Wait()

	if failed > 0 {
		fmt.Fprintf(os.Stderr, "Done with %d failure(s).\n", failed)
		os.Exit(1)
	}
	fmt.Println("Done.")
}

// outputName maps a source filename to its rendered PNG name. PREFIX is
// optional: the resolution already lives in the output directory name, so the
// default is an unprefixed mirror of the source basename.
func outputName(srcName string) string {
	base := strings.TrimSuffix(srcName, filepath.Ext(srcName))
	if prefix != "" {
		base = prefix + "_" + base
	}
	return base + ".png"
}

// render dithers one source image and writes it atomically, so an interrupted
// run never leaves a half-written PNG that the skip check would treat as done.
func render(fname string) error {
	img, err := loadImage(filepath.Join(inputDir, fname))
	if err != nil {
		return err
	}

	outPath := filepath.Join(outputDir, outputName(fname))
	tmp, err := os.CreateTemp(outputDir, ".tmp-*.png")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if err := png.Encode(tmp, ditherImage(img, screenW, screenH)); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, outPath)
}
