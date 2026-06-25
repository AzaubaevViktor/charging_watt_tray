package main

import "github.com/getlantern/systray"

func (a *App) buildMenu() {
	a.adapter = systray.AddMenuItem("Adapter", "")
	a.system = systray.AddMenuItem("System", "")
	a.cpu = systray.AddMenuItem("CPU", "")
	a.gpu = systray.AddMenuItem("GPU", "")
	a.other = systray.AddMenuItem("Other", "")
	a.battery = systray.AddMenuItem("Battery", "")
	a.timeItem = systray.AddMenuItem("Time", "")
	systray.AddSeparator()
	a.status = systray.AddMenuItem("Status", "")
	systray.AddSeparator()

	a.topParent = systray.AddMenuItem("Top usage", "")
	for i := 0; i < topCount; i++ {
		it := a.topParent.AddSubMenuItem("…", "")
		it.Disable()
		a.topItems = append(a.topItems, it)
	}
	systray.AddSeparator()
	refresh := systray.AddMenuItem("Refresh now", "")
	quit := systray.AddMenuItem("Quit", "")

	// the readout rows are display, not buttons — grey them out so they don't
	// highlight on hover (the submenu parent stays active so it opens).
	for _, it := range []*systray.MenuItem{
		a.adapter, a.system, a.cpu, a.gpu, a.other, a.battery, a.timeItem, a.status,
	} {
		it.Disable()
	}

	go func() {
		for range refresh.ClickedCh {
			a.refresh(lowPowerMode())
		}
	}()
	go func() {
		<-quit.ClickedCh
		systray.Quit()
	}()
}
