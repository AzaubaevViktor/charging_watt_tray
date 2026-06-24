import re
import subprocess
import time
from collections import deque

import rumps
from AppKit import (NSFont, NSFontAttributeName, NSImage,
                    NSMutableParagraphStyle, NSParagraphStyleAttributeName,
                    NSTextAlignmentRight, NSTextTab)
from Foundation import NSAttributedString, NSData

from ioreport import EnergyModel
from smc import SMC

INTERVAL = 5           # seconds between refreshes
TOP_COUNT = 5          # processes shown in the "Top usage" submenu
TOP_EVERY = 2          # refresh top list every N ticks (N * INTERVAL seconds)
FONT_SIZE = 11.0       # menu-bar title font size (smaller than default)

try:
    smc = SMC()
except OSError as exc:  # pragma: no cover - hardware dependent
    print("SMC unavailable, falling back to ioreg:", exc)
    smc = None

try:
    energy = EnergyModel()
except OSError as exc:  # pragma: no cover - hardware dependent
    print("IOReport unavailable, anchoring top to total power:", exc)
    energy = None


def signed(value, bits=64):
    """ioreg reports negative integers as unsigned two's complement."""
    if value >= 1 << (bits - 1):
        value -= 1 << bits
    return value


def read_power():
    """Real-time power in watts: (system, adapter, battery).

      system  - total current consumption (SMC PSTR).
      adapter - power delivered by the charger (SMC PDTR; ~0 on battery).
      battery - power flowing through the battery (SMC PBAT).
    """
    if smc is not None:
        return (smc.read_float("PSTR") or 0.0,
                smc.read_float("PDTR") or 0.0,
                smc.read_float("PBAT") or 0.0)

    # Fallback: ioreg telemetry (laggy, but works without SMC).
    out = subprocess.check_output(["ioreg", "-rw0", "-c", "AppleSmartBattery"]).decode()
    m = re.search(r'"PowerTelemetryData" = \{(.*?)\}', out, re.S)
    pt = dict(re.findall(r'"([^"]+)"=(-?\d+)', m.group(1))) if m else {}
    sysw = signed(int(pt.get("SystemLoad", 0))) / 1000.0
    adapter = int(pt.get("SystemPowerIn", 0)) / 1000.0
    return sysw, adapter, abs(sysw - adapter)


def read_battery():
    """State from ioreg: external, charging, level %, minutes-to-full/empty."""
    out = subprocess.check_output(["ioreg", "-rw0", "-c", "AppleSmartBattery"]).decode()

    def field(name):
        m = re.search(r'"%s" = ([-0-9A-Za-z]+)' % name, out)
        return m.group(1) if m else ""

    return {
        "external": field("ExternalConnected") == "Yes",
        "charging": field("IsCharging") == "Yes",
        "level": int(field("CurrentCapacity") or 0),
        "to_full": int(field("AvgTimeToFull") or 0),
        "to_empty": int(field("AvgTimeToEmpty") or 0),
    }


_FRACTIONS = [(0.0, ""), (1 / 6, "⅙"), (1 / 5, "⅕"), (1 / 4, "¼"),
              (1 / 3, "⅓"), (2 / 5, "⅖"), (1 / 2, "½"), (3 / 5, "⅗"),
              (2 / 3, "⅔"), (3 / 4, "¾"), (4 / 5, "⅘"), (5 / 6, "⅚"),
              (1.0, "")]


def fmt_time(minutes):
    """Minutes -> '20m' under an hour, else hours with the nearest fraction
    ('2⅓h', '1½h'). '—' when unknown (65535) or zero."""
    if not minutes or minutes >= 65535:
        return "—"
    if minutes < 60:
        return f"{minutes}m"
    hours = minutes / 60.0
    whole = int(hours)
    value, glyph = min(_FRACTIONS, key=lambda f: abs(f[0] - (hours - whole)))
    if value == 1.0:  # rounded up to the next whole hour
        whole += 1
        glyph = ""
    return f"{whole}{glyph}h"


def fmt_time_exact(minutes):
    """Precise time for the menu: '20m' under an hour, else 'XhYYm'."""
    if not minutes or minutes >= 65535:
        return "—"
    if minutes < 60:
        return f"{minutes}m"
    return f"{minutes // 60}h{minutes % 60:02d}m"


# --- on-the-fly template icons ----------------------------------------------

def nsimage_from_svg(svg, width, height):
    raw = svg.encode("utf-8")
    data = NSData.dataWithBytes_length_(raw, len(raw))
    img = NSImage.alloc().initWithData_(data)
    img.setSize_((width, height))
    img.setTemplate_(True)
    return img


