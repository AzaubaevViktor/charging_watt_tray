package main

import (
	"fmt"
	"math"
	"strconv"
	"unicode/utf8"
)

// figureSpace (U+2007) is a blank the width of a digit, so padding with it keeps
// numbers in a fixed-width column even in the menu bar's proportional font.
const figureSpace = " "

// wattTitle formats watts for the menu-bar title at a constant width: the
// integer part is left-padded with figure spaces to `digits`, so the title
// doesn't jiggle when the value crosses from one digit to two (todo #1).
func wattTitle(w float64, digits int) string {
	n := strconv.Itoa(int(math.Round(w)))
	for utf8.RuneCountInString(n) < digits {
		n = figureSpace + n
	}
	return n + "W"
}

var fractions = []struct {
	v float64
	g string
}{
	{0, ""}, {1.0 / 6, "⅙"}, {1.0 / 5, "⅕"}, {1.0 / 4, "¼"}, {1.0 / 3, "⅓"},
	{2.0 / 5, "⅖"}, {1.0 / 2, "½"}, {3.0 / 5, "⅗"}, {2.0 / 3, "⅔"}, {3.0 / 4, "¾"},
	{4.0 / 5, "⅘"}, {5.0 / 6, "⅚"}, {1.0, ""},
}

// fmtTime is the compact form for the title: "20m" under an hour, else hours
// with the nearest unicode fraction ("2⅓h", "1½h"). "—" when unknown or zero.
func fmtTime(minutes int) string {
	if minutes <= 0 || minutes >= 65535 {
		return "—"
	}
	if minutes < 60 {
		return strconv.Itoa(minutes) + "m"
	}
	hours := float64(minutes) / 60.0
	whole := int(hours)
	frac := hours - float64(whole)
	best := fractions[0]
	for _, f := range fractions {
		if math.Abs(f.v-frac) < math.Abs(best.v-frac) {
			best = f
		}
	}
	glyph := best.g
	if best.v == 1.0 { // rounded up to the next whole hour
		whole++
		glyph = ""
	}
	return strconv.Itoa(whole) + glyph + "h"
}

// fmtTimeExact is the precise form for the menu: "20m" or "XhYYm".
func fmtTimeExact(minutes int) string {
	if minutes <= 0 || minutes >= 65535 {
		return "—"
	}
	if minutes < 60 {
		return strconv.Itoa(minutes) + "m"
	}
	return fmt.Sprintf("%dh%02dm", minutes/60, minutes%60)
}

// truncate caps a string to n runes (for long process names in the menu).
func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}
