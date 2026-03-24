package main

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

const (
	relationProcessorCore    = 0
	relationNumaNode         = 1
	relationCache            = 2
	relationProcessorPackage = 3
	relationAll              = 0xffff

	cacheUnified = 0
	cacheData    = 1
	cacheInstr   = 2
)

type logicalProcessorInformationExHeader struct {
	Relationship uint32
	Size         uint32
}

type cacheRelationship struct {
	Level         byte
	Associativity byte
	LineSize      uint16
	CacheSize     uint32
	Type          uint32
	Reserved      [20]byte
}

type cacheKey struct {
	level int
	kind  string
}

type cpuInfo struct {
	Architecture   string
	LogicalCPUs    int
	VendorID       string
	ModelName      string
	Family         string
	Model          string
	Stepping       string
	SocketCount    int
	CoreCount      int
	NumaNodeCount  int
	MaxMHz         string
	Caches         map[cacheKey]uint64
}

func main() {
	info, err := collectCPUInfo()
	if err != nil {
		fmt.Printf("failed to collect CPU info: %v\n", err)
		return
	}

	printRow("Architecture", info.Architecture)
	printRow("CPU(s)", strconv.Itoa(info.LogicalCPUs))
	printRow("Vendor ID", fallback(info.VendorID, "unknown"))
	printRow("Model name", fallback(info.ModelName, "unknown"))
	printRow("CPU family", fallback(info.Family, "unknown"))
	printRow("Model", fallback(info.Model, "unknown"))
	printRow("Stepping", fallback(info.Stepping, "unknown"))
	printRow("Socket(s)", printableCount(info.SocketCount))
	printRow("Core(s)", printableCount(info.CoreCount))
	printRow("Core(s) per socket", divideLabel(info.CoreCount, info.SocketCount))
	printRow("Thread(s) per core", divideLabel(info.LogicalCPUs, info.CoreCount))
	printRow("NUMA node(s)", printableCount(info.NumaNodeCount))
	if info.MaxMHz != "" {
		printRow("CPU max MHz", info.MaxMHz)
	}

	for _, label := range orderedCacheLabels(info.Caches) {
		printRow(label, formatBytes(info.Caches[parseCacheLabel(label)]))
	}
}

func collectCPUInfo() (*cpuInfo, error) {
	info := &cpuInfo{
		Architecture: runtime.GOARCH,
		LogicalCPUs:  runtime.NumCPU(),
		Caches:       map[cacheKey]uint64{},
	}

	if err := fillTopology(info); err != nil {
		return nil, err
	}

	registryData := queryCPURegistry()
	info.VendorID = registryData["VendorIdentifier"]
	info.ModelName = registryData["ProcessorNameString"]
	info.MaxMHz = normalizeMHz(registryData["~MHz"])

	family, model, stepping := parseIdentifier(registryData["Identifier"])
	info.Family = family
	info.Model = model
	info.Stepping = stepping

	return info, nil
}

func fillTopology(info *cpuInfo) error {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetLogicalProcessorInformationEx")
	if err := proc.Find(); err != nil {
		return err
	}

	var needed uint32
	r1, _, callErr := proc.Call(uintptr(relationAll), 0, uintptr(unsafe.Pointer(&needed)))
	if r1 != 0 {
		return errors.New("unexpected success with zero buffer")
	}
	if needed == 0 {
		return callErr
	}

	buf := make([]byte, needed)
	r1, _, callErr = proc.Call(
		uintptr(relationAll),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&needed)),
	)
	if r1 == 0 {
		return callErr
	}

	for offset := uint32(0); offset < needed; {
		header := (*logicalProcessorInformationExHeader)(unsafe.Pointer(&buf[offset]))
		switch header.Relationship {
		case relationProcessorCore:
			info.CoreCount++
		case relationProcessorPackage:
			info.SocketCount++
		case relationNumaNode:
			info.NumaNodeCount++
		case relationCache:
			rel := (*cacheRelationship)(unsafe.Pointer(&buf[offset+8]))
			key := cacheKey{level: int(rel.Level), kind: cacheKind(rel.Type)}
			info.Caches[key] += uint64(rel.CacheSize)
		}
		offset += header.Size
	}

	return nil
}

func queryCPURegistry() map[string]string {
	keys := map[string]string{}
	for _, valueName := range []string{"VendorIdentifier", "ProcessorNameString", "Identifier", "~MHz"} {
		value, err := regQuery(`HKLM\HARDWARE\DESCRIPTION\System\CentralProcessor\0`, valueName)
		if err == nil {
			keys[valueName] = value
		}
	}
	return keys
}

func regQuery(path, valueName string) (string, error) {
	cmd := exec.Command("reg", "query", path, "/v", valueName)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}

	for _, line := range strings.Split(stdout.String(), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[0] != valueName {
			continue
		}
		return strings.Join(fields[2:], " "), nil
	}

	return "", fmt.Errorf("registry value %s not found", valueName)
}

func parseIdentifier(identifier string) (string, string, string) {
	if identifier == "" {
		return "", "", ""
	}

	family := extractRegex(identifier, `Family\s+(\d+)`)
	model := extractRegex(identifier, `Model\s+(\d+)`)
	stepping := extractRegex(identifier, `Stepping\s+(\d+)`)
	return family, model, stepping
}

func normalizeMHz(value string) string {
	if value == "" {
		return ""
	}

	parsed, err := strconv.ParseInt(value, 0, 64)
	if err != nil {
		return value
	}
	return strconv.FormatInt(parsed, 10)
}

func extractRegex(value, pattern string) string {
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(value)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

func cacheKind(kind uint32) string {
	switch kind {
	case cacheData:
		return "d"
	case cacheInstr:
		return "i"
	default:
		return ""
	}
}

func orderedCacheLabels(caches map[cacheKey]uint64) []string {
	keys := make([]cacheKey, 0, len(caches))
	for key := range caches {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].level != keys[j].level {
			return keys[i].level < keys[j].level
		}
		return keys[i].kind < keys[j].kind
	})

	labels := make([]string, 0, len(keys))
	for _, key := range keys {
		labels = append(labels, formatCacheLabel(key))
	}
	return labels
}

func formatCacheLabel(key cacheKey) string {
	prefix := fmt.Sprintf("L%d", key.level)
	switch key.kind {
	case "d":
		return prefix + "d cache"
	case "i":
		return prefix + "i cache"
	default:
		return prefix + " cache"
	}
}

func parseCacheLabel(label string) cacheKey {
	re := regexp.MustCompile(`L(\d)([di]?) cache`)
	match := re.FindStringSubmatch(label)
	if len(match) != 3 {
		return cacheKey{}
	}
	level, _ := strconv.Atoi(match[1])
	return cacheKey{level: level, kind: match[2]}
}

func formatBytes(size uint64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := uint64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}

func divideLabel(total, divisor int) string {
	if total == 0 || divisor == 0 {
		return "unknown"
	}
	if total%divisor != 0 {
		return "mixed"
	}
	return strconv.Itoa(total / divisor)
}

func fallback(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func printableCount(value int) string {
	if value <= 0 {
		return "unknown"
	}
	return strconv.Itoa(value)
}

func printRow(label, value string) {
	fmt.Printf("%-20s %s\n", label+":", value)
}
