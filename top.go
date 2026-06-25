package main

import (
	"os/exec"
	"strconv"
	"strings"
)

// procRow is one process and its `top` energy-impact score (a unitless number,
// not watts — it is later normalised to real CPU+GPU power).
type procRow struct {
	command string
	score   float64
}

// runTop runs `top -l 2` and returns the parsed rows from its second sample,
// sorted by power. It blocks for ~1s, so call it from a goroutine.
func runTop() []procRow {
	out, err := exec.Command("top", "-l", "2", "-o", "power",
		"-stats", "pid,command,power", "-s", "1").Output()
	if err != nil {
		return nil
	}
	return parseTop(string(out))
}

// parseTop extracts (command, score) rows from the last sample of `top` output.
func parseTop(output string) []procRow {
	lines := strings.Split(output, "\n")
	lastHeader := -1
	for i, ln := range lines {
		if strings.HasPrefix(ln, "PID") {
			lastHeader = i
		}
	}
	if lastHeader < 0 {
		return nil
	}
	var rows []procRow
	for _, ln := range lines[lastHeader+1:] {
		parts := strings.Fields(ln)
		if len(parts) < 3 {
			continue
		}
		score, err := strconv.ParseFloat(parts[len(parts)-1], 64)
		if err != nil {
			continue
		}
		rows = append(rows, procRow{strings.Join(parts[1:len(parts)-1], " "), score})
	}
	return rows
}
