package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Thresholds for "Black-ish" and "White-ish" pixels.
// We increased the black threshold to 60 to catch dark gray shadows/borders.
const (
	blackThreshold = 60
	whiteThreshold = 195
)

// isBlack checks if a color is considered "black".
// Kept for testing purposes and potential single-pixel checks.
func isBlack(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	r8 := r >> 8
	g8 := g >> 8
	b8 := b >> 8
	return r8 <= blackThreshold && g8 <= blackThreshold && b8 <= blackThreshold
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <directory_path>")
		return
	}

	dirPath := os.Args[1]
	fmt.Printf("Processing images in: %s\n", dirPath)

	err := processDirectory(dirPath)
	if err != nil {
		fmt.Printf("Error processing directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Processing complete.")
}

func processDirectory(dirPath string) error {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()

		// Skip hidden files
		if strings.HasPrefix(filename, ".") {
			continue
		}

		// Skip already processed files to avoid infinite loops or double processing
		if strings.HasPrefix(filename, "processed_") {
			continue
		}

		fullPath := filepath.Join(dirPath, filename)

		// Check if file is a supported image based on content (MIME type)
		if !isSupportedImage(fullPath) {
			continue
		}

		fmt.Printf("Processing: %s\n", filename)

		if err := processImage(fullPath, dirPath, filename); err != nil {
			fmt.Printf("  Failed to process %s: %v\n", filename, err)
		} else {
			fmt.Printf("  Saved processed_%s\n", filename)
		}
	}
	return nil
}

func isSupportedImage(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	// Only read the first 512 bytes to determine the content type
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return false
	}

	contentType := http.DetectContentType(buffer)
	return contentType == "image/jpeg" || contentType == "image/png"
}

func processImage(filePath, dirPath, filename string) error {
	img, format, err := loadImage(filePath)
	if err != nil {
		return err
	}

	bounds := findContentBounds(img)
	if bounds.Empty() {
		return fmt.Errorf("image is completely black or empty")
	}

	// If the bounds match the original image, no cropping is needed, but we save it anyway as per requirement
	// Or we could skip. For now, let's proceed with cropping (which will just be a copy) and saving.

	croppedImg := cropImage(img, bounds)

	outFilename := "processed_" + filename
	// Append extension if missing (e.g. for extensionless screenshots)
	if filepath.Ext(outFilename) == "" {
		if format == "jpeg" {
			outFilename += ".jpg"
		} else if format == "png" {
			outFilename += ".png"
		}
	}
	outPath := filepath.Join(dirPath, outFilename)

	return saveImage(outPath, croppedImg, format)
}

func loadImage(path string) (image.Image, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	img, format, err := image.Decode(file)
	if err != nil {
		return nil, "", err
	}
	return img, format, nil
}

// isPixelRemovable determines if a pixel is considered "background" (very dark or very light).
// However, for a row to be removed, it usually must be uniform.
// We'll handle uniformity in the scanning logic.

