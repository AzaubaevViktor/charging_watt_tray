package main

// Small AppKit/Foundation helpers that systray doesn't expose:
//   - shrinkTitleFont: systray caps the icon at 16x16 and offers no title-font
//     knob, so the menu-bar text renders at the default ~14pt. The original
//     Python app set 11pt. systray keeps its NSStatusItem in a private ivar of
//     its app delegate; we reach it by KVC and set the button font, guarded so
//     a miss can never crash.
//   - lowPowerMode: macOS Low Power Mode, used to back off the refresh rate.

/*
#cgo darwin CFLAGS: -x objective-c -fobjc-arc
#cgo darwin LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

static void shrink_title_font(double size) {
	@try {
		id delegate = [NSApp delegate];
		if (!delegate) return;
		NSStatusItem *si = [delegate valueForKey:@"statusItem"];
		if (si && si.button) {
			si.button.font = [NSFont systemFontOfSize:size];
		}
	} @catch (NSException *e) {}
}

static int low_power_mode(void) {
	if (@available(macOS 12.0, *)) {
		return [[NSProcessInfo processInfo] isLowPowerModeEnabled] ? 1 : 0;
	}
	return 0;
}
*/
import "C"

// shrinkTitleFont sets the menu-bar title font size. Call it on the main thread
// (e.g. from onReady) once the status item exists.
func shrinkTitleFont(size float64) { C.shrink_title_font(C.double(size)) }

// lowPowerMode reports whether macOS Low Power Mode is enabled.
func lowPowerMode() bool { return C.low_power_mode() != 0 }
