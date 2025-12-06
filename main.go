package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

// Threshold for determining "black" color.
// Pixel values (R, G, B) below this threshold are considered black.
const blackThreshold = 15

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
		ext := strings.ToLower(filepath.Ext(filename))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			continue
		}

		// Skip already processed files to avoid infinite loops or double processing
		if strings.HasPrefix(filename, "processed_") {
			continue
		}

		fullPath := filepath.Join(dirPath, filename)
		fmt.Printf("Processing: %s\n", filename)

		if err := processImage(fullPath, dirPath, filename); err != nil {
			fmt.Printf("  Failed to process %s: %v\n", filename, err)
		} else {
			fmt.Printf("  Saved processed_%s\n", filename)
		}
	}
	return nil
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

func isBlack(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	// RGBA returns values in [0, 65535], so we shift right by 8 to get [0, 255]
	r8 := r >> 8
	g8 := g >> 8
	b8 := b >> 8

	return r8 <= blackThreshold && g8 <= blackThreshold && b8 <= blackThreshold
}

func findContentBounds(img image.Image) image.Rectangle {
	bounds := img.Bounds()
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X, bounds.Min.Y

	foundContent := false

	// Scan to find minY (Top)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if !isBlack(img.At(x, y)) {
				minY = y
				foundContent = true
				break
			}
		}
		if foundContent {
			break
		}
	}

	if !foundContent {
		return image.Rectangle{} // Return empty rectangle if full black
	}

	// Scan to find maxY (Bottom)
	foundContent = false
	for y := bounds.Max.Y - 1; y >= minY; y-- {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if !isBlack(img.At(x, y)) {
				maxY = y + 1 // +1 because bounds are exclusive at the max end
				foundContent = true
				break
			}
		}
		if foundContent {
			break
		}
	}

	// Scan to find minX (Left)
	foundContent = false
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := minY; y < maxY; y++ {
			if !isBlack(img.At(x, y)) {
				minX = x
				foundContent = true
				break
			}
		}
		if foundContent {
			break
		}
	}

	// Scan to find maxX (Right)
	foundContent = false
	for x := bounds.Max.X - 1; x >= minX; x-- {
		for y := minY; y < maxY; y++ {
			if !isBlack(img.At(x, y)) {
				maxX = x + 1
				foundContent = true
				break
			}
		}
		if foundContent {
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
