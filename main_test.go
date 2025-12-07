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
		{"Old Threshold Limit (15)", color.RGBA{15, 15, 15, 255}, true},
		{"New Threshold Limit (60)", color.RGBA{60, 60, 60, 255}, true},
		{"Above Check (61)", color.RGBA{61, 61, 61, 255}, false},
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

	t.Run("Black Border with White Content", func(t *testing.T) {
		// 100x100 black image (target: Black)
		img := createImage(100, 100, color.Black)
		// White content rect at (20,20)-(80,80)
		// Even if content is white, it should NOT be cropped because target is Black.
		// Wait, if target is Black, white pixels are "content".
		drawRect(img, 20, 20, 80, 80, color.White)

		bounds := findContentBounds(img)
		// Expect crop to the white box
		expected := image.Rect(20, 20, 80, 80)
		if bounds != expected {
			t.Errorf("Expected %v, got %v", expected, bounds)
		}
	})

	t.Run("White Border with Black Content", func(t *testing.T) {
		// 100x100 white image (target: White)
		img := createImage(100, 100, color.White)
		// Black content rect at (20,20)-(80,80)
		// Target is White, so Black pixels are content.
		drawRect(img, 20, 20, 80, 80, color.Black)

		bounds := findContentBounds(img)
		expected := image.Rect(20, 20, 80, 80)
		if bounds != expected {
			t.Errorf("Expected %v, got %v", expected, bounds)
		}
	})

	t.Run("Mixed Background (Ambiguous)", func(t *testing.T) {
		// If corners are mixed, we expect NO cropping (safe fallback).
		img := createImage(100, 100, color.Gray16{Y: 30000}) // Gray
		// TopLeft: Black
		img.Set(0, 0, color.Black)
		// BottomRight: White
		img.Set(99, 99, color.White)

		bounds := findContentBounds(img)
		expected := image.Rect(0, 0, 100, 100)
		if bounds != expected {
			t.Errorf("Expected full image %v, got %v", expected, bounds)
		}
	})

	t.Run("Black Border protects White Content edge", func(t *testing.T) {
		// Scenario: Black border, but inside there is a White block touching the crop edge.
		// If we didn't lock the mode to Black, the White block might be eaten if we treated White as removable too.
		img := createImage(100, 100, color.Black)
		// Draw White Content at (10, 10) to (90, 90)
		drawRect(img, 10, 10, 90, 90, color.White)

		// Corners are Black (0,0), (99,0) etc. -> Mode = Black.
		// Process should remove black border 0-10.
		// At y=10, row becomes White.
		// Since Mode=Black, White pixels are NOT removable.
		// So cropping should stop exactly at 10.

		bounds := findContentBounds(img)
		expected := image.Rect(10, 10, 90, 90)
		if bounds != expected {
			t.Errorf("Expected %v, got %v", expected, bounds)
		}
	})

	t.Run("Noise Tolerance (Black Border)", func(t *testing.T) {
		// 100x100 Black
		img := createImage(100, 100, color.Black)
		// Content
		drawRect(img, 20, 20, 80, 80, color.White)

		// Add noise to the black border (e.g. at y=5, put some white dots)
		// 95% tolerance means in a 100px row, we can have up to 5 bad pixels.
		for x := 0; x < 4; x++ {
			img.Set(x, 5, color.White)
		}

		bounds := findContentBounds(img)
		expected := image.Rect(20, 20, 80, 80)
		if bounds != expected {
			t.Errorf("Expected %v, got %v", expected, bounds)
		}
	})

	t.Run("Lookahead Gap (Skipping dirty lines)", func(t *testing.T) {
		// 100x100 Black
		img := createImage(100, 100, color.Black)
		// Content starts at 30
		drawRect(img, 30, 30, 70, 70, color.White)

		// Dirty line at y=10 (Full white line)
		// This line is NOT removable (it's 100% white, and mode is Black).
		// But it's followed by 19 lines of pure Black (11 to 29).
		// Logic with lookaheadGap=5 should skip this single dirty line IF lookahead sees removable lines.
		// Wait, lookaheadGap=5 checks only next 5 lines.
		// The lines 11,12,13,14,15 are Black (Removable).
		// So y=10 should be skipped.
		for x := 0; x < 100; x++ {
			img.Set(x, 10, color.White)
		}

		bounds := findContentBounds(img)
		expected := image.Rect(30, 30, 70, 70)
		if bounds != expected {
			t.Errorf("Expected %v, got %v", expected, bounds)
		}
	})
}
