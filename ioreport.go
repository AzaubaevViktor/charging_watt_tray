package main

// Apple Silicon CPU/GPU power from the private IOReport framework, no sudo.
//
// We subscribe once to the "Energy Model" group and read the energy delta
// between successive samples; Go divides joules by elapsed wall-time to get
// watts. The private libIOReport.dylib has no SDK stub, so it is dlopen'd at
// runtime and its symbols resolved by name. All CoreFoundation traversal stays
// in C; Go keeps only the timestamp of the previous sample.

/*
#cgo LDFLAGS: -framework CoreFoundation
#include <dlfcn.h>
#include <ctype.h>
#include <string.h>
#include <CoreFoundation/CoreFoundation.h>

typedef CFMutableDictionaryRef (*CopyChannels_t)(CFStringRef, CFStringRef, uint64_t, uint64_t, uint64_t);
typedef void*           (*CreateSub_t)(void*, CFMutableDictionaryRef, CFMutableDictionaryRef*, uint64_t, CFTypeRef);
typedef CFDictionaryRef (*CreateSamples_t)(void*, CFMutableDictionaryRef, CFTypeRef);
typedef CFDictionaryRef (*CreateDelta_t)(CFDictionaryRef, CFDictionaryRef, CFTypeRef);
typedef CFStringRef     (*ChanName_t)(CFDictionaryRef);
typedef CFStringRef     (*ChanUnit_t)(CFDictionaryRef);
typedef int64_t         (*ChanValue_t)(CFDictionaryRef, int32_t);

static CopyChannels_t  pCopy;
static CreateSub_t     pSub;
static CreateSamples_t pSamp;
static CreateDelta_t   pDelta;
static ChanName_t      pName;
static ChanUnit_t      pUnit;
static ChanValue_t     pVal;

static void *gSub = NULL;
static CFMutableDictionaryRef gSubbed = NULL;
static CFDictionaryRef gPrev = NULL;
static CFStringRef gKeyChannels = NULL;

static int ioreport_init(void) {
	void *h = dlopen("/usr/lib/libIOReport.dylib", RTLD_NOW);
	if (!h) return -1;
	pCopy  = (CopyChannels_t)  dlsym(h, "IOReportCopyChannelsInGroup");
	pSub   = (CreateSub_t)     dlsym(h, "IOReportCreateSubscription");
	pSamp  = (CreateSamples_t) dlsym(h, "IOReportCreateSamples");
	pDelta = (CreateDelta_t)   dlsym(h, "IOReportCreateSamplesDelta");
	pName  = (ChanName_t)      dlsym(h, "IOReportChannelGetChannelName");
	pUnit  = (ChanUnit_t)      dlsym(h, "IOReportChannelGetUnitLabel");
	pVal   = (ChanValue_t)     dlsym(h, "IOReportSimpleGetIntegerValue");
	if (!pCopy || !pSub || !pSamp || !pDelta || !pName || !pUnit || !pVal) return -1;

	CFStringRef group = CFStringCreateWithCString(NULL, "Energy Model", kCFStringEncodingUTF8);
	CFMutableDictionaryRef channels = pCopy(group, NULL, 0, 0, 0);
	CFRelease(group);
	if (!channels) return -1;

	gSub = pSub(NULL, channels, &gSubbed, 0, NULL);
	if (!gSub) return -1;
	gKeyChannels = CFStringCreateWithCString(NULL, "IOReportChannels", kCFStringEncodingUTF8);
	gPrev = pSamp(gSub, gSubbed, NULL);
	return gPrev ? 0 : -1;
}

// energy-unit label -> joule scale (0 if unrecognised).
static double unit_scale(CFStringRef unit) {
	char buf[32];
	if (!unit || !CFStringGetCString(unit, buf, sizeof(buf), kCFStringEncodingUTF8)) return 0;
	for (char *p = buf; *p; ++p) *p = tolower((unsigned char)*p);
	if (!strcmp(buf, "j"))  return 1.0;
	if (!strcmp(buf, "mj")) return 1e-3;
	if (!strcmp(buf, "uj")) return 1e-6;
	if (!strcmp(buf, "nj")) return 1e-9;
	return 0;
}

// ioreport_sample sums CPU/GPU joules since the previous sample. Returns 0 ok.
static int ioreport_sample(double *cpu_j, double *gpu_j) {
	*cpu_j = 0;
	*gpu_j = 0;
	if (!gSub) return -1;
	CFDictionaryRef cur = pSamp(gSub, gSubbed, NULL);
	if (!cur) return -1;
	CFDictionaryRef delta = pDelta(gPrev, cur, NULL);
	CFRelease(gPrev);
	gPrev = cur;
	if (!delta) return -1;

	CFArrayRef arr = (CFArrayRef)CFDictionaryGetValue(delta, gKeyChannels);
	if (arr) {
		CFIndex n = CFArrayGetCount(arr);
		for (CFIndex i = 0; i < n; i++) {
			CFDictionaryRef ch = (CFDictionaryRef)CFArrayGetValueAtIndex(arr, i);
			char name[32];
			CFStringRef nm = pName(ch);
			if (!nm || !CFStringGetCString(nm, name, sizeof(name), kCFStringEncodingUTF8)) continue;
			double *bucket = NULL;
			if (!strcmp(name, "CPU Energy")) bucket = cpu_j;
			else if (!strcmp(name, "GPU Energy")) bucket = gpu_j;
			if (!bucket) continue;
			*bucket += (double)pVal(ch, 0) * unit_scale(pUnit(ch));
		}
	}
	CFRelease(delta);
	return 0;
}
*/
import "C"

import "time"

// EnergyModel yields CPU/GPU power from successive IOReport energy samples.
type EnergyModel struct {
	prevT time.Time
}

// openEnergyModel subscribes to the Energy Model group, or returns nil if the
// private IOReport framework is unavailable.
func openEnergyModel() *EnergyModel {
	if C.ioreport_init() != 0 {
		return nil
	}
	return &EnergyModel{prevT: time.Now()}
}

// Sample returns (cpuWatts, gpuWatts) averaged over the interval since the last
// call. Not safe for concurrent use — drive it from a single goroutine.
func (e *EnergyModel) Sample() (cpu, gpu float64) {
	if e == nil {
		return 0, 0
	}
	var cpuJ, gpuJ C.double
	rc := C.ioreport_sample(&cpuJ, &gpuJ)
	now := time.Now()
	dt := now.Sub(e.prevT).Seconds()
	e.prevT = now
	if rc != 0 || dt <= 0 {
		return 0, 0
	}
	return float64(cpuJ) / dt, float64(gpuJ) / dt
}
