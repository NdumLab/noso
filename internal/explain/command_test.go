package explain

import (
	"strings"
	"testing"

	"github.com/NdumLab/noso/internal/evidence"
)

func TestExplainDangerousGitReset(t *testing.T) {
	response, err := Command("explain `git reset --hard HEAD~1`", evidence.NewCollector())
	if err != nil {
		t.Fatalf("Command() error = %v", err)
	}
	if response.Risk != "High" {
		t.Fatalf("Risk = %q, want High", response.Risk)
	}
	if !strings.Contains(response.Explanation, "permanently lose uncommitted work") {
		t.Fatalf("Explanation = %q", response.Explanation)
	}
}

func TestExplainReadOnlyCommandsAreLowRisk(t *testing.T) {
	cases := []string{
		"explain df -h",
		"explain git status",
		"explain systemctl status nginx",
		"explain kubectl get pods",
		"explain helm list",
	}
	for _, query := range cases {
		t.Run(query, func(t *testing.T) {
			resp, err := Command(query, evidence.NewCollector())
			if err != nil {
				t.Fatalf("Command() error = %v", err)
			}
			if resp.Risk != "Low" {
				t.Errorf("Risk = %q, want Low for read-only command", resp.Risk)
			}
		})
	}
}

func TestExplainDestructiveCommandsAreHighRisk(t *testing.T) {
	cases := []string{
		"explain rm -rf /var",
		"explain kubectl delete pod mypod",
		"explain terraform destroy",
		"explain docker system prune -a",
	}
	for _, query := range cases {
		t.Run(query, func(t *testing.T) {
			resp, err := Command(query, evidence.NewCollector())
			if err != nil {
				t.Fatalf("Command() error = %v", err)
			}
			if resp.Risk != "High" {
				t.Errorf("Risk = %q, want High for destructive command", resp.Risk)
			}
		})
	}
}

func TestExplainExtractsCommandFromQuery(t *testing.T) {
	cases := []struct {
		query   string
		wantCmd string
	}{
		{"explain `git log --oneline`", "git log --oneline"},
		{`explain "git diff HEAD"`, "git diff HEAD"},
		{"what does systemctl restart nginx do", "systemctl restart nginx do"},
	}
	for _, tc := range cases {
		got := extractCommand(tc.query)
		if got != tc.wantCmd {
			t.Errorf("extractCommand(%q) = %q, want %q", tc.query, got, tc.wantCmd)
		}
	}
}

func TestExplainUnknownCommandReturnsFallback(t *testing.T) {
	resp, err := Command("explain some-totally-unknown-command --flag", evidence.NewCollector())
	if err != nil {
		t.Fatalf("Command() error = %v", err)
	}
	if resp.IntentID != "explain_command" {
		t.Errorf("IntentID = %q", resp.IntentID)
	}
	if resp.Explanation == "" {
		t.Error("Explanation should not be empty for unknown command")
	}
}

