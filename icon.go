package main

import (
	"bytes"
	"image/png"
	"math"

	"github.com/fogleman/gg"
)

// battScale renders the 10x18 icon at 4x so it stays crisp on retina; systray
// scales the PNG down to the menu-bar height.
const battScale = 4.0

// renderBattery draws a vertical battery (terminal on top) with a fill rising
// from the bottom proportional to level, as a black-on-transparent template PNG
// so macOS tints it for light/dark menu bars.
func renderBattery(level int) []byte {
	const vbW, vbH = 10.0, 18.0
	dc := gg.NewContext(int(vbW*battScale), int(vbH*battScale))
	dc.Scale(battScale, battScale)
	black := func(a float64) { dc.SetRGBA(0, 0, 0, a) }

	// terminal nub
	black(0.5)
	dc.DrawRoundedRectangle(3, 0.4, 4, 2, 0.9)
	dc.Fill()

	// body outline (stroke only); gg ignores the scale for line width, so scale it
	black(0.5)
	dc.DrawRoundedRectangle(1, 2.4, 8, 15, 2)
	dc.SetLineWidth(1.3 * battScale)
	dc.Stroke()

	// fill, rising from the bottom
	const innerTop, innerBottom = 3.6, 16.2
	frac := math.Max(0, math.Min(1, float64(level)/100))
	fillH := math.Max(0.6, frac*(innerBottom-innerTop))
	r := math.Min(1, fillH/2)
	black(1.0)
	dc.DrawRoundedRectangle(2.3, innerBottom-fillH, 5.4, fillH, r)
	dc.Fill()

	var buf bytes.Buffer
	_ = png.Encode(&buf, dc.Image())
	return buf.Bytes()
}
