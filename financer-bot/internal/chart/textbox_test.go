package chart

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

// The layout checks trust each text's recorded box, so the box must hold every inked pixel.
func TestRecordedTextBoxHoldsItsInk(t *testing.T) {
	for _, tt := range []struct {
		name  string
		label block
	}{
		{"title bold", block{{{bold(titleSize, inkPrimary), "September 2026 · day 30 of 30"}}}},
		{"status with two runs", block{{{bold(statusSize, inkPrimary), "Under by 11,700"}, {regular(labelSize, inkSecondary), " · spent 37,800 of 49,500 so far"}}}},
		{"note over amount", block{{{regular(noteSize, inkSecondary), "target"}}, {{bold(labelSize, inkPrimary), "1,234,567"}}}},
		{"descenders and a minus", block{{{regular(minTextSize, inkPrimary), "jgqy,/−"}}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			pic := newPicture(100)
			box := pic.write(draw.New(pic.canvas), vg.Point{X: 20, Y: 80}, nil, tt.label)
			img := pic.canvas.Image()
			scale := pic.canvas.DPI() / 72
			_, height := pic.canvas.Size()
			const tolerance = 1 // px of anti-aliasing
			for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
				for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
					if color.NRGBAModel.Convert(img.At(x, y)) == surface {
						continue
					}
					pointX := float64(x) / scale
					pointY := float64(height) - float64(y)/scale
					inside := pointX >= float64(box.Min.X)-tolerance/scale && pointX <= float64(box.Max.X)+tolerance/scale &&
						pointY >= float64(box.Min.Y)-tolerance/scale && pointY <= float64(box.Max.Y)+tolerance/scale
					if !assert.True(t, inside, "ink at (%.1f, %.1f) pt outside box %v", pointX, pointY, box) {
						return
					}
				}
			}
		})
	}
}
