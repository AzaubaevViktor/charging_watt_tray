package main

// AppleSMC reader for the real-time power sensors (watts, type 'flt '), no sudo.
//
//	PSTR - System Total power (current consumption)
//	PDTR - DC In Total power (what the adapter delivers; ~0 on battery)
//	PBAT - Battery power (into/out of the battery)
//
// The SMCKeyData_t protocol struct and the two-step keyinfo/read-bytes dance
// live in C; Go just opens the connection once and reads keys by fourcc.

/*
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation
#include <string.h>
#include <mach/mach.h>
#include <IOKit/IOKitLib.h>

typedef struct { uint8_t major, minor, build, reserved; uint16_t release; } SMCVersion;
typedef struct { uint16_t version, length; uint32_t cpuPLimit, gpuPLimit, memPLimit; } SMCPLimitData;
typedef struct { uint32_t dataSize, dataType; uint8_t dataAttributes; } SMCKeyInfoData;
typedef struct {
	uint32_t       key;
	SMCVersion     vers;
	SMCPLimitData  pLimitData;
	SMCKeyInfoData keyInfo;
	uint8_t        result, status, data8;
	uint32_t       data32;
	uint8_t        bytes[32];
} SMCKeyData_t;

enum { KERNEL_INDEX_SMC = 2, CMD_READ_BYTES = 5, CMD_READ_KEYINFO = 9 };

static io_connect_t gConn = 0;

static int smc_open(void) {
	io_service_t svc = IOServiceGetMatchingService(0, IOServiceMatching("AppleSMC"));
	if (!svc) return -1;
	kern_return_t rc = IOServiceOpen(svc, mach_task_self(), 0, &gConn);
	IOObjectRelease(svc);
	return rc == 0 ? 0 : -1;
}

static kern_return_t smc_call(SMCKeyData_t *in, SMCKeyData_t *out) {
	size_t outSize = sizeof(SMCKeyData_t);
	return IOConnectCallStructMethod(gConn, KERNEL_INDEX_SMC, in,
		sizeof(SMCKeyData_t), out, &outSize);
}

// smc_read_float reads a 4-byte 'flt ' key into *out. Returns 0 on success.
static int smc_read_float(uint32_t key, float *out) {
	SMCKeyData_t in, res;
	memset(&in, 0, sizeof(in));
	in.key = key;
	in.data8 = CMD_READ_KEYINFO;
	if (smc_call(&in, &res) != 0 || res.keyInfo.dataSize != 4) return -1;

	memset(&in, 0, sizeof(in));
	in.key = key;
	in.data8 = CMD_READ_BYTES;
	in.keyInfo.dataSize = 4;
	if (smc_call(&in, &res) != 0) return -1;
	memcpy(out, res.bytes, 4); // little-endian float
	return 0;
}
*/
import "C"

// SMC is an open connection to the AppleSMC service.
type SMC struct{}

// openSMC connects to AppleSMC, or returns nil if it is unavailable.
func openSMC() *SMC {
	if C.smc_open() != 0 {
		return nil
	}
	return &SMC{}
}

// ReadFloat reads an SMC 'flt ' key (a 4-char fourcc). Returns (value, true) or
// (0, false) on any failure.
func (s *SMC) ReadFloat(key string) (float64, bool) {
	if s == nil || len(key) != 4 {
		return 0, false
	}
	fourcc := uint32(key[0])<<24 | uint32(key[1])<<16 | uint32(key[2])<<8 | uint32(key[3])
	var out C.float
	if C.smc_read_float(C.uint32_t(fourcc), &out) != 0 {
		return 0, false
	}
	return float64(out), true
}
