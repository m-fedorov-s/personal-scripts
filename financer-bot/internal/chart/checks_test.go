package chart

import (
	"fmt"
	"image/color"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

const (
	maxLongSide = 1280 // px
	minContrast = 4.5
)

// problems lists every way pic breaks the layout rules: a text that comes within textGap of
// another text or a mark, a text outside the image or not textGap inside the area that holds it,
// text ink below minContrast on what it sits on, a label left out, and an image too large.
func problems(pic *picture) []string {
	var found []string
	width, height := pic.canvas.Size()
	image := rect(0, 0, width, height)
	pixels := pic.canvas.Image().Bounds()
	if long := max(pixels.Dx(), pixels.Dy()); long > maxLongSide {
		found = append(found, fmt.Sprintf("image is %d px on its long side", long))
	}
	for _, placed := range pic.texts {
		if !contains(image, placed.box) {
			found = append(found, fmt.Sprintf("%q is outside the image", placed.text))
		}
	}
	// A text written over whatever lay at its spot is held to the image bounds alone, and no other
	// text counts as touching it.
	strict := *pic
	strict.texts = slices.DeleteFunc(slices.Clone(pic.texts), func(placed placedText) bool { return placed.overlapping })
	for index, placed := range strict.texts {
		if what, blocked := strict.obstacle(placed.box, &strict.texts[index], placed.in, placed.beneath); blocked {
			found = append(found, fmt.Sprintf("%q touches %s", placed.text, what))
		}
		background := surface
		if placed.in != nil {
			background = composite(placed.in.fill, surface)
			if !contains(boundsOf(placed.in.outline), grow(placed.box, textGap)) {
				found = append(found, fmt.Sprintf("%q is not %v pt inside its area", placed.text, textGap))
			}
		}
		for _, ink := range placed.inks {
			shown := composite(ink, background)
			if ratio := contrast(shown, background); ratio < minContrast {
				found = append(found, fmt.Sprintf("%q has contrast %.2f:1", placed.text, ratio))
			}
			for _, wash := range placed.beneath {
				if ratio := contrast(composite(wash.fill, shown), composite(wash.fill, background)); ratio < minContrast {
					found = append(found, fmt.Sprintf("%q has contrast %.2f:1 beneath a wash", placed.text, ratio))
				}
			}
		}
	}
	for _, label := range pic.missed {
		found = append(found, fmt.Sprintf("%q was left out", label))
	}
	return found
}

func boundsOf(points []vg.Point) vg.Rectangle {
	box := vg.Rectangle{Min: points[0], Max: points[0]}
	for _, point := range points[1:] {
		box = union(box, vg.Rectangle{Min: point, Max: point})
	}
	return box
}

// plantedBase is a small clean picture: one bar, one line and one label, well apart.
func plantedBase(t *testing.T) (*picture, draw.Canvas) {
	pic := newPicture(200)
	region := draw.New(pic.canvas)
	pic.fillRect(region, withinInk, rect(100, 20, 120, 120))
	pic.stroke(region, draw.LineStyle{Color: inkSecondary, Width: vg.Points(1.25)}, vg.Point{X: 20, Y: 150}, vg.Point{X: 360, Y: 150})
	pic.write(region, vg.Point{X: 200, Y: 100}, nil, block{{{bold(labelSize, inkPrimary), "existing"}}})
	require.Empty(t, problems(pic), "the base picture must be clean")
	return pic, region
}

func TestProblemsFindsEachPlantedDefect(t *testing.T) {
	planted := block{{{bold(labelSize, inkPrimary), "planted"}}}
	tests := []struct {
		name  string
		plant func(pic *picture, region draw.Canvas)
		want  string
	}{
		{"label moved onto a bar", func(pic *picture, region draw.Canvas) { pic.write(region, vg.Point{X: 105, Y: 80}, nil, planted) }, "touches an area"},
		{"label 1.5 pt right of a bar", func(pic *picture, region draw.Canvas) { pic.write(region, vg.Point{X: 121.5, Y: 80}, nil, planted) }, "touches an area"},
		{"label on a line", func(pic *picture, region draw.Canvas) { pic.write(region, vg.Point{X: 40, Y: 155}, nil, planted) }, "touches a line"},
		{"label on a label", func(pic *picture, region draw.Canvas) { pic.write(region, vg.Point{X: 210, Y: 95}, nil, planted) }, `touches text "existing"`},
		{"label past the right edge", func(pic *picture, region draw.Canvas) { pic.write(region, vg.Point{X: 360, Y: 60}, nil, planted) }, "outside the image"},
		{"white on mid blue", func(pic *picture, region draw.Canvas) {
			cell := pic.fillRect(region, rgb(0x5598e7), rect(250, 20, 330, 60))
			pic.write(region, vg.Point{X: 254, Y: 54}, cell, block{{{bold(labelSize, white), "planted"}}})
		}, "contrast"},
		{"label in an area without its margin", func(pic *picture, region draw.Canvas) {
			cell := pic.fillRect(region, rgb(0xcde2fb), rect(250, 20, 330, 60))
			pic.write(region, vg.Point{X: 251, Y: 54}, cell, planted)
		}, "inside its area"},
		{"translucent ink on the surface", func(pic *picture, region draw.Canvas) {
			pic.write(region, vg.Point{X: 270, Y: 80}, nil, block{{{bold(labelSize, withAlpha(inkPrimary, 0.15)), "planted"}}})
		}, "contrast"},
		{"white on a translucent grey wash", func(pic *picture, region draw.Canvas) {
			cell := pic.fillRect(region, withAlpha(inkSecondary, 0.2), rect(250, 20, 330, 60))
			pic.write(region, vg.Point{X: 254, Y: 54}, cell, block{{{bold(labelSize, white), "planted"}}})
		}, "contrast"},
		{"grey dimmed by a wash painted over it", func(pic *picture, region draw.Canvas) {
			pic.write(region, vg.Point{X: 254, Y: 54}, nil, block{{{bold(labelSize, rgb(0x6e6e6e)), "planted"}}})
			wash := pic.fillRect(region, gapWash(1), rect(250, 20, 330, 60))
			pic.texts[len(pic.texts)-1].beneath = []*area{wash}
		}, "beneath a wash"},
		{"label inside a wash it does not declare", func(pic *picture, region draw.Canvas) {
			pic.fillRect(region, rgb(0xcde2fb), rect(230, 20, 350, 70))
			pic.write(region, vg.Point{X: 250, Y: 60}, nil, planted)
		}, "touches an area"},
		{"label 1.5 pt from the ink of a thick line", func(pic *picture, region draw.Canvas) {
			pic.stroke(region, draw.LineStyle{Color: inkSecondary, Width: vg.Points(4)}, vg.Point{X: 200, Y: 30}, vg.Point{X: 360, Y: 30})
			_, height := planted.size()
			pic.write(region, vg.Point{X: 220, Y: 33.5 + height}, nil, planted)
		}, "touches a line"},
		{"label left out", func(pic *picture, region draw.Canvas) { pic.missed = append(pic.missed, "planted") }, `"planted" was left out`},
		{"image taller than the long-side limit", func(pic *picture, region draw.Canvas) {
			pic.canvas = vgimg.NewWith(vgimg.UseWH(canvasWidth, 600), vgimg.UseDPI(216), vgimg.UseBackgroundColor(surface))
		}, "image is 1800 px on its long side"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pic, region := plantedBase(t)
			tt.plant(pic, region)
			assert.Contains(t, strings.Join(problems(pic), "\n"), tt.want)
		})
	}
}

func TestProblemsAcceptsALabelJustOverTheGapFromABar(t *testing.T) {
	pic, region := plantedBase(t)
	pic.write(region, vg.Point{X: 122.5, Y: 80}, nil, block{{{bold(labelSize, inkPrimary), "planted"}}})
	assert.Empty(t, problems(pic))
}

func TestProblemsAcceptsALabelInsideItsOwnArea(t *testing.T) {
	pic, region := plantedBase(t)
	cell := pic.fillRect(region, rgb(0xcde2fb), rect(250, 20, 330, 70))
	pic.write(region, vg.Point{X: 254, Y: 54}, cell, block{{{bold(labelSize, inkPrimary), "planted"}}})
	assert.Empty(t, problems(pic))
}

// composite is top as it looks drawn over an opaque bottom.
func composite(top, bottom color.NRGBA) color.NRGBA {
	alpha := float64(top.A) / 255
	mix := func(over, under uint8) uint8 {
		return uint8(math.Round(float64(over)*alpha + float64(under)*(1-alpha)))
	}
	return color.NRGBA{R: mix(top.R, bottom.R), G: mix(top.G, bottom.G), B: mix(top.B, bottom.B), A: 0xff}
}
