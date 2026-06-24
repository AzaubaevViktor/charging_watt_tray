"""Read Apple Silicon CPU/GPU power from the private IOReport framework.

No sudo required. Subscribes once to the "Energy Model" group and computes
power from the energy delta between successive samples (one per call), so it
never blocks. Used to anchor per-process energy-impact to real compute watts.
"""

import ctypes
import time
from ctypes import (POINTER, c_char_p, c_int32, c_int64, c_long, c_uint64,
                    c_void_p, byref)

_cf = ctypes.CDLL("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation")
_ir = ctypes.CDLL("/usr/lib/libIOReport.dylib")
_UTF8 = 0x08000100

_cf.CFStringCreateWithCString.restype = c_void_p
_cf.CFStringCreateWithCString.argtypes = [c_void_p, c_char_p, c_uint64]
_cf.CFStringGetCStringPtr.restype = c_char_p
_cf.CFStringGetCStringPtr.argtypes = [c_void_p, c_uint64]
_cf.CFStringGetCString.restype = ctypes.c_bool
_cf.CFStringGetCString.argtypes = [c_void_p, c_char_p, c_long, c_uint64]
_cf.CFDictionaryGetValue.restype = c_void_p
_cf.CFDictionaryGetValue.argtypes = [c_void_p, c_void_p]
_cf.CFArrayGetCount.restype = c_long
_cf.CFArrayGetCount.argtypes = [c_void_p]
_cf.CFArrayGetValueAtIndex.restype = c_void_p
_cf.CFArrayGetValueAtIndex.argtypes = [c_void_p, c_long]
_cf.CFRelease.argtypes = [c_void_p]

for _name, _res, _args in [
    ("IOReportCopyChannelsInGroup", c_void_p, [c_void_p, c_void_p, c_uint64, c_uint64, c_uint64]),
    ("IOReportCreateSubscription", c_void_p, [c_void_p, c_void_p, POINTER(c_void_p), c_uint64, c_void_p]),
    ("IOReportCreateSamples", c_void_p, [c_void_p, c_void_p, c_void_p]),
    ("IOReportCreateSamplesDelta", c_void_p, [c_void_p, c_void_p, c_void_p]),
    ("IOReportChannelGetChannelName", c_void_p, [c_void_p]),
    ("IOReportChannelGetUnitLabel", c_void_p, [c_void_p]),
    ("IOReportSimpleGetIntegerValue", c_int64, [c_void_p, c_int32]),
]:
    _fn = getattr(_ir, _name)
    _fn.restype, _fn.argtypes = _res, _args

_JOULES = {"j": 1.0, "mj": 1e-3, "uj": 1e-6, "nj": 1e-9}


def _cfstr(s):
    return _cf.CFStringCreateWithCString(None, s.encode(), _UTF8)


def _pystr(ref):
    if not ref:
        return None
    ptr = _cf.CFStringGetCStringPtr(ref, _UTF8)
    if ptr:
        return ptr.decode()
    buf = ctypes.create_string_buffer(128)
    return buf.value.decode() if _cf.CFStringGetCString(ref, buf, 128, _UTF8) else None


class EnergyModel:
    """Subscribes to the Energy Model group; .sample() returns (cpu_w, gpu_w)."""

    # Aggregate channel names we care about (totals, not per-core blocks).
    _WANTED = {"CPU Energy": "cpu", "GPU Energy": "gpu"}

    def __init__(self):
        channels = _ir.IOReportCopyChannelsInGroup(_cfstr("Energy Model"), None, 0, 0, 0)
        if not channels:
            raise OSError("IOReport: no Energy Model channels")
        self._channels = channels
        subbed = c_void_p()
        self._sub = _ir.IOReportCreateSubscription(None, channels, byref(subbed), 0, None)
        if not self._sub:
            raise OSError("IOReport: subscription failed")
        self._subbed = subbed
        self._key_channels = _cfstr("IOReportChannels")
        self._prev = _ir.IOReportCreateSamples(self._sub, self._subbed, None)
        self._prev_t = time.monotonic()

    def sample(self):
        cur = _ir.IOReportCreateSamples(self._sub, self._subbed, None)
        now = time.monotonic()
        dt = now - self._prev_t
        delta = _ir.IOReportCreateSamplesDelta(self._prev, cur, None)
        _cf.CFRelease(self._prev)
        self._prev = cur
        self._prev_t = now

        joules = {"cpu": 0.0, "gpu": 0.0}
        arr = _cf.CFDictionaryGetValue(delta, self._key_channels)
        for i in range(_cf.CFArrayGetCount(arr)):
            ch = _cf.CFArrayGetValueAtIndex(arr, i)
            bucket = self._WANTED.get(_pystr(_ir.IOReportChannelGetChannelName(ch)))
            if bucket is None:
                continue
            unit = (_pystr(_ir.IOReportChannelGetUnitLabel(ch)) or "").lower()
            value = _ir.IOReportSimpleGetIntegerValue(ch, 0)
            joules[bucket] += value * _JOULES.get(unit, 0.0)
        _cf.CFRelease(delta)

        if dt <= 0:
            return 0.0, 0.0
        return joules["cpu"] / dt, joules["gpu"] / dt
