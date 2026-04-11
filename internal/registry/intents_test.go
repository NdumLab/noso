package registry

import (
	"strings"
	"testing"

	"github.com/NdumLab/noso/internal/evidence"
	"github.com/NdumLab/noso/pkg/models"
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

func TestResolveServiceRestartIntent(t *testing.T) {
	cases := []struct {
		query   string
		service string
		action  string
	}{
		{"restart nginx", "nginx", "restart"},
		{"restart the sshd service", "sshd", "restart"},
		{"stop apache2", "apache2", "stop"},
		{"start postgresql", "postgresql", "start"},
		{"enable firewalld", "firewalld", "enable"},
		{"disable chronyd", "chronyd", "disable"},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			resp, err := Resolve(tc.query, models.Environment{}, evidence.NewCollector())
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if resp.IntentID != "control_service" {
				t.Fatalf("IntentID = %q, want control_service", resp.IntentID)
			}
			if !strings.Contains(resp.Command, tc.action) {
				t.Errorf("Command %q does not contain action %q", resp.Command, tc.action)
			}
			if !strings.Contains(resp.Command, tc.service) {
				t.Errorf("Command %q does not contain service %q", resp.Command, tc.service)
			}
			if resp.Risk == "" {
				t.Error("Risk should not be empty")
			}
		})
	}
}

func TestResolveServiceControlDoesNotClobberStatus(t *testing.T) {
	// "nginx service status" must still resolve to inspect_service_status, not control_service
	resp, err := Resolve("nginx service status", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resp.IntentID != "inspect_service_status" {
		t.Fatalf("IntentID = %q, want inspect_service_status", resp.IntentID)
	}
}

func TestResolveGitPushIntent(t *testing.T) {
	cases := []string{"git push", "push git changes", "push my commits"}
	for _, q := range cases {
		t.Run(q, func(t *testing.T) {
			resp, err := Resolve(q, models.Environment{}, evidence.NewCollector())
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if resp.IntentID != "push_git_changes" {
				t.Fatalf("IntentID = %q, want push_git_changes", resp.IntentID)
			}
			if resp.Command != "git push" {
				t.Errorf("Command = %q, want 'git push'", resp.Command)
			}
			// Must warn about force-push risk
			found := false
			for _, w := range resp.Warnings {
				if strings.Contains(w, "force") {
					found = true
				}
			}
			if !found {
				t.Error("expected force-push warning in Warnings")
			}
		})
	}
}

func TestResolvePackageInstallIntent(t *testing.T) {
	cases := []struct {
		query string
		pkg   string
	}{
		{"install curl", "curl"},
		{"install package nginx", "nginx"},
		{"install openssl", "openssl"},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			resp, err := Resolve(tc.query, models.Environment{PackageManager: "dnf"}, evidence.NewCollector())
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if resp.IntentID != "install_package" {
				t.Fatalf("IntentID = %q, want install_package", resp.IntentID)
			}
			if !strings.Contains(resp.Command, tc.pkg) {
				t.Errorf("Command %q does not contain package %q", resp.Command, tc.pkg)
			}
		})
	}
}

func TestResolvePackageInstallDistroAware(t *testing.T) {
	cases := []struct {
		pm      string
		wantPfx string
	}{
		{"dnf", "dnf install"},
		{"apt", "apt-get install"},
		{"pacman", "pacman -S"},
		{"zypper", "zypper install"},
	}
	for _, tc := range cases {
		t.Run(tc.pm, func(t *testing.T) {
			resp, err := Resolve("install curl", models.Environment{PackageManager: tc.pm}, evidence.NewCollector())
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if !strings.HasPrefix(resp.Command, tc.wantPfx) {
				t.Errorf("Command = %q, want prefix %q", resp.Command, tc.wantPfx)
			}
		})
	}
}

func TestResolveDNSLookupIntent(t *testing.T) {
	cases := []struct {
		query string
		host  string
	}{
		{"nslookup example.com", "example.com"},
		{"dns lookup for google.com", "google.com"},
		{"dig 8.8.8.8", "8.8.8.8"},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			resp, err := Resolve(tc.query, models.Environment{}, evidence.NewCollector())
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if resp.IntentID != "inspect_dns_lookup" {
				t.Fatalf("IntentID = %q, want inspect_dns_lookup", resp.IntentID)
			}
			if !strings.Contains(resp.Command, tc.host) {
				t.Errorf("Command %q does not contain host %q", resp.Command, tc.host)
			}
		})
	}
}

func TestResolveCronListIntent(t *testing.T) {
	cases := []string{
		"list cron jobs",
		"show crontab",
		"view cron",
		"list scheduled tasks",
	}
	for _, q := range cases {
		t.Run(q, func(t *testing.T) {
			resp, err := Resolve(q, models.Environment{}, evidence.NewCollector())
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if resp.IntentID != "inspect_cron_jobs" {
				t.Fatalf("IntentID = %q, want inspect_cron_jobs", resp.IntentID)
			}
			if resp.Command != "crontab -l" {
				t.Errorf("Command = %q, want 'crontab -l'", resp.Command)
			}
		})
	}
}
