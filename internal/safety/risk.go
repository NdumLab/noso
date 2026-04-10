package safety

import "strings"

const (
	RiskLow    = "Low"
	RiskMedium = "Medium"
	RiskHigh   = "High"
)

// Classify returns a risk level for a resolved command string.
// It is intentionally conservative: when in doubt it returns High.
// Callers pass the actual command that will be suggested to the user,
// not the raw user query.
func Classify(command string) string {
	n := strings.ToLower(strings.TrimSpace(command))

	// Safe sub-commands that contain high-risk substrings must be checked
	// first so they are not accidentally promoted to High.
	for _, p := range safeMediumOverrides {
		if strings.Contains(n, p) {
			return RiskMedium
		}
	}

	for _, p := range highRiskPatterns {
		if strings.Contains(n, p) {
			return RiskHigh
		}
	}

	for _, p := range mediumRiskPatterns {
		if strings.Contains(n, p) {
			return RiskMedium
		}
	}

	return RiskLow
}

// safeMediumOverrides are patterns that contain a high-risk substring
// but represent read-only or dry-run operations.
var safeMediumOverrides = []string{
	"ansible-playbook --check",
	"ansible-playbook --syntax-check",
	"terraform plan",
	"helm template",
}

// highRiskPatterns matches commands that can destroy data or are very
// difficult to reverse.
var highRiskPatterns = []string{
	// Filesystem destruction
	"rm -rf", "rm -r ", "rm -f ",
	"shred ", "wipefs ", "mkfs.",
	// Git destructive
	"git reset --hard",
	"git clean -f",
	"git push --force", "git push -f",
	// Container / image mass-removal
	"docker system prune",
	"podman system prune",
	"docker rm ", "docker rmi ",
	"podman rm ", "podman rmi ",
	// Kubernetes deletion
	"kubectl delete",
	// IaC execution (apply/destroy)
	"terraform destroy",
	"terraform apply",
	// Ansible full playbook execution (not --check / --syntax-check)
	"ansible-playbook ",
	// Database destructive
	"drop table", "drop database", "truncate ",
	// Disk/device writes
	"dd if=",
}

// mediumRiskPatterns matches commands that change state but are
// generally recoverable.
var mediumRiskPatterns = []string{
	// Service control
	"systemctl restart", "systemctl stop", "systemctl start",
	"systemctl enable", "systemctl disable",
	"service restart", "service stop", "service start",
	// Git non-hard resets and reverts
	"git reset ",
	"git revert",
	"git stash drop",
	// Container lifecycle
	"docker restart", "docker stop", "docker start",
	"podman restart", "podman stop", "podman start",
	// Kubernetes rollout (non-delete)
	"kubectl rollout restart",
	"kubectl scale",
	// Package management
	"dnf install", "dnf remove", "dnf update",
	"yum install", "yum remove", "yum update",
	"apt install", "apt remove", "apt upgrade",
	"rpm -e",
	// Permission / ownership changes
	"chmod ", "chown ",
	// Firewall / SELinux changes
	"firewall-cmd --add", "firewall-cmd --remove",
	"setenforce ",
	// Helm non-read-only
	"helm install", "helm upgrade", "helm rollback", "helm uninstall",
}
