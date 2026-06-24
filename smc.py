"""Minimal AppleSMC reader (real-time power sensors), no sudo required.

Power keys (watts, type 'flt ') confirmed on Apple Silicon:
  PSTR - System Total power (current consumption)
  PDTR - DC In Total power (what the adapter delivers; ~0 on battery)
  PBAT - Battery power (into/out of the battery)
"""

import ctypes
import struct
from ctypes import (POINTER, Structure, byref, c_char_p, c_size_t, c_uint8,
                    c_uint16, c_uint32, c_void_p, sizeof)

_KERNEL_INDEX_SMC = 2
_CMD_READ_BYTES = 5
_CMD_READ_KEYINFO = 9


class _Vers(Structure):
    _fields_ = [("major", c_uint8), ("minor", c_uint8), ("build", c_uint8),
                ("reserved", c_uint8), ("release", c_uint16)]


class _PLimit(Structure):
    _fields_ = [("version", c_uint16), ("length", c_uint16), ("cpuPLimit", c_uint32),
                ("gpuPLimit", c_uint32), ("memPLimit", c_uint32)]


class _KeyInfo(Structure):
    _fields_ = [("dataSize", c_uint32), ("dataType", c_uint32), ("dataAttributes", c_uint8)]


class _KeyData(Structure):
    _fields_ = [("key", c_uint32), ("vers", _Vers), ("pLimitData", _PLimit),
                ("keyInfo", _KeyInfo), ("result", c_uint8), ("status", c_uint8),
                ("data8", c_uint8), ("data32", c_uint32), ("bytes", c_uint8 * 32)]


def _fourcc(s):
    return struct.unpack(">I", s.encode())[0]


class SMC:
    def __init__(self):
        self._iokit = ctypes.CDLL("/System/Library/Frameworks/IOKit.framework/IOKit")
        self._libc = ctypes.CDLL("/usr/lib/libc.dylib")
        io = self._iokit
        io.IOServiceMatching.restype = c_void_p
        io.IOServiceMatching.argtypes = [c_char_p]
        io.IOServiceGetMatchingService.restype = c_uint32
        io.IOServiceGetMatchingService.argtypes = [c_uint32, c_void_p]
        io.IOServiceOpen.restype = c_uint32
        io.IOServiceOpen.argtypes = [c_uint32, c_uint32, c_uint32, POINTER(c_uint32)]
        io.IOConnectCallStructMethod.restype = c_uint32
        io.IOConnectCallStructMethod.argtypes = [
            c_uint32, c_uint32, c_void_p, c_size_t, c_void_p, POINTER(c_size_t)]
        io.IOServiceClose.restype = c_uint32
        io.IOServiceClose.argtypes = [c_uint32]
        self._libc.mach_task_self.restype = c_uint32

        device = io.IOServiceGetMatchingService(0, io.IOServiceMatching(b"AppleSMC"))
        if not device:
            raise OSError("AppleSMC service not found")
        conn = c_uint32(0)
        if io.IOServiceOpen(device, self._libc.mach_task_self(), 0, byref(conn)) != 0:
            raise OSError("failed to open AppleSMC")
        self._conn = conn.value

    def _call(self, inp):
        out = _KeyData()
        out_size = c_size_t(sizeof(_KeyData))
        rc = self._iokit.IOConnectCallStructMethod(
            self._conn, _KERNEL_INDEX_SMC, byref(inp), sizeof(_KeyData),
            byref(out), byref(out_size))
        return rc, out

    def read_float(self, key):
        """Read an SMC 'flt ' key. Returns float, or None on failure."""
        inp = _KeyData()
        inp.key = _fourcc(key)
        inp.data8 = _CMD_READ_KEYINFO
        rc, info = self._call(inp)
        if rc != 0 or info.keyInfo.dataSize != 4:
            return None
        inp = _KeyData()
        inp.key = _fourcc(key)
        inp.data8 = _CMD_READ_BYTES
        inp.keyInfo.dataSize = info.keyInfo.dataSize
        rc, out = self._call(inp)
        if rc != 0:
            return None
        return struct.unpack("<f", bytes(out.bytes[:4]))[0]

    def close(self):
        if getattr(self, "_conn", None):
            self._iokit.IOServiceClose(self._conn)
            self._conn = None
