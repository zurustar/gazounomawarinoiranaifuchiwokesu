package main

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

func TestIsBlack(t *testing.T) {
	tests := []struct {
		name     string
		color    color.Color
		expected bool
	}{
		{"Black", color.RGBA{0, 0, 0, 255}, true},
		{"Near Black", color.RGBA{10, 10, 10, 255}, true},
		{"Threshold Limit", color.RGBA{15, 15, 15, 255}, true},
		{"Above Threshold", color.RGBA{16, 16, 16, 255}, false},
		{"White", color.RGBA{255, 255, 255, 255}, false},
		{"Red", color.RGBA{255, 0, 0, 255}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBlack(tt.color); got != tt.expected {
				t.Errorf("isBlack() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFindContentBounds(t *testing.T) {
	// Helper to create a uniform image
	createImage := func(w, h int, c color.Color) *image.RGBA {
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.Draw(img, img.Bounds(), &image.Uniform{c}, image.Point{}, draw.Src)
		return img
	}

	// Helper to draw a rect
	drawRect := func(img *image.RGBA, x0, y0, x1, y1 int, c color.Color) {
		draw.Draw(img, image.Rect(x0, y0, x1, y1), &image.Uniform{c}, image.Point{}, draw.Src)
	}

	t.Run("Full Black Image", func(t *testing.T) {
		img := createImage(100, 100, color.Black)
		bounds := findContentBounds(img)
		if !bounds.Empty() {
			t.Errorf("Expected empty bounds for full black image, got %v", bounds)
		}
	})

	t.Run("Full White Image", func(t *testing.T) {
		img := createImage(100, 100, color.White)
		bounds := findContentBounds(img)
		expected := image.Rect(0, 0, 100, 100)
		if bounds != expected {
			t.Errorf("Expected %v, got %v", expected, bounds)
		}
	})

	t.Run("Center Content with Black Border", func(t *testing.T) {
		// 100x100 black image
		img := createImage(100, 100, color.Black)
		// 50x50 white rect at (25, 25)
		drawRect(img, 25, 25, 75, 75, color.White)

		bounds := findContentBounds(img)
		expected := image.Rect(25, 25, 75, 75)
		if bounds != expected {
			t.Errorf("Expected %v, got %v", expected, bounds)
		}
	})

	t.Run("Off-center Content", func(t *testing.T) {
		// 100x100 black image
		img := createImage(100, 100, color.Black)
		// Rect at (10, 10) to (20, 20)
		drawRect(img, 10, 10, 20, 20, color.White)

		bounds := findContentBounds(img)
		expected := image.Rect(10, 10, 20, 20)
		if bounds != expected {
			t.Errorf("Expected %v, got %v", expected, bounds)
		}
	})

	t.Run("Noise Handling (Below Threshold)", func(t *testing.T) {
		// 100x100 black image
		img := createImage(100, 100, color.Black)
		// Draw some "noise" pixels that are below threshold (e.g., RGB 5,5,5)
		// These should be treated as black and ignored
		noiseColor := color.RGBA{5, 5, 5, 255}
		drawRect(img, 0, 0, 10, 10, noiseColor)

		// Real content
		drawRect(img, 20, 20, 30, 30, color.White)

		bounds := findContentBounds(img)
		expected := image.Rect(20, 20, 30, 30)
		if bounds != expected {
			t.Errorf("Expected %v, got %v", expected, bounds)
		}
	})
}