def battery_icon(level):
    """Vertical battery outline (terminal on top) with a fill rising from the
    bottom proportional to charge level."""
    inner_top, inner_bottom = 3.6, 16.2
    inner_h = inner_bottom - inner_top
    fill_h = max(0.6, min(1.0, level / 100.0) * inner_h)
    fill_y = inner_bottom - fill_h
    svg = (
        '<svg xmlns="http://www.w3.org/2000/svg" width="10" height="18" '
        'viewBox="0 0 10 18">'
        '<rect x="3" y="0.4" width="4" height="2" rx="0.9" fill="black" '
        'fill-opacity="0.5"/>'
        '<rect x="1" y="2.4" width="8" height="15" rx="2" fill="none" '
        'stroke="black" stroke-width="1.3" stroke-opacity="0.5"/>'
        f'<rect x="2.3" y="{fill_y:.2f}" width="5.4" height="{fill_h:.2f}" '
        'rx="1" fill="black"/></svg>'
    )
    return nsimage_from_svg(svg, 10, 18)


def aligned(item, left, right, width=230.0):
    """Set a menu item's title with `right` flush to the right edge.

    Menus use a proportional font, so space-padding can't align columns;
    a right-aligned tab stop does it regardless of glyph widths.
    """
    ps = NSMutableParagraphStyle.alloc().init()
    tab = NSTextTab.alloc().initWithTextAlignment_location_options_(
        NSTextAlignmentRight, width, None)
    ps.setTabStops_([tab])
    attrs = {NSParagraphStyleAttributeName: ps,
             NSFontAttributeName: NSFont.menuFontOfSize_(0)}
    title = NSAttributedString.alloc().initWithString_attributes_(
        f"{left}\t{right}", attrs)
    item._menuitem.setAttributedTitle_(title)


def parse_top(output):
    """Parse `top -l 2` 2nd sample -> [(command, energy_impact_score), ...]
    for ALL processes (sorted by power), so the scores can be normalized."""
    lines = output.splitlines()
    headers = [i for i, ln in enumerate(lines) if ln.startswith("PID")]
    if not headers:
        return []
    rows = []
    for ln in lines[headers[-1] + 1:]:
        parts = ln.split()
        if len(parts) < 3:
            continue
        try:
            score = float(parts[-1])
        except ValueError:
            continue
        rows.append((" ".join(parts[1:-1]), score))
    return rows