func TestExplainCommandExplanations(t *testing.T) {
	cases := []struct {
		command string
		want    string
	}{
		{"git log --oneline -20", "commit history"},
		{"git diff HEAD", "line-by-line differences"},
		{"git status --short --branch", "compact working-tree"},
		{"journalctl -u nginx -n 50 --no-pager", "systemd journal"},
		{"ss -ltnp", "socket"},
		{"find /var/log -name '*.log'", "filesystem"},
		{"du -sh /var/log", "space usage"},
		{"df -h", "filesystem capacity"},
		{"grep -r error /var/log", "Searches"},
		{"rpm -qi bash", "RPM metadata"},
		{"tar -tf archive.tar.gz", "archive members"},
		{"ip addr show", "network interface"},
		{"ping 8.8.8.8", "ICMP echo"},
		{"curl -I https://example.com", "response headers"},
		{"helm repo list", "Helm repositories"},
		{"helm status myapp", "single Helm release"},
		{"helm history myapp", "revision history"},
		{"helm get values myapp", "user-supplied values"},
		{"helm template mychart", "chart templates"},
		{"terraform fmt -check", "formatting"},
		{"terraform validate", "syntax"},
		{"terraform plan", "change set"},
		{"terraform workspace list", "workspaces"},
		{"terraform state list", "resource addresses"},
		{"ansible --version", "Ansible version"},
		{"ansible-inventory --list", "inventory"},
		{"ansible-playbook --syntax-check site.yml", "YAML syntax"},
		{"ansible-playbook --check site.yml", "check mode"},
		{"ssh -G myhost", "SSH client configuration"},
		{"ssh-keyscan github.com", "public keys"},
		{"nc -zv 10.0.0.1 22", "TCP port is reachable"},
		{"rsync -avhn /src/ /dst/", "dry-run"},
		{"aws sts get-caller-identity", "AWS identity"},
		{"aws configure list-profiles", "AWS CLI profiles"},
		{"az account show", "Azure subscription"},
		{"az account list", "subscriptions"},
		{"gcloud auth list", "Google Cloud accounts"},
		{"gcloud config get-value project", "project"},
		{"argocd account get-user-info", "Argo CD account"},
		{"argocd app list", "Argo CD applications"},
		{"argocd app get myapp", "single Argo CD application"},
		{"argocd proj list", "projects"},
		{"argocd cluster list", "clusters"},
		{"getenforce", "SELinux enforcement mode"},
		{"sestatus", "SELinux status"},
		{"firewall-cmd --list-all", "firewalld zone"},
		{"firewall-cmd --get-active-zones", "active firewalld zones"},
		{"openssl x509 -in cert.pem -text -noout", "X.509 certificate"},
		{"lscpu", "CPU architecture"},
		{"free -h", "memory"},
		{"lsblk -o NAME,SIZE,TYPE,FSTYPE,MOUNTPOINT", "block devices"},
		{"dmidecode -t system", "SMBIOS"},
		{"smartctl -H /dev/sda", "SMART health"},
		{"ipmitool mc info", "IPMI"},
		{"nvidia-smi", "GPU"},
		{"psql --version", "PostgreSQL client version"},
		{"psql -l", "PostgreSQL databases"},
		{"mysql --version", "MySQL client version"},
		{"mysql --execute=\"show databases;\"", "databases"},
		{"redis-cli --version", "Redis CLI version"},
		{"redis-cli ping", "Redis ping"},
	}
	for _, tc := range cases {
		t.Run(tc.command, func(t *testing.T) {
			got := explainCommand(tc.command)
			if !strings.Contains(strings.ToLower(got), strings.ToLower(tc.want)) {
				t.Errorf("explainCommand(%q) = %q, want substring %q", tc.command, got, tc.want)
			}
		})
	}
}

func TestExplainCommandEmptyInput(t *testing.T) {
	got := explainCommand("")
	if got == "" {
		t.Error("empty command should return a non-empty explanation")
	}
}

func TestExplainCommandUnknownFallback(t *testing.T) {
	got := explainCommand("some-totally-unknown-tool --args")
	if !strings.Contains(got, "some-totally-unknown-tool") {
		t.Errorf("fallback explanation should include base command name, got: %s", got)
	}
}

func TestConfidenceForExistsWithHelp(t *testing.T) {
	ev := evidence.Evidence{Exists: true, HelpSnippet: []string{"usage: foo"}}
	if got := confidenceFor(ev); got != "High" {
		t.Errorf("confidenceFor(exists+help) = %q, want High", got)
	}
}

func TestConfidenceForExistsNoHelp(t *testing.T) {
	ev := evidence.Evidence{Exists: true}
	if got := confidenceFor(ev); got != "Medium" {
		t.Errorf("confidenceFor(exists, no help) = %q, want Medium", got)
	}
}

func TestConfidenceForNotExists(t *testing.T) {
	ev := evidence.Evidence{Exists: false}
	if got := confidenceFor(ev); got != "Low" {
		t.Errorf("confidenceFor(not exists) = %q, want Low", got)
	}
}

func TestCommandAddsWarningForMissingTool(t *testing.T) {
	resp, err := Command("explain some-missing-tool --flag", evidence.NewCollector())
	if err != nil {
		t.Fatalf("Command() error = %v", err)
	}
	found := false
	for _, w := range resp.Warnings {
		if strings.Contains(w, "not currently installed") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'not currently installed' warning, got warnings: %v", resp.Warnings)
	}
}
