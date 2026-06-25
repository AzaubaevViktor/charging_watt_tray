package main

import (
	"fmt"
	"math"
)

// renderBattery builds a vertical battery (terminal on top) with a fill rising
// from the bottom proportional to level, as an SVG that NSImage rasterises
// itself — no PNG, no drawing library. It is a template image (black + alpha),
// so macOS tints it for light/dark menu bars.
//
// The viewBox is square (18x18) with the 10-wide battery centered: systray
// forces every status icon to 16x16 ([image setSize:16,16]), so a non-square
// viewBox would be stretched.
func renderBattery(level int) []byte {
	const innerTop, innerBottom = 3.6, 16.2
	frac := math.Max(0, math.Min(1, float64(level)/100))
	fillH := math.Max(0.6, frac*(innerBottom-innerTop))
	fillY := innerBottom - fillH
	svg := fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 18 18">`+
			`<rect x="7" y="0.4" width="4" height="2" rx="0.9" fill="black" fill-opacity="0.5"/>`+
			`<rect x="5" y="2.4" width="8" height="15" rx="2" fill="none" stroke="black" `+
			`stroke-width="1.3" stroke-opacity="0.5"/>`+
			`<rect x="6.3" y="%.2f" width="5.4" height="%.2f" rx="1" fill="black"/></svg>`,
		fillY, fillH)
	return []byte(svg)
}
