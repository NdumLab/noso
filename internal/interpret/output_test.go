package interpret

import (
	"strings"
	"testing"
)

func TestInterpretDFDetectsHotFilesystem(t *testing.T) {
	text := "Filesystem Size Used Avail Use% Mounted on\n/dev/mapper/root 100G 95G 5G 95% /\n"
	response, err := Output("df -h", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if response.IntentID != "interpret_df" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if response.Confidence != "High" {
		t.Fatalf("Confidence = %q", response.Confidence)
	}
	if len(response.NextSteps) == 0 {
		t.Fatal("expected follow-up guidance")
	}
}

func TestInterpretFreeDetectsMemoryPressure(t *testing.T) {
	text := "               total        used        free      shared  buff/cache   available\nMem:            31Gi        27Gi       512Mi       1.0Gi       3.0Gi       2.0Gi\nSwap:          8.0Gi       2.0Gi       6.0Gi\n"
	response, err := Output("free -h", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if response.IntentID != "interpret_free" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if !strings.Contains(strings.ToLower(response.Explanation), "memory pressure") {
		t.Fatalf("Explanation = %q", response.Explanation)
	}
}

func TestInterpretSystemctlFailedUnit(t *testing.T) {
	text := "Active: failed (Result: exit-code) since Fri 2026-04-10 12:00:00 UTC; 3min ago\n"
	response, err := Output("systemctl status nginx", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if response.IntentID != "interpret_systemctl_status" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if !strings.Contains(strings.ToLower(response.Explanation), "failed state") {
		t.Fatalf("Explanation = %q", response.Explanation)
	}
}

func TestInterpretSystemctlMissingUnit(t *testing.T) {
	text := "Unit worker2.service could not be found.\n"
	response, err := Output("systemctl status worker2", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if response.Confidence != "High" {
		t.Fatalf("Confidence = %q", response.Confidence)
	}
	if !strings.Contains(strings.ToLower(response.Explanation), "could not be found") {
		t.Fatalf("Explanation = %q", response.Explanation)
	}
}

func TestInterpretKubectlDetectsUnhealthyPods(t *testing.T) {
	text := "NAME READY STATUS RESTARTS AGE\nweb-7c5c 0/1 CrashLoopBackOff 8 10m\napi-9b1d 1/1 Running 0 10m\n"
	response, err := Output("kubectl get pods -n prod", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if response.IntentID != "interpret_kubectl_get_pods" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if len(response.NextSteps) == 0 {
		t.Fatal("expected pod follow-up guidance")
	}
}

func TestInterpretKubectlDescribePodDetectsCrashLoop(t *testing.T) {
	text := "State:          Waiting\n  Reason:       CrashLoopBackOff\nEvents:\n  Warning  BackOff  kubelet  Back-off restarting failed container\n"
	response, err := Output("kubectl describe pod web-7c5c", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if response.IntentID != "interpret_kubectl_describe_pod" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if response.Confidence != "High" {
		t.Fatalf("Confidence = %q", response.Confidence)
	}
}

func TestInterpretRuntimePSDetectsExitedContainer(t *testing.T) {
	text := "CONTAINER ID  IMAGE   COMMAND   CREATED   STATUS                     NAMES\nabc123        app     app       1m ago    Exited (1) 10 seconds ago  worker2\n"
	response, err := Output("podman ps -a", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if response.IntentID != "interpret_runtime_ps" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if response.Confidence != "High" {
		t.Fatalf("Confidence = %q", response.Confidence)
	}
}

func TestInterpretRuntimeLogsDetectsErrors(t *testing.T) {
	text := "error: failed to bind socket: permission denied\n"
	response, err := Output("podman logs --tail 100 worker2", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if response.IntentID != "interpret_runtime_logs" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if response.Confidence != "High" {
		t.Fatalf("Confidence = %q", response.Confidence)
	}
}

func TestInterpretUnsupportedCommand(t *testing.T) {
	response, err := Output("uname -a", "Linux")
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if response.IntentID != "interpret_unsupported_output" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestInterpretJournalctlDetectsErrors(t *testing.T) {
	text := "Apr 10 12:00:01 host nginx[1234]: [error] failed to bind socket: permission denied\n"
	response, err := Output("journalctl -u nginx -n 50 --no-pager", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if response.IntentID != "interpret_journalctl" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if response.Confidence != "High" {
		t.Fatalf("Confidence = %q, want High when errors detected", response.Confidence)
	}
	if len(response.NextSteps) == 0 {
		t.Fatal("expected next-step guidance for error signals")
	}
}

func TestInterpretJournalctlCleanLog(t *testing.T) {
	text := "Apr 10 12:00:01 host nginx[1234]: worker process started\nApr 10 12:00:01 host nginx[1234]: listening on :80\n"
	response, err := Output("journalctl -u nginx -n 50 --no-pager", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if response.Confidence != "Medium" {
		t.Fatalf("Confidence = %q, want Medium for clean log", response.Confidence)
	}
}

func TestInterpretJournalctlOOMActivity(t *testing.T) {
	text := "kernel: Out of memory: Killed process 4567 (java) total-vm:2048000kB\n"
	response, err := Output("journalctl -k", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	combined := strings.Join(response.NextSteps, " ")
	if !strings.Contains(combined, "free -h") {
		t.Errorf("expected free -h in next steps for OOM, got: %s", combined)
	}
}

func TestInterpretPSDetectsHighUsage(t *testing.T) {
	text := "USER       PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND\n" +
		"root      1234 85.0  5.2 120000 50000 ?        R    00:00   5:00 myjob\n"
	response, err := Output("ps aux --sort=-%cpu | head", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if response.IntentID != "interpret_ps" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if response.Confidence != "High" {
		t.Fatalf("Confidence = %q", response.Confidence)
	}
	if !strings.Contains(response.Explanation, "myjob") {
		t.Errorf("Explanation should mention the high-CPU process: %s", response.Explanation)
	}
}

func TestInterpretPSNoIssues(t *testing.T) {
	text := "USER       PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND\n" +
		"root         1  0.0  0.1   1234   500 ?        Ss   00:00   0:01 init\n"
	response, err := Output("ps aux", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if response.Confidence != "High" {
		t.Fatalf("Confidence = %q", response.Confidence)
	}
}

func TestInterpretIPAddrDetectsDownInterface(t *testing.T) {
	text := "2: eth0: <BROADCAST,MULTICAST> mtu 1500 qdisc noop state DOWN group default\n    link/ether 00:00:00:00:00:00 brd ff:ff:ff:ff:ff:ff\n"
	response, err := Output("ip addr show", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if response.IntentID != "interpret_ip_addr" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if response.Confidence != "High" {
		t.Fatalf("Confidence = %q", response.Confidence)
	}
	if !strings.Contains(response.Explanation, "DOWN") {
		t.Errorf("Explanation should mention DOWN interface: %s", response.Explanation)
	}
}

func TestInterpretIPAddrHealthyInterfaces(t *testing.T) {
	text := "1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 state UNKNOWN\n    inet 127.0.0.1/8 scope host lo\n" +
		"2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 state UP\n    inet 10.0.0.5/24 brd 10.0.0.255 scope global eth0\n"
	response, err := Output("ip addr", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if !strings.Contains(response.Explanation, "UP") {
		t.Errorf("Explanation should confirm UP state: %s", response.Explanation)
	}
}

func TestParseHumanBytesIECUnits(t *testing.T) {
	value, ok := parseHumanBytes("2.0Gi")
	if !ok {
		t.Fatal("parseHumanBytes() should parse Gi values")
	}
	if value == 0 {
		t.Fatal("parsed value should be non-zero")
	}
}
