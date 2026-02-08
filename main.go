package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
)

const (
	screenW     = 2256 // Framework 13 native resolution
	screenH     = 1504
	scaleFactor = 0.50 // 50% — image fills half the screen, rest is border
	noiseAmount = 0.05 // matches webapp
)

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

func ditherImage(src image.Image) *image.NRGBA {
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

func loadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	return img, err
}

func main() {
	inputDir := "/input"
	outputDir := "/output"
	os.MkdirAll(outputDir, 0755)

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

	var files []string
	for _, e := range entries {
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".jpeg" || ext == ".jpg" || ext == ".png" {
			files = append(files, e.Name())
		}
	}
	fmt.Printf("Found %d images to process\n", len(files))

	for _, fname := range files {
		base := strings.TrimSuffix(fname, filepath.Ext(fname))
		outName := "fm13_" + base + ".png"

		if existing[outName] {
			fmt.Printf("Skipping %s (already processed)\n", fname)
			continue
		}

		fmt.Printf("Processing %s...\n", fname)
		img, err := loadImage(filepath.Join(inputDir, fname))
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error loading %s: %v\n", fname, err)
			continue
		}

		result := ditherImage(img)

		outPath := filepath.Join(outputDir, outName)
		f, err := os.Create(outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error creating %s: %v\n", outName, err)
			continue
		}
		if err := png.Encode(f, result); err != nil {
			f.Close()
			fmt.Fprintf(os.Stderr, "  Error encoding %s: %v\n", outName, err)
			continue
		}
		f.Close()
		fmt.Printf("  -> %s\n", outName)
	}

	fmt.Println("Done.")
}
