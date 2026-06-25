// Command charging is a macOS menu-bar power monitor: charging / discharging
// watts, battery % and time-to-full/empty, and a CPU/GPU/Other breakdown with
// the top energy-using apps.
//
// Real-time power comes from the AppleSMC sensors (smc.go) and Apple Silicon
// CPU/GPU power from the private IOReport framework (ioreport.go) — both without
// sudo. Battery state is parsed from ioreg, and per-app attribution from `top`.
// The menu bar is driven by getlantern/systray and the icon drawn with
// fogleman/gg. This is a Go port of the original Python/rumps app.
package main

import "github.com/getlantern/systray"

func main() {
	app := newApp()
	systray.Run(app.onReady, app.onExit)
}
