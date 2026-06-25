package main

import "testing"

func TestFmtTime(t *testing.T) {
	cases := map[int]string{
		0:     "—",
		65535: "—",
		20:    "20m",
		59:    "59m",
		60:    "1h",
		80:    "1⅓h",
		90:    "1½h",
		140:   "2⅓h",
		170:   "2⅚h",
		178:   "3h", // 2h58m rounds up to the next whole hour
	}
	for min, want := range cases {
		if got := fmtTime(min); got != want {
			t.Errorf("fmtTime(%d) = %q, want %q", min, got, want)
		}
	}
}

func TestFmtTimeExact(t *testing.T) {
	cases := map[int]string{0: "—", 65535: "—", 45: "45m", 60: "1h00m", 145: "2h25m"}
	for min, want := range cases {
		if got := fmtTimeExact(min); got != want {
			t.Errorf("fmtTimeExact(%d) = %q, want %q", min, got, want)
		}
	}
}

func TestWattTitle(t *testing.T) {
	// single- and double-digit values must occupy the same column width.
	if a, b := wattTitle(5, 2), wattTitle(12, 2); len([]rune(a)) != len([]rune(b)) {
		t.Errorf("width mismatch: %q vs %q", a, b)
	}
	if got := wattTitle(5, 2); got != figureSpace+"5W" {
		t.Errorf("wattTitle(5,2) = %q", got)
	}
	if got := wattTitle(12, 2); got != "12W" {
		t.Errorf("wattTitle(12,2) = %q", got)
	}
	if got := wattTitle(7.6, 2); got != figureSpace+"8W" {
		t.Errorf("wattTitle(7.6,2) = %q (want rounded)", got)
	}
}
