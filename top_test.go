package main

import "testing"

func TestParseTop(t *testing.T) {
	sample := `Processes: 400 total
PID    COMMAND          POWER
1      WindowServer     12.3
PID    COMMAND          POWER
501    Google Chrome    45.6
733    com.apple.Web    10.0
9      kernel_task      0.0
notanumber line
`
	rows := parseTop(sample)
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3: %+v", len(rows), rows)
	}
	if rows[0].command != "Google Chrome" || rows[0].score != 45.6 {
		t.Errorf("row0 = %+v", rows[0])
	}
	if rows[1].command != "com.apple.Web" || rows[1].score != 10.0 {
		t.Errorf("row1 = %+v", rows[1])
	}
}

func TestParseTopEmpty(t *testing.T) {
	if rows := parseTop("no header here\n"); rows != nil {
		t.Errorf("want nil, got %+v", rows)
	}
}