func findContentBounds(img image.Image) image.Rectangle {
	bounds := img.Bounds()
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X, bounds.Min.Y

	// Helpers to check color type
	isPixelBlack := func(r8, g8, b8 uint32) bool {
		return r8 <= blackThreshold && g8 <= blackThreshold && b8 <= blackThreshold
	}
	isPixelWhite := func(r8, g8, b8 uint32) bool {
		return r8 >= whiteThreshold && g8 >= whiteThreshold && b8 >= whiteThreshold
	}

	// 1. Determine the target background color (Black or White) based on corners.
	// We check the 4 corners of the image.
	corners := []struct{ x, y int }{
		{bounds.Min.X, bounds.Min.Y},
		{bounds.Max.X - 1, bounds.Min.Y},
		{bounds.Min.X, bounds.Max.Y - 1},
		{bounds.Max.X - 1, bounds.Max.Y - 1},
	}

	blackCornerCount := 0
	whiteCornerCount := 0

	for _, p := range corners {
		c := img.At(p.x, p.y)
		r, g, b, _ := c.RGBA()
		r8, g8, b8 := r>>8, g>>8, b>>8

		if isPixelBlack(r8, g8, b8) {
			blackCornerCount++
		} else if isPixelWhite(r8, g8, b8) {
			whiteCornerCount++
		}
	}

	type TargetMode int
	const (
		ModeNone TargetMode = iota
		ModeBlack
		ModeWhite
	)

	var mode TargetMode
	if blackCornerCount > whiteCornerCount {
		mode = ModeBlack
	} else if whiteCornerCount > blackCornerCount {
		mode = ModeWhite
	} else {
		// Tie or neither.
		// If we found some black corners but no white, use black (and vice versa).
		if blackCornerCount > 0 {
			mode = ModeBlack
		} else if whiteCornerCount > 0 {
			mode = ModeWhite
		} else {
			// If corners are colors (neither black nor white), check edges?
			// For now, if corners aren't background, we assume no cropping needed.
			mode = ModeNone
		}
	}

	if mode == ModeNone {
		// No detectable background color at corners, return original bounds
		return bounds
	}

	// Helpers to check row/col uniformity
	// A row is removable if it is MOSTLY (>95%) the Target Color.
	const noiseTolerance = 0.95
	const lookaheadGap = 5 // Ensure we skip over thin noise lines if real background continues

	isRowRemovable := func(y int) bool {
		width := bounds.Dx()
		matchCount := 0

		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)
			r, g, b, _ := c.RGBA()
			r8, g8, b8 := r>>8, g>>8, b>>8

			if mode == ModeBlack && isPixelBlack(r8, g8, b8) {
				matchCount++
			} else if mode == ModeWhite && isPixelWhite(r8, g8, b8) {
				matchCount++
			}
		}

		total := float64(width)
		return float64(matchCount)/total >= noiseTolerance
	}

	isColRemovable := func(x int) bool {
		height := bounds.Dy()
		matchCount := 0

		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			c := img.At(x, y)
			r, g, b, _ := c.RGBA()
			r8, g8, b8 := r>>8, g>>8, b>>8

			if mode == ModeBlack && isPixelBlack(r8, g8, b8) {
				matchCount++
			} else if mode == ModeWhite && isPixelWhite(r8, g8, b8) {
				matchCount++
			}
		}

		total := float64(height)
		return float64(matchCount)/total >= noiseTolerance
	}

	// Scan MinY (Top)
	minY = bounds.Min.Y
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		if isRowRemovable(y) {
			minY = y + 1
			continue
		}
		// Lookahead
		allNextRemovable := true
		if y+lookaheadGap >= bounds.Max.Y {
			allNextRemovable = false
		} else {
			for k := 1; k <= lookaheadGap; k++ {
				if !isRowRemovable(y + k) {
					allNextRemovable = false
					break
				}
			}
		}
		if allNextRemovable {
			minY = y + 1
		} else {
			break
		}
	}

	// If whole image is removable (minY reached MaxY), return empty
	if minY >= bounds.Max.Y {
		return image.Rectangle{}
	}

	// Scan MaxY (Bottom)
	maxY = bounds.Max.Y
	for y := bounds.Max.Y - 1; y >= minY; y-- {
		if isRowRemovable(y) {
			maxY = y
			continue
		}
		// Lookahead (Upwards)
		allPriorRemovable := true
		if y-lookaheadGap < minY {
			allPriorRemovable = false
		} else {
			for k := 1; k <= lookaheadGap; k++ {
				if !isRowRemovable(y - k) {
					allPriorRemovable = false
					break
				}
			}
		}
		if allPriorRemovable {
			maxY = y
		} else {
			break
		}
	}

	// Scan MinX (Left)
	minX = bounds.Min.X
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		if isColRemovable(x) {
			minX = x + 1
			continue
		}
		// Lookahead
		allNextRemovable := true
		if x+lookaheadGap >= bounds.Max.X {
			allNextRemovable = false
		} else {
			for k := 1; k <= lookaheadGap; k++ {
				if !isColRemovable(x + k) {
					allNextRemovable = false
					break
				}
			}
		}
		if allNextRemovable {
			minX = x + 1
		} else {
			break
		}
	}

	// Scan MaxX (Right)
	maxX = bounds.Max.X
	for x := bounds.Max.X - 1; x >= minX; x-- {
		if isColRemovable(x) {
			maxX = x
			continue
		}
		// Lookahead (Leftwards)
		allPriorRemovable := true
		if x-lookaheadGap < minX {
			allPriorRemovable = false
		} else {
			for k := 1; k <= lookaheadGap; k++ {
				if !isColRemovable(x - k) {
					allPriorRemovable = false
					break
				}
			}
		}
		if allPriorRemovable {
			maxX = x
		} else {
			break
		}
	}

	return image.Rect(minX, minY, maxX, maxY)
}

func cropImage(img image.Image, rect image.Rectangle) image.Image {
	// For sub-image support (if the image implementation supports it)
	if subImg, ok := img.(interface {
		SubImage(r image.Rectangle) image.Image
	}); ok {
		return subImg.SubImage(rect)
	}

	// Fallback: create a new image and draw
	dst := image.NewRGBA(rect.Sub(rect.Min))
	draw.Draw(dst, dst.Bounds(), img, rect.Min, draw.Src)
	return dst
}

func saveImage(path string, img image.Image, format string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	switch format {
	case "jpeg":
		return jpeg.Encode(file, img, nil)
	case "png":
		return png.Encode(file, img)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}
