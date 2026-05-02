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

func TestInterpretKubectlGetEventsExtractsPodAndContainer(t *testing.T) {
	text := "LAST SEEN TYPE REASON OBJECT MESSAGE\n10s Warning BackOff pod/web-7c5c Back-off restarting failed container api in pod web-7c5c_prod(1234)\n"
	response, err := Output("kubectl get events -n prod --sort-by=.metadata.creationTimestamp", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if response.IntentID != "interpret_kubectl_get_events" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if response.ContainerHint != "api" {
		t.Fatalf("ContainerHint = %q, want api", response.ContainerHint)
	}
	combined := strings.Join(response.NextSteps, " ")
	if !strings.Contains(combined, "kubectl describe pod -n prod web-7c5c") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
	if !strings.Contains(combined, "kubectl logs -n prod web-7c5c -c api --previous") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
}

func TestInterpretKubectlGetEventsImagePullBackoff(t *testing.T) {
	text := "LAST SEEN TYPE REASON OBJECT MESSAGE\n12s Warning Failed pod/api-6d8f Failed to pull image \"ghcr.io/example/app:bad\": rpc error\n11s Warning ImagePullBackOff pod/api-6d8f Back-off pulling image \"ghcr.io/example/app:bad\"\n"
	response, err := Output("kubectl get events -n prod --sort-by=.metadata.creationTimestamp", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	combined := strings.Join(response.NextSteps, " ")
	if !strings.Contains(combined, "kubectl describe pod -n prod api-6d8f") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
	if !strings.Contains(combined, "imagePullSecrets") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
	if !strings.Contains(combined, "dig +short ghcr.io") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
	if strings.Contains(combined, "kubectl logs -n prod api-6d8f") {
		t.Fatalf("NextSteps should not prioritize logs for image-pull failures: %#v", response.NextSteps)
	}
}

func TestInterpretKubectlGetEventsImagePullRegistryPort(t *testing.T) {
	text := "LAST SEEN TYPE REASON OBJECT MESSAGE\n12s Warning Failed pod/api-6d8f Failed to pull image \"registry.internal:5000/team/app:bad\": rpc error\n"
	response, err := Output("kubectl get events -n prod --sort-by=.metadata.creationTimestamp", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	combined := strings.Join(response.NextSteps, " ")
	if !strings.Contains(combined, "dig +short registry.internal") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
	if !strings.Contains(combined, "nc -vz registry.internal 5000") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
}

func TestInterpretKubectlGetEventsSchedulingFailure(t *testing.T) {
	text := "LAST SEEN TYPE REASON OBJECT MESSAGE\n20s Warning FailedScheduling pod/web-7c5c 0/3 nodes are available: 3 Insufficient memory.\n"
	response, err := Output("kubectl get events -n prod --sort-by=.metadata.creationTimestamp", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	combined := strings.Join(response.NextSteps, " ")
	if !strings.Contains(combined, "kubectl describe pod -n prod web-7c5c") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
	if !strings.Contains(combined, "taint") && !strings.Contains(combined, "capacity") && !strings.Contains(combined, "affinity") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
	if strings.Contains(combined, "kubectl logs -n prod web-7c5c") {
		t.Fatalf("NextSteps should not prioritize logs for scheduling failures: %#v", response.NextSteps)
	}
	if !strings.Contains(combined, "allocatable memory") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
}

func TestInterpretKubectlGetEventsSchedulingTaintFailure(t *testing.T) {
	text := "LAST SEEN TYPE REASON OBJECT MESSAGE\n20s Warning FailedScheduling pod/web-7c5c 0/3 nodes are available: 1 node(s) had untolerated taint {dedicated: gpu}.\n"
	response, err := Output("kubectl get events -n prod --sort-by=.metadata.creationTimestamp", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	combined := strings.Join(response.NextSteps, " ")
	if !strings.Contains(combined, "tolerations") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
}

func TestInterpretKubectlGetEventsSchedulingAffinityFailure(t *testing.T) {
	text := "LAST SEEN TYPE REASON OBJECT MESSAGE\n20s Warning FailedScheduling pod/web-7c5c 0/3 nodes are available: 3 node(s) didn't match Pod's node affinity.\n"
	response, err := Output("kubectl get events -n prod --sort-by=.metadata.creationTimestamp", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	combined := strings.Join(response.NextSteps, " ")
	if !strings.Contains(combined, "node affinity") && !strings.Contains(combined, "selector") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
}

func TestInterpretKubectlGetEventsSchedulingNamedNodeFailure(t *testing.T) {
	text := "LAST SEEN TYPE REASON OBJECT MESSAGE\n20s Warning FailedScheduling pod/web-7c5c node ip-10-0-1-12 had untolerated taint {dedicated: gpu}.\n"
	response, err := Output("kubectl get events -n prod --sort-by=.metadata.creationTimestamp", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	combined := strings.Join(response.NextSteps, " ")
	if !strings.Contains(combined, "kubectl describe node ip-10-0-1-12") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
}

func TestInterpretKubectlGetEventsMountFailurePVC(t *testing.T) {
	text := "LAST SEEN TYPE REASON OBJECT MESSAGE\n20s Warning FailedMount pod/web-7c5c Unable to attach or mount volumes: unmounted volumes=[data], unattached volumes=[data]: timed out waiting for the condition, persistentvolumeclaim \"web-data\" not found\n"
	response, err := Output("kubectl get events -n prod --sort-by=.metadata.creationTimestamp", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	combined := strings.Join(response.NextSteps, " ")
	if !strings.Contains(combined, "kubectl describe pvc -n prod web-data") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
}

func TestInterpretKubectlGetEventsMountFailureSecret(t *testing.T) {
	text := "LAST SEEN TYPE REASON OBJECT MESSAGE\n20s Warning FailedMount pod/web-7c5c MountVolume.SetUp failed for volume \"tls\": secret \"web-tls\" not found\n"
	response, err := Output("kubectl get events -n prod --sort-by=.metadata.creationTimestamp", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	combined := strings.Join(response.NextSteps, " ")
	if !strings.Contains(combined, "kubectl describe secret -n prod web-tls") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
}

func TestInterpretKubectlGetEventsMountFailureConfigMap(t *testing.T) {
	text := "LAST SEEN TYPE REASON OBJECT MESSAGE\n20s Warning FailedMount pod/web-7c5c MountVolume.SetUp failed for volume \"cfg\": configmap \"web-config\" not found\n"
	response, err := Output("kubectl get events -n prod --sort-by=.metadata.creationTimestamp", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	combined := strings.Join(response.NextSteps, " ")
	if !strings.Contains(combined, "kubectl describe configmap -n prod web-config") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
}

func TestInterpretKubectlGetEventsDeploymentOwnerFollowUp(t *testing.T) {
	text := "LAST SEEN TYPE REASON OBJECT MESSAGE\n20s Warning BackOff pod/web-7c5c Back-off restarting failed container api in pod web-7c5c_prod(1234), controlled by deployment/web\n"
	response, err := Output("kubectl get events -n prod --sort-by=.metadata.creationTimestamp", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	combined := strings.Join(response.NextSteps, " ")
	if !strings.Contains(combined, "kubectl describe deployment -n prod web") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
}

func TestInterpretKubectlGetEventsServiceOwnerFollowUp(t *testing.T) {
	text := "LAST SEEN TYPE REASON OBJECT MESSAGE\n20s Warning FailedMount pod/api-6d8f secret \"api-tls\" not found while serving traffic for service/api\n"
	response, err := Output("kubectl get events -n prod --sort-by=.metadata.creationTimestamp", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	combined := strings.Join(response.NextSteps, " ")
	if !strings.Contains(combined, "kubectl describe service -n prod api") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
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

func TestInterpretKubectlDescribePodExtractsContainerHintFromEvents(t *testing.T) {
	text := "Events:\n  Warning  BackOff  kubelet  Back-off restarting failed container api in pod web-7c5c_prod(1234)\n"
	response, err := Output("kubectl describe pod web-7c5c", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if response.ContainerHint != "api" {
		t.Fatalf("ContainerHint = %q, want api", response.ContainerHint)
	}
}

func TestInterpretKubectlLogsExtractsContainerHint(t *testing.T) {
	text := "Defaulted container \"api\" out of: api, sidecar\npanic: failed to connect to database: connection to server at db.internal port 5432 failed\n"
	response, err := Output("kubectl logs -n prod web-7c5c --tail=100", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if response.ContainerHint != "api" {
		t.Fatalf("ContainerHint = %q, want api", response.ContainerHint)
	}
}

func TestInterpretKubectlLogsDetectsDatabaseConnectivityFromConnectionRefused(t *testing.T) {
	text := "dial tcp db.prod.svc.cluster.local:5432: connect: connection refused\n"
	response, err := Output("kubectl logs -n prod worker-2 --tail=100", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	combined := strings.Join(response.NextSteps, " ")
	if !strings.Contains(response.Explanation, "database connectivity errors detected") {
		t.Fatalf("Explanation = %q", response.Explanation)
	}
	if !strings.Contains(combined, "dig +short db.prod.svc.cluster.local") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
	if !strings.Contains(combined, "nc -vz db.prod.svc.cluster.local 5432") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
	if strings.Contains(combined, "journalctl -u <service>") {
		t.Fatalf("NextSteps should not mention journalctl for kubectl logs: %#v", response.NextSteps)
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

func TestInterpretRuntimeLogsDetectsDatabaseConnectivity(t *testing.T) {
	text := "panic: failed to connect to database: connection to server at db.internal port 5432 failed\n"
	response, err := Output("podman logs --tail 100 worker2", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	combined := strings.Join(response.NextSteps, " ")
	if !strings.Contains(combined, "dig +short db.internal") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
	if !strings.Contains(combined, "nc -vz db.internal 5432") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
}

func TestInterpretRuntimeLogsAvoidsJournalctl(t *testing.T) {
	text := "permission denied opening /srv/data\n"
	response, err := Output("podman logs --tail 100 worker2", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	combined := strings.Join(response.NextSteps, " ")
	if strings.Contains(combined, "journalctl -u <service>") {
		t.Fatalf("NextSteps should not mention journalctl for runtime logs: %#v", response.NextSteps)
	}
	if !strings.Contains(combined, "podman ps -a") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
}

func TestInterpretJournalctlDetectsDNSResolution(t *testing.T) {
	text := "lookup api.internal on 10.96.0.10:53: no such host\n"
	response, err := Output("journalctl -u nginx -n 50 --no-pager", text)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	combined := strings.Join(response.NextSteps, " ")
	if !strings.Contains(combined, "dig +short api.internal") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
	if !strings.Contains(combined, "/etc/resolv.conf") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
}

func TestExtractDependencyHostFallbackEmpty(t *testing.T) {
	if got := extractDependencyHost("generic connection failure without a host"); got != "" {
		t.Fatalf("extractDependencyHost() = %q, want empty", got)
	}
}

func TestExtractDependencyPort(t *testing.T) {
	if got := extractDependencyPort("connection to server at db.internal port 5432 failed"); got != "5432" {
		t.Fatalf("extractDependencyPort() = %q, want 5432", got)
	}
	if got := extractDependencyPort("dial tcp api.internal:8443: connect: connection refused"); got != "8443" {
		t.Fatalf("extractDependencyPort() = %q, want 8443", got)
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
