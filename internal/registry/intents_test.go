package registry

import (
	"strings"
	"testing"

	"github.com/noso-dev/noso/internal/evidence"
	"github.com/noso-dev/noso/pkg/models"
)

func TestResolveLargeFilesIntent(t *testing.T) {
	response, err := Resolve("find files larger than 2G in /srv", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if response.IntentID != "find_large_files" {
		t.Fatalf("IntentID = %q, want find_large_files", response.IntentID)
	}
	if !strings.Contains(response.Command, "find /srv -type f -size +2G") {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestResolveUnsupportedIntent(t *testing.T) {
	response, err := Resolve("explain git rebase", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if response.IntentID != "explain_command" {
		t.Fatalf("IntentID = %q, want explain_command", response.IntentID)
	}
}

func TestResolvePackageInfoIntent(t *testing.T) {
	response, err := Resolve("package info for openssl", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if response.IntentID != "inspect_package_info" {
		t.Fatalf("IntentID = %q, want inspect_package_info", response.IntentID)
	}
	if !strings.Contains(response.Command, "rpm -qi openssl") {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestResolveGrepIntent(t *testing.T) {
	response, err := Resolve(`search for "panic" in /var/log`, models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if response.IntentID != "search_text_in_files" {
		t.Fatalf("IntentID = %q, want search_text_in_files", response.IntentID)
	}
	if !strings.Contains(response.Command, `grep -Rni -- "panic" /var/log`) {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestResolveGitStatusIntent(t *testing.T) {
	response, err := Resolve("show git status", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if response.IntentID != "inspect_git_status" {
		t.Fatalf("IntentID = %q, want inspect_git_status", response.IntentID)
	}
	if response.Risk != "Low" {
		t.Fatalf("Risk = %q, want Low", response.Risk)
	}
}

func TestResolveDiskFreeIntent(t *testing.T) {
	response, err := Resolve("show disk free space", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_disk_free_space" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestResolvePingIntent(t *testing.T) {
	response, err := Resolve("ping example.com", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_host_reachability" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if !strings.Contains(response.Command, "example.com") {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestResolveGitLogIntent(t *testing.T) {
	response, err := Resolve("git log", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_git_log" {
		t.Fatalf("IntentID = %q, want inspect_git_log", response.IntentID)
	}
}

func TestResolveGitDiffIntent(t *testing.T) {
	response, err := Resolve("git diff", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_git_diff" {
		t.Fatalf("IntentID = %q, want inspect_git_diff", response.IntentID)
	}
}

func TestResolveGitBranchIntent(t *testing.T) {
	response, err := Resolve("git branch", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_git_branches" {
		t.Fatalf("IntentID = %q, want inspect_git_branches", response.IntentID)
	}
}

func TestResolveProcessIntent(t *testing.T) {
	response, err := Resolve("top processes", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_top_processes" {
		t.Fatalf("IntentID = %q, want inspect_top_processes", response.IntentID)
	}
}

func TestResolveIPAddressIntent(t *testing.T) {
	response, err := Resolve("show network interfaces", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_ip_addresses" {
		t.Fatalf("IntentID = %q, want inspect_ip_addresses", response.IntentID)
	}
}

func TestResolveCurlHeadIntent(t *testing.T) {
	response, err := Resolve("check website http headers", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_http_headers" {
		t.Fatalf("IntentID = %q, want inspect_http_headers", response.IntentID)
	}
}

func TestResolveTailFileIntent(t *testing.T) {
	response, err := Resolve("tail log /var/log/nginx/error.log", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_file_tail" {
		t.Fatalf("IntentID = %q, want inspect_file_tail", response.IntentID)
	}
	if !strings.Contains(response.Command, "/var/log/nginx/error.log") {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestResolveTarListIntent(t *testing.T) {
	response, err := Resolve("list tar archive.tar.gz", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_archive_contents" {
		t.Fatalf("IntentID = %q, want inspect_archive_contents", response.IntentID)
	}
}

func TestResolvePortIntent(t *testing.T) {
	response, err := Resolve("what process is on port 8080", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_port_listener" {
		t.Fatalf("IntentID = %q, want inspect_port_listener", response.IntentID)
	}
	if !strings.Contains(response.Command, "8080") {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestResolveServiceLogsIntent(t *testing.T) {
	response, err := Resolve("logs for nginx service", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_service_logs" {
		t.Fatalf("IntentID = %q, want inspect_service_logs", response.IntentID)
	}
}

func TestResolveServiceStatusIntent(t *testing.T) {
	response, err := Resolve("nginx service status", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_service_status" {
		t.Fatalf("IntentID = %q, want inspect_service_status", response.IntentID)
	}
}

func TestResolveDiskUsageIntent(t *testing.T) {
	response, err := Resolve("disk usage of /var", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_directory_size" {
		t.Fatalf("IntentID = %q, want inspect_directory_size", response.IntentID)
	}
	if !strings.Contains(response.Command, "/var") {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestResolveSSHVersionIntent(t *testing.T) {
	response, err := Resolve("ssh version", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_ssh_version" {
		t.Fatalf("IntentID = %q, want inspect_ssh_version", response.IntentID)
	}
}

func TestResolveSCPPreviewIntent(t *testing.T) {
	response, err := Resolve("preview scp", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "preview_scp_transfer" {
		t.Fatalf("IntentID = %q, want preview_scp_transfer", response.IntentID)
	}
}

func TestResolveExplainIntent(t *testing.T) {
	response, err := Resolve("explain `rm -rf /tmp/demo`", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "explain_command" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if response.Risk != "High" {
		t.Fatalf("Risk = %q", response.Risk)
	}
}

func TestResolveContainerdStatusIntent(t *testing.T) {
	response, err := Resolve("containerd status", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if response.IntentID != "inspect_containerd_status" {
		t.Fatalf("IntentID = %q, want inspect_containerd_status", response.IntentID)
	}
	if response.Command != "systemctl status containerd --no-pager -l" {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestResolveContainerdLogsIntent(t *testing.T) {
	response, err := Resolve("logs for containerd", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if response.IntentID != "inspect_containerd_logs" {
		t.Fatalf("IntentID = %q, want inspect_containerd_logs", response.IntentID)
	}
	if response.Command != "journalctl -u containerd -n 50 --no-pager" {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestResolveContainerdVersionIntent(t *testing.T) {
	response, err := Resolve("containerd version", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if response.IntentID != "inspect_containerd_version" {
		t.Fatalf("IntentID = %q, want inspect_containerd_version", response.IntentID)
	}
	if response.Command == "" {
		t.Fatal("Command should not be empty")
	}
}
