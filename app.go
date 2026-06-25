package main

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/getlantern/systray"
)

const (
	interval         = 5  // seconds between refreshes
	lowPowerInterval = 15 // …slower in macOS Low Power Mode
	topCount         = 5  // processes shown in the "Top usage" submenu
	topEvery         = 2  // re-run `top` every N ticks
	iconLevel        = 5  // re-render the icon only when level crosses a 5% bucket
	titleFontSize    = 11 // menu-bar title font (systray defaults to ~14pt)
)

// App holds the readers, the cached state, and the menu items.
type App struct {
	mu     sync.Mutex
	smc    *SMC
	energy *EnergyModel

	tick    int
	iconKey string
	compute float64 // cpu+gpu watts, the anchor for per-process attribution

	// refresh-rate tracking (average of the last 10 value changes)
	lastPower  *float64
	lastChange time.Time
	intervals  []float64

	// `top` runs in the background; results arrive on this channel
	topResults  chan []procRow
	topInflight bool
	topRows     []procRow

	// menu items
	adapter, system, cpu, gpu, other *systray.MenuItem
	battery, timeItem, status        *systray.MenuItem
	topParent                        *systray.MenuItem
	topItems                         []*systray.MenuItem
}

func newApp() *App {
	a := &App{
		smc:        openSMC(),
		energy:     openEnergyModel(),
		topResults: make(chan []procRow, 1),
		lastChange: time.Now(),
	}
	if a.smc == nil {
		fmt.Println("SMC unavailable — power will read as 0")
	}
	if a.energy == nil {
		fmt.Println("IOReport unavailable — anchoring top to total power")
	}
	return a
}

func (a *App) onReady() {
	systray.SetTitle("…W")
	systray.SetTooltip("Wattmeter")
	a.buildMenu()
	shrinkTitleFont(titleFontSize) // systray has no font knob; match the old 11pt
	a.setIcon(50)                  // placeholder until the first refresh
	go a.loop()
}

// loop refreshes on a cadence that slows down in Low Power Mode.
func (a *App) loop() {
	for {
		lowPower := lowPowerMode()
		d := interval
		if lowPower {
			d = lowPowerInterval
		}
		time.Sleep(time.Duration(d) * time.Second)
		a.refresh(lowPower)
	}
}

func (a *App) onExit() {}

// refresh reads every source and repaints the title and menu. It is serialized
// by the lock so the menu-item click handlers don't race the ticker.
func (a *App) refresh(lowPower bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	sysw, adapter, batt := a.readPower()
	b := readBattery()
	level := b.Level

	if a.energy != nil {
		cpu, gpu := a.energy.Sample()
		a.compute = cpu + gpu
		other := math.Max(0, sysw-cpu-gpu) // display, peripherals, DRAM, base
		a.cpu.SetTitle(fmt.Sprintf("🧠 CPU: %.1fW", cpu))
		a.gpu.SetTitle(fmt.Sprintf("🎮 GPU: %.1fW", gpu))
		a.other.SetTitle(fmt.Sprintf("🖥 Other: %.1fW", other))
	} else {
		a.compute = sysw // no IOReport: attribute against whole-system power
		a.cpu.SetTitle("🧠 CPU: —")
		a.gpu.SetTitle("🎮 GPU: —")
		a.other.SetTitle("🖥 Other: —")
	}

	a.setIcon(level)
	// System = CPU + GPU + Other (single source, so the numbers add up).
	a.system.SetTitle(fmt.Sprintf("∑ System: %.1fW", sysw))

	var title string
	switch {
	case b.External && b.Charging:
		title = withTime(level, fmtTime(b.ToFull), batt)
		a.adapter.SetTitle(fmt.Sprintf("⚡ Adapter: %.1fW", adapter))
		a.battery.SetTitle(fmt.Sprintf("🔋 To battery: %.1fW", batt))
		a.timeItem.SetTitle("⏱ Full in " + fmtTimeExact(b.ToFull))
	case b.External:
		title = fmt.Sprintf("%d%% %s", level, wattTitle(sysw, 2))
		a.adapter.SetTitle(fmt.Sprintf("⚡ Adapter: %.1fW", adapter))
		a.battery.SetTitle(fmt.Sprintf("🔋 Battery: full (%d%%)", level))
		a.timeItem.SetTitle("⏱ Fully charged")
	default:
		title = withTime(level, fmtTime(b.ToEmpty), sysw)
		a.adapter.SetTitle("⚡ Adapter: not connected")
		// PSTR (consumption) matches System; the raw PBAT rail is ~10% higher
		// due to DC-DC losses and would look inconsistent.
		a.battery.SetTitle(fmt.Sprintf("🔋 Draining: %.1fW", sysw))
		a.timeItem.SetTitle("⏱ Empty in " + fmtTimeExact(b.ToEmpty))
	}

	avg := a.trackRefresh(sysw)
	status := fmt.Sprintf("🔋 %d%%   ·   refresh ~%.1fs", level, avg)
	if lowPower {
		status += "   ·   🪫 low power"
	}
	a.status.SetTitle(status)
	systray.SetTitle(title)

	a.pollTop()
	a.renderTop()
}

