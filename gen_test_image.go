//go:build ignore

package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
)

func main() {
	// Create a 200x200 image
	width, height := 200, 200
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with black
	black := color.RGBA{0, 0, 0, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{black}, image.Point{}, draw.Src)

	// Draw a red rectangle in the center (50,50) to (150,150)
	// This leaves a 50px black border around it
	red := color.RGBA{255, 0, 0, 255}
	rect := image.Rect(50, 50, 150, 150)
	draw.Draw(img, rect, &image.Uniform{red}, image.Point{}, draw.Src)

	// Save to test_image.png
	f, err := os.Create("test_image.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}
