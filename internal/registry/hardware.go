package registry

import (
	"fmt"
	"strings"

	"github.com/NdumLab/noso/internal/evidence"
	"github.com/NdumLab/noso/internal/safety"
	"github.com/NdumLab/noso/pkg/models"
)

func cpuInfoIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("lscpu")
	command := "lscpu"
	response := models.Response{
		IntentID:       "inspect_cpu_info",
		Command:        command,
		Explanation:    "Shows CPU architecture, model, socket, core, thread, and virtualization details from the local host.",
		ExpectedOutput: "A CPU summary with architecture, model name, CPU count, sockets, cores per socket, threads, caches, and virtualization flags.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "lscpu")
	return response, nil
}

func memoryInfoIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("free")
	command := "free -h"
	response := models.Response{
		IntentID:       "inspect_memory_info",
		Command:        command,
		Explanation:    "Shows memory and swap totals, usage, and available memory in human-readable units.",
		ExpectedOutput: "A table with total, used, free, shared, buff or cache, available memory, and swap usage.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "free")
	return response, nil
}

func blockDevicesIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("lsblk")
	command := "lsblk -o NAME,SIZE,TYPE,FSTYPE,MOUNTPOINT"
	response := models.Response{
		IntentID:       "inspect_block_devices",
		Command:        command,
		Explanation:    "Lists block devices, their sizes, types, filesystems, and mountpoints so you can inspect disk layout safely.",
		ExpectedOutput: "A table of disks, partitions, and logical volumes with size, type, filesystem, and mountpoint columns.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "lsblk")
	return response, nil
}

func systemHardwareIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("dmidecode")
	command := "dmidecode -t system"
	response := models.Response{
		IntentID:       "inspect_system_hardware",
		Command:        command,
		Explanation:    "Shows vendor, product, serial, UUID, and other SMBIOS system identity details. Some systems require elevated privileges to read it.",
		ExpectedOutput: "A system information block with vendor, product name, version, serial number, UUID, and wake-up type.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "dmidecode")
	return response, nil
}

func diskHealthIntent(query string, collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("smartctl")
	device := extractDevicePath(query)
	command := fmt.Sprintf("smartctl -H %s", device)
	response := models.Response{
		IntentID:       "inspect_disk_health",
		Command:        command,
		Explanation:    fmt.Sprintf("Shows the overall SMART health assessment for %s without modifying the device.", device),
		ExpectedOutput: "A SMART status line reporting whether the device passed the overall health self-assessment.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "smartctl")
	return response, nil
}

func ipmiInfoIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("ipmitool")
	command := "ipmitool mc info"
	response := models.Response{
		IntentID:       "inspect_ipmi_info",
		Command:        command,
		Explanation:    "Shows basic BMC or IPMI management-controller information such as firmware revision and manufacturer ID.",
		ExpectedOutput: "A controller information block with device ID, firmware revision, manufacturer, and supported IPMI version.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "ipmitool")
	return response, nil
}

func gpuInfoIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("nvidia-smi")
	command := "nvidia-smi"
	response := models.Response{
		IntentID:       "inspect_gpu_info",
		Command:        command,
		Explanation:    "Shows NVIDIA GPU inventory, driver state, utilization, memory usage, and active compute processes.",
		ExpectedOutput: "A GPU summary table with model names, driver and CUDA versions, temperature, power draw, memory usage, utilization, and active processes.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "nvidia-smi")
	return response, nil
}

func extractDevicePath(query string) string {
	for _, field := range strings.Fields(query) {
		trimmed := strings.Trim(field, "`\"'")
		if strings.HasPrefix(trimmed, "/dev/") {
			return trimmed
		}
	}
	return "/dev/sda"
}