if __name__ == '__main__':
    adapter_item = rumps.MenuItem("Adapter")
    load_item = rumps.MenuItem("System")
    cpu_item = rumps.MenuItem("CPU")
    gpu_item = rumps.MenuItem("GPU")
    other_item = rumps.MenuItem("Other")
    battery_item = rumps.MenuItem("Battery")
    time_item = rumps.MenuItem("Time")
    status_item = rumps.MenuItem("Status")
    top_parent = rumps.MenuItem("Top usage")
    top_items = [rumps.MenuItem("…") for _ in range(TOP_COUNT)]
    for it in top_items:
        top_parent.add(it)

    state = {
        "icon_key": None,
        "tick": 0,
        "proc": None,
        "font_set": False,
        "last_power": None,
        "last_change": time.monotonic(),
        "intervals": deque(maxlen=10),
        "total": 0.0,       # SMC PSTR (whole-system watts)
        "cpu": 0.0,         # IOReport CPU watts
        "gpu": 0.0,         # IOReport GPU watts
        "compute": 0.0,     # cpu + gpu (process attribution anchor)
        "top_rows": [],     # cached (command, energy_impact_score) from `top`
    }

    def set_icon(image, key):
        if state["icon_key"] == key:
            return
        state["icon_key"] = key
        app._icon = "<generated>"  # marker so rumps keeps the image
        app._icon_nsimage = image
        try:
            app._nsapp.setStatusBarIcon()
        except AttributeError:
            pass  # before run(): rumps applies _icon_nsimage on launch

    def shrink_font():
        if state["font_set"]:
            return
        try:
            app._nsapp.nsstatusitem.button().setFont_(NSFont.systemFontOfSize_(FONT_SIZE))
            state["font_set"] = True
        except Exception:
            pass

    def track_refresh(power):
        """Record how often the power value actually changes (avg of last 10)."""
        if state["last_power"] is None or abs(power - state["last_power"]) >= 0.05:
            now = time.monotonic()
            if state["last_power"] is not None:
                state["intervals"].append(now - state["last_change"])
            state["last_change"] = now
            state["last_power"] = power
        if state["intervals"]:
            return sum(state["intervals"]) / len(state["intervals"])
        return 0.0

    def poll_top():
        """Run `top` in the background; cache process scores when it finishes."""
        proc = state["proc"]
        if proc is None:
            if state["tick"] % TOP_EVERY == 0:
                state["proc"] = subprocess.Popen(
                    ["top", "-l", "2", "-o", "power",
                     "-stats", "pid,command,power", "-s", "1"],
                    stdout=subprocess.PIPE, stderr=subprocess.DEVNULL)
        elif proc.poll() is not None:
            output = proc.stdout.read().decode("utf-8", "replace")
            state["proc"] = None
            state["top_rows"] = parse_top(output)
        state["tick"] += 1

    def render_top():
        """Render the app list from cached scores, anchored to CPU+GPU watts.

        energy-impact only models compute, so per-process watts are normalized
        to the real CPU+GPU power (IOReport). The submenu lists apps only; the
        total apps consumption is shown on the parent (before the ▸ arrow).
        """
        rows = state["top_rows"]
        total_score = sum(s for _, s in rows) or 1.0
        compute = state["compute"]
        for i, item in enumerate(top_items):
            if i < len(rows):
                cmd, score = rows[i]
                aligned(item, cmd[:24], f"~{compute * score / total_score:.1f}W")
            else:
                item.title = "—"
        top_parent.title = f"Top usage  {compute:.1f}W"

    def watt_check(sender):
        shrink_font()
        sysw, adapter, batt = read_power()
        b = read_battery()
        level = b["level"]
        state["total"] = sysw
        if energy is not None:
            cpu_w, gpu_w = energy.sample()
            state["cpu"], state["gpu"] = cpu_w, gpu_w
            state["compute"] = cpu_w + gpu_w
            other = max(0.0, sysw - cpu_w - gpu_w)  # display, peripherals, DRAM, base
            cpu_item.title = f"\U0001f9e0 CPU: {cpu_w:.1f}W"
            gpu_item.title = f"\U0001f3ae GPU: {gpu_w:.1f}W"
            other_item.title = f"\U0001f5a5 Other: {other:.1f}W"
        else:
            state["compute"] = sysw  # no IOReport: fall back to whole-system
            cpu_item.title = "\U0001f9e0 CPU: —"
            gpu_item.title = "\U0001f3ae GPU: —"
            other_item.title = "\U0001f5a5 Other: —"

        # Battery icon (filled to the current level) in every state.
        set_icon(battery_icon(level), f"batt-{level // 5}")
        # System = CPU + GPU + Other (single source, so the math adds up).
        load_item.title = f"∑ System: {sysw:.1f}W"

        if b["external"] and b["charging"]:
            # Charging: percent, time-to-full (omitted if unknown), watts in.
            remaining = fmt_time(b["to_full"])
            title = f"{level}%" + (f" {remaining}" if remaining != "—" else "") + f" {batt:.0f}W"
            adapter_item.title = f"⚡ Adapter: {adapter:.1f}W"
            battery_item.title = f"\U0001f50b To battery: {batt:.1f}W"
            time_item.title = f"⏱ Full in {fmt_time_exact(b['to_full'])}"
        elif b["external"]:
            # Plugged in but not charging (battery full).
            title = f"{level}% {sysw:.0f}W"
            adapter_item.title = f"⚡ Adapter: {adapter:.1f}W"
            battery_item.title = f"\U0001f50b Battery: full ({level}%)"
            time_item.title = "⏱ Fully charged"
        else:
            # Discharging: percent, time-to-empty, watts consumed.
            remaining = fmt_time(b["to_empty"])
            title = f"{level}%" + (f" {remaining}" if remaining != "—" else "") + f" {sysw:.0f}W"
            adapter_item.title = "⚡ Adapter: not connected"
            # Use PSTR (consumption) so it matches System; the raw battery rail
            # (PBAT) is ~10% higher due to DC-DC losses and would look inconsistent.
            battery_item.title = f"\U0001f50b Draining: {sysw:.1f}W"
            time_item.title = f"⏱ Empty in {fmt_time_exact(b['to_empty'])}"

        avg = track_refresh(sysw)
        status_item.title = f"\U0001f50b {level}%   ·   refresh ~{avg:.1f}s"

        app.title = title
        poll_top()
        render_top()

    app = rumps.App("Wattmeter", title="…W", template=True)
    set_icon(battery_icon(50), "init")
    app.menu = [adapter_item, load_item, cpu_item, gpu_item, other_item,
                battery_item, time_item, None, status_item, None, top_parent]

    rumps.Timer(watt_check, INTERVAL).start()
    app.run()
