package chart

import (
	"image/color"
	"math"

	xfont "golang.org/x/image/font"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/font"
	"gonum.org/v1/plot/text"
)

func rgb(hex uint32) color.NRGBA {
	return color.NRGBA{R: uint8(hex >> 16), G: uint8(hex >> 8), B: uint8(hex), A: 0xff}
}

func withAlpha(base color.NRGBA, alpha float64) color.NRGBA {
	base.A = uint8(math.Round(alpha * 255))
	return base
}

var (
	surface      = rgb(0xfcfcfb)
	inkPrimary   = rgb(0x0b0b0b)
	inkSecondary = rgb(0x52514e)
	inkMuted     = rgb(0x898781)
	spendInk     = rgb(0x1f1f1d)
	gridInk      = rgb(0xe1e0d9)
	axisInk      = rgb(0xc3c2b7)
	withinInk    = rgb(0x2a78d6)
	overInk      = rgb(0xd03b3b)
	white        = rgb(0xffffff)
)

// Sizes in points on a canvas laid out for a Telegram photo bubble about 330 dp wide.
const (
	canvasWidth = 380
	pageMargin  = 14
	pageWidth   = canvasWidth - 2*pageMargin
	titleSize   = 17
	statusSize  = 14.5
	labelSize   = 13
	noteSize    = 12
	minTextSize = 11
	// textGap is the clear space every placed text keeps from other texts and marks.
	textGap = 2
)

// Liberation Sans is registered in font.DefaultCache by gonum's plot package, so nothing is
// registered here and no font global is written.
func textStyle(size font.Length, weight xfont.Weight, ink color.Color) text.Style {
	return text.Style{
		Color:   ink,
		Font:    font.Font{Typeface: "Liberation", Variant: "Sans", Weight: weight, Size: size},
		Handler: plot.DefaultTextHandler,
	}
}

func regular(size font.Length, ink color.Color) text.Style {
	return textStyle(size, xfont.WeightNormal, ink)
}

func bold(size font.Length, ink color.Color) text.Style {
	return textStyle(size, xfont.WeightBold, ink)
}

// luminance is the WCAG relative luminance of an opaque colour.
func luminance(shade color.NRGBA) float64 {
	channel := func(value uint8) float64 {
		linear := float64(value) / 255
		if linear <= 0.04045 {
			return linear / 12.92
		}
		return math.Pow((linear+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(shade.R) + 0.7152*channel(shade.G) + 0.0722*channel(shade.B)
}

// contrast is the WCAG contrast ratio of two opaque colours.
func contrast(first, second color.NRGBA) float64 {
	lighter, darker := luminance(first), luminance(second)
	if lighter < darker {
		lighter, darker = darker, lighter
	}
	return (lighter + 0.05) / (darker + 0.05)
}
