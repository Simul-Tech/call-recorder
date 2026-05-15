package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

func iconIdle() []byte      { return circleIcon(34, 197, 94) }  // green
func iconRecording() []byte { return circleIcon(239, 68, 68) }  // red

func circleIcon(r, g, b uint8) []byte {
	const size = 22
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	cx, cy := float64(size)/2, float64(size)/2
	radius := float64(size)/2 - 1

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - cx + 0.5
			dy := float64(y) - cy + 0.5
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist <= radius {
				alpha := uint8(255)
				if dist > radius-1 {
					alpha = uint8((radius - dist) * 255)
				}
				img.Set(x, y, color.RGBA{r, g, b, alpha})
			}
		}
	}

	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}
