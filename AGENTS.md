# Agent guidelines

A macOS menu-bar app (rumps + pyobjc) that shows real-time power: charging /
discharging watts, battery % and time-to-full/empty, and a CPU/GPU/Other
breakdown with top energy-using apps. See `__main__.py` for the app and
`TODO.md` for open ideas.

## Commits

- **Every commit message must start with the `[AI]` prefix.**
  Example: `[AI] anchor per-process watts to CPU+GPU instead of total power`
- **Commit every completed step** — don't batch unrelated changes; once a
  discrete piece of work builds and runs, commit it before moving on.
- Keep the subject line short and imperative; add a body when the change needs it.

## Working on the code

- Three modules, no build step:
  - `__main__.py` — the `rumps.App`, the 5 s timer, menu rendering, the
    on-the-fly SVG battery icon, and `top` parsing.
  - `smc.py` — AppleSMC reader via IOKit/ctypes. Real-time, no sudo. Keys:
    `PSTR` (system total), `PDTR` (adapter in), `PBAT` (battery).
  - `ioreport.py` — Apple Silicon CPU/GPU power from the private IOReport
    framework (`/usr/lib/libIOReport.dylib`, group "Energy Model"). No sudo.

- Power model is single-source so the numbers add up: `System` = `PSTR`, and
  `System = CPU + GPU + Other`. Per-process watts are `top`'s energy-impact
  scores normalized to the real CPU+GPU power (display/peripherals are *not*
  blamed on apps — they live in `Other`).

- Avoid these data traps: ioreg `PowerTelemetryData` is laggy (tens of seconds)
  and reports negatives as unsigned (giant numbers) — use SMC/IOReport for
  power, and ioreg `AppleSmartBattery` only for state/level/time. `top`'s POWER
  column is a unitless "energy impact", not watts.

- No third-party deps beyond `rumps` (which pulls pyobjc); subprocess out to
  `ioreg` / `top`. Don't add network/numeric libraries for this.

- Run it with `./charging.sh`; quick sanity check after edits:
  `.venv/bin/python -m py_compile __main__.py smc.py ioreport.py`
