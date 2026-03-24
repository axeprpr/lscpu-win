//go:build amd64

package main

import (
	"encoding/binary"
	"sort"
	"strings"
)

func cpuidSupported() bool {
	return true
}

func detectFlags() ([]string, string, string, bool) {
	if !cpuidSupported() {
		return nil, "", "", false
	}

	maxBasic, ebx0, ecx0, edx0 := cpuid(0, 0)
	vendorID := registersToString(ebx0, edx0, ecx0)

	_, _, ecx1, edx1 := cpuid(1, 0)
	maxExtended, _, _, _ := cpuid(0x80000000, 0)

	flags := map[string]bool{}
	addFeatureBits(flags, edx1, []bitFeature{
		{0, "fpu"},
		{1, "vme"},
		{2, "de"},
		{3, "pse"},
		{4, "tsc"},
		{5, "msr"},
		{6, "pae"},
		{7, "mce"},
		{8, "cx8"},
		{9, "apic"},
		{11, "sep"},
		{12, "mtrr"},
		{13, "pge"},
		{14, "mca"},
		{15, "cmov"},
		{16, "pat"},
		{17, "pse36"},
		{19, "clflush"},
		{23, "mmx"},
		{24, "fxsr"},
		{25, "sse"},
		{26, "sse2"},
		{28, "ht"},
	})
	addFeatureBits(flags, ecx1, []bitFeature{
		{0, "sse3"},
		{1, "pclmulqdq"},
		{3, "monitor"},
		{5, "vmx"},
		{9, "ssse3"},
		{12, "fma"},
		{13, "cx16"},
		{19, "sse4_1"},
		{20, "sse4_2"},
		{22, "movbe"},
		{23, "popcnt"},
		{25, "aes"},
		{26, "xsave"},
		{27, "osxsave"},
		{28, "avx"},
		{29, "f16c"},
		{30, "rdrand"},
	})

	if maxBasic >= 7 {
		_, ebx7, ecx7, edx7 := cpuid(7, 0)
		addFeatureBits(flags, ebx7, []bitFeature{
			{0, "fsgsbase"},
			{3, "bmi1"},
			{4, "hle"},
			{5, "avx2"},
			{8, "bmi2"},
			{9, "erms"},
			{10, "invpcid"},
			{11, "rtm"},
			{16, "avx512f"},
			{18, "rdseed"},
			{19, "adx"},
			{29, "sha_ni"},
		})
		addFeatureBits(flags, ecx7, []bitFeature{
			{0, "prefetchwt1"},
			{1, "avx512vbmi"},
			{8, "gfni"},
			{9, "vaes"},
			{10, "vpclmulqdq"},
		})
		addFeatureBits(flags, edx7, []bitFeature{
			{2, "avx512_4vnniw"},
			{3, "avx512_4fmaps"},
		})
	}

	if maxExtended >= 0x80000001 {
		_, _, ecx8, edx8 := cpuid(0x80000001, 0)
		addFeatureBits(flags, ecx8, []bitFeature{
			{0, "lahf_lm"},
			{5, "abm"},
			{6, "sse4a"},
			{11, "xop"},
			{16, "fma4"},
			{21, "tbm"},
			{29, "mwaitx"},
		})
		addFeatureBits(flags, edx8, []bitFeature{
			{11, "syscall"},
			{20, "nx"},
			{22, "mmxext"},
			{25, "fxsr_opt"},
			{26, "pdpe1gb"},
			{27, "rdtscp"},
			{29, "lm"},
			{30, "3dnowext"},
			{31, "3dnow"},
		})
	}

	virtualization := ""
	switch {
	case flags["vmx"]:
		virtualization = "VT-x"
	case flags["svm"]:
		virtualization = "AMD-V"
	}

	hypervisorDetected := bitSet(ecx1, 31)
	hypervisorVendor := ""
	if hypervisorDetected {
		_, ebx, ecx, edx := cpuid(0x40000000, 0)
		hypervisorVendor = normalizeVendor(registersToString(ebx, ecx, edx), vendorID)
	}

	ordered := make([]string, 0, len(flags))
	for flag := range flags {
		ordered = append(ordered, flag)
	}
	sort.Strings(ordered)
	return ordered, virtualization, hypervisorVendor, hypervisorDetected
}

type bitFeature struct {
	bit  uint
	name string
}

func addFeatureBits(target map[string]bool, value uint32, features []bitFeature) {
	for _, feature := range features {
		if bitSet(value, feature.bit) {
			target[feature.name] = true
		}
	}
}

func bitSet(value uint32, bit uint) bool {
	return value&(1<<bit) != 0
}

func registersToString(values ...uint32) string {
	buf := make([]byte, 0, len(values)*4)
	for _, value := range values {
		var raw [4]byte
		binary.LittleEndian.PutUint32(raw[:], value)
		buf = append(buf, raw[:]...)
	}
	return strings.TrimRight(string(buf), "\x00 ")
}

func normalizeVendor(vendor, fallback string) string {
	if vendor == "" {
		return fallback
	}
	return vendor
}

func cpuid(eaxArg, ecxArg uint32) (eax, ebx, ecx, edx uint32)
