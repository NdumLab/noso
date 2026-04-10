package registry

import (
	"strings"
	"testing"

	"github.com/noso-dev/noso/internal/evidence"
	"github.com/noso-dev/noso/pkg/models"
)

func TestCPUInfoIntent(t *testing.T) {
	response, err := Resolve("show cpu info", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_cpu_info" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestMemoryInfoIntent(t *testing.T) {
	response, err := Resolve("show memory usage", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_memory_info" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestBlockDevicesIntent(t *testing.T) {
	response, err := Resolve("list disks", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_block_devices" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestSystemHardwareIntent(t *testing.T) {
	response, err := Resolve("show hardware info", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_system_hardware" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestDiskHealthIntent(t *testing.T) {
	response, err := Resolve("check disk health for /dev/nvme0n1", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_disk_health" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if !strings.Contains(response.Command, "/dev/nvme0n1") {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestGPUInfoIntent(t *testing.T) {
	response, err := Resolve("show gpu status", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_gpu_info" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestIPMIInfoIntent(t *testing.T) {
	response, err := Resolve("show ipmi info", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_ipmi_info" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}
