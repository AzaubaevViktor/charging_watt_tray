package main

import (
	"os/exec"
	"regexp"
	"strconv"
)

// Battery is the charge state read from ioreg's AppleSmartBattery node.
type Battery struct {
	External bool // running on the adapter
	Charging bool // actively charging
	Level    int  // CurrentCapacity, percent
	ToFull   int  // AvgTimeToFull, minutes (65535 = unknown)
	ToEmpty  int  // AvgTimeToEmpty, minutes (65535 = unknown)
}

var fieldRe = map[string]*regexp.Regexp{}

func batteryField(out, name string) string {
	re, ok := fieldRe[name]
	if !ok {
		re = regexp.MustCompile(`"` + name + `" = ([-0-9A-Za-z]+)`)
		fieldRe[name] = re
	}
	if m := re.FindStringSubmatch(out); m != nil {
		return m[1]
	}
	return ""
}

// readBattery shells out to ioreg for state/level/time. Power does not come from
// here — ioreg's PowerTelemetryData is laggy and wraps negatives as unsigned;
// see smc.go / ioreport.go for real-time watts.
func readBattery() Battery {
	out, err := exec.Command("ioreg", "-rw0", "-c", "AppleSmartBattery").Output()
	if err != nil {
		return Battery{}
	}
	s := string(out)
	atoi := func(name string) int { n, _ := strconv.Atoi(batteryField(s, name)); return n }
	return Battery{
		External: batteryField(s, "ExternalConnected") == "Yes",
		Charging: batteryField(s, "IsCharging") == "Yes",
		Level:    atoi("CurrentCapacity"),
		ToFull:   atoi("AvgTimeToFull"),
		ToEmpty:  atoi("AvgTimeToEmpty"),
	}
}
