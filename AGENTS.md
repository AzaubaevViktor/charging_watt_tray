# Agent guidelines

A macOS menu-bar app (Go: `fogleman/gg` + `getlantern/systray`) that shows
real-time power: charging / discharging watts, battery % and time-to-full/empty,
and a CPU/GPU/Other breakdown with the top energy-using apps. Ported from the
original Python/rumps app. See `TODO.md` for open ideas.

## Commits

- **Every commit message must start with the `[AI]` prefix.**
  Example: `[AI] anchor per-process watts to CPU+GPU instead of total power`
- **Commit every completed step** — don't batch unrelated changes; once a
  discrete piece of work builds and runs, commit it before moving on.
- Keep the subject line short and imperative; add a body when the change needs it.

## Working on the code

- Files (keep the pure logic free of GUI deps so it stays unit-testable):
  - `smc.go` — AppleSMC reader via IOKit (cgo). Real-time, no sudo. Keys:
    `PSTR` (system total), `PDTR` (adapter in), `PBAT` (battery).
  - `ioreport.go` — Apple Silicon CPU/GPU power from the private IOReport
    framework (cgo, `dlopen`'d `/usr/lib/libIOReport.dylib`, group "Energy
    Model"). No sudo.
  - `battery.go` — state/level/time from `ioreg AppleSmartBattery`.
  - `top.go` — `top -l 2` energy-impact parsing (pure `parseTop`).
  - `format.go` — `fmtTime`/`fmtTimeExact` and `wattTitle` (the fixed-width,
    figure-space-padded menu-bar watt field). Pure; unit-tested.
  - `icon.go` — the on-the-fly battery template PNG (`gg`).
  - `app.go` — the `App`: readers, 5 s tick loop, state, title/menu rendering.
  - `menu.go` — menu layout and click wiring; thin `main.go`.

- Power model is single-source so the numbers add up: `System` = `PSTR`, and
  `System = CPU + GPU + Other`. Per-process watts are `top`'s energy-impact
  scores normalized to the real CPU+GPU power (display/peripherals are *not*
  blamed on apps — they live in `Other`).

- Avoid these data traps: ioreg `PowerTelemetryData` is laggy (tens of seconds)
  and reports negatives as unsigned (giant numbers) — use SMC/IOReport for
  power, and `ioreg AppleSmartBattery` only for state/level/time. `top`'s POWER
  column is a unitless "energy impact", not watts.

- No third-party deps beyond `gg` (drawing) and `systray` (menu bar); subprocess
  out to `ioreg` / `top`. Don't add network/numeric libraries for this.

- Run it with `./charging.sh` (builds then runs). After edits:
  `go build -o charging-app . && go test ./...`.