// withTime composes the title "NN% [time] WW" with a fixed-width watt field.
func withTime(level int, remaining string, watts float64) string {
	title := fmt.Sprintf("%d%%", level)
	if remaining != "—" {
		title += " " + remaining
	}
	return title + " " + wattTitle(watts, 2)
}

func (a *App) readPower() (sys, adapter, batt float64) {
	if a.smc == nil {
		return 0, 0, 0
	}
	sys, _ = a.smc.ReadFloat("PSTR")
	adapter, _ = a.smc.ReadFloat("PDTR")
	batt, _ = a.smc.ReadFloat("PBAT")
	return sys, adapter, batt
}

func (a *App) setIcon(level int) {
	key := fmt.Sprintf("batt-%d", level/iconLevel)
	if a.iconKey == key {
		return
	}
	a.iconKey = key
	png := renderBattery(level)
	systray.SetTemplateIcon(png, png)
}

// trackRefresh records how often the power value actually changes and returns
// the average interval over the last 10 changes.
func (a *App) trackRefresh(power float64) float64 {
	if a.lastPower == nil || math.Abs(power-*a.lastPower) >= 0.05 {
		now := time.Now()
		if a.lastPower != nil {
			a.intervals = append(a.intervals, now.Sub(a.lastChange).Seconds())
			if len(a.intervals) > 10 {
				a.intervals = a.intervals[1:]
			}
		}
		a.lastChange = now
		a.lastPower = &power
	}
	if len(a.intervals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range a.intervals {
		sum += v
	}
	return sum / float64(len(a.intervals))
}

func (a *App) pollTop() {
	select {
	case rows := <-a.topResults:
		a.topRows = rows
		a.topInflight = false
	default:
	}
	if !a.topInflight && a.tick%topEvery == 0 {
		a.topInflight = true
		go func() { a.topResults <- runTop() }()
	}
	a.tick++
}

// renderTop lists apps with watts normalised to CPU+GPU power; energy-impact
// only models compute, so display/peripherals stay in "Other", not on apps.
func (a *App) renderTop() {
	total := 0.0
	for _, r := range a.topRows {
		total += r.score
	}
	if total == 0 {
		total = 1
	}
	for i, it := range a.topItems {
		if i < len(a.topRows) {
			w := a.compute * a.topRows[i].score / total
			it.SetTitle(fmt.Sprintf("%s   ~%.1fW", truncate(a.topRows[i].command, 24), w))
		} else {
			it.SetTitle("—")
		}
	}
	a.topParent.SetTitle(fmt.Sprintf("Top usage  %.1fW", a.compute))
}
