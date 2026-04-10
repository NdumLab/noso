package explain

import (
	"fmt"
	"strings"

	"github.com/NdumLab/noso/internal/evidence"
	"github.com/NdumLab/noso/internal/safety"
	"github.com/NdumLab/noso/pkg/models"
)

func Command(query string, collector evidence.Collector) (models.Response, error) {
	command := extractCommand(query)
	base := firstWord(command)
	ev := collector.Lookup(base)

	response := models.Response{
		IntentID:       "explain_command",
		Command:        command,
		Explanation:    explainCommand(command),
		ExpectedOutput: "No command is run. This mode explains behavior, impact, and likely outcomes for the provided command.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	if len(ev.HelpSnippet) > 0 {
		response.VerifiedFrom = append(response.VerifiedFrom, base+" --help")
	}
	if !ev.Exists && base != "" {
		response.Warnings = append(response.Warnings, base+" is not currently installed on this host")
	}
	return response, nil
}

func extractCommand(query string) string {
	trimmed := strings.TrimSpace(query)
	lower := strings.ToLower(trimmed)
	switch {
	case strings.HasPrefix(lower, "explain "):
		trimmed = strings.TrimSpace(trimmed[len("explain "):])
	case strings.HasPrefix(lower, "what does "):
		trimmed = strings.TrimSpace(trimmed[len("what does "):])
	}
	trimmed = strings.Trim(trimmed, "`\"'")
	return trimmed
}

func explainCommand(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return "No command was provided to explain."
	}

	base := fields[0]
	switch {
	case command == "git status --short --branch":
		return "Shows the current branch and a compact working-tree summary. Modified, added, deleted, and untracked files appear in a short two-column format."
	case strings.HasPrefix(command, "git log"):
		return "Displays commit history from newest to oldest. Common flags shorten output, limit commit count, or render one-line summaries."
	case strings.HasPrefix(command, "git diff"):
		return "Shows line-by-line differences between the working tree, index, or commits. This is read-only and useful before any Git cleanup or commit step."
	case strings.HasPrefix(command, "git reset --hard"):
		return "Moves HEAD and the current branch to the target commit, resets the index, and discards tracked working-tree changes. This can permanently lose uncommitted work."
	case strings.HasPrefix(command, "rm -rf"):
		return "Recursively removes files and directories without prompting. This is destructive and can delete large parts of the filesystem if the path is wrong."
	case strings.HasPrefix(command, "systemctl restart"):
		return "Restarts the named systemd unit. Existing service processes are stopped and started again, which changes host state and can interrupt traffic."
	case strings.HasPrefix(command, "systemctl status"):
		return "Queries systemd for the current unit state, recent log lines, and service metadata. This is read-only and commonly used for first diagnostics."
	case strings.HasPrefix(command, "journalctl"):
		return "Reads entries from the systemd journal. Flags here usually scope results to a unit and limit the number of lines returned."
	case strings.HasPrefix(command, "ss "):
		return "Shows socket and listening-port information from the kernel. Flags like -l, -t, -n, and -p narrow the results and expose process ownership."
	case strings.HasPrefix(command, "find "):
		return "Walks the filesystem and applies tests like type, size, and name to each path. It is read-only unless explicit action flags such as -delete are used."
	case strings.HasPrefix(command, "du "):
		return "Summarizes file or directory space usage. `-s` reports a total and `-h` renders sizes in human-readable units."
	case strings.HasPrefix(command, "df "):
		return "Reports filesystem capacity, used space, and free space for mounted filesystems. `-h` prints human-readable units."
	case strings.HasPrefix(command, "grep "):
		return "Searches input text or files for matching patterns. Recursive and line-number flags help narrow results during troubleshooting."
	case strings.HasPrefix(command, "rpm -qi"):
		return "Queries RPM metadata for an installed package and prints version, release, vendor, summary, and related package details."
	case strings.HasPrefix(command, "tar -tf"):
		return "Lists archive members without extracting them. This is a safe way to inspect tarball contents before unpacking."
	case strings.HasPrefix(command, "ip addr"):
		return "Displays network interface addresses, link states, and assigned IPv4 or IPv6 information."
	case strings.HasPrefix(command, "ping "):
		return "Sends ICMP echo requests to test basic network reachability and latency to a host."
	case strings.HasPrefix(command, "curl -I"):
		return "Fetches only the response headers for a URL, which is useful for checking HTTP status and server metadata without downloading the full body."
	case strings.HasPrefix(command, "helm repo list"):
		return "Lists configured Helm repositories from the local client configuration without modifying any cluster or release state."
	case strings.HasPrefix(command, "helm list"):
		return "Lists Helm releases and their namespace, revision, status, and chart metadata."
	case strings.HasPrefix(command, "helm status"):
		return "Shows detailed information for a single Helm release, including resources, notes, and last deployment metadata."
	case strings.HasPrefix(command, "helm history"):
		return "Displays the revision history of a Helm release so you can inspect prior deploy states without changing them."
	case strings.HasPrefix(command, "helm get values"):
		return "Prints the user-supplied values for a Helm release, which helps confirm override settings and chart inputs."
	case strings.HasPrefix(command, "helm template"):
		return "Renders chart templates locally and prints Kubernetes manifests without installing or modifying a release."
	case strings.HasPrefix(command, "terraform fmt -check"):
		return "Checks Terraform file formatting without rewriting files, which makes it safe for CI-style validation."
	case strings.HasPrefix(command, "terraform validate"):
		return "Validates Terraform configuration syntax and internal references without changing infrastructure."
	case strings.HasPrefix(command, "terraform plan"):
		return "Builds a proposed infrastructure change set without applying it, which is safer than apply but can still reveal intended creates, updates, or destroys."
	case strings.HasPrefix(command, "terraform workspace list"):
		return "Lists Terraform workspaces and marks the currently selected one."
	case strings.HasPrefix(command, "terraform state list"):
		return "Shows resource addresses tracked in the current Terraform state without mutating that state."
	case strings.HasPrefix(command, "ansible --version"):
		return "Shows the installed Ansible version, configuration path, module locations, and Python runtime details."
	case strings.HasPrefix(command, "ansible-inventory --list"):
		return "Prints the resolved inventory so you can inspect hosts, groups, and variables before running automation."
	case strings.HasPrefix(command, "ansible-playbook --syntax-check"):
		return "Validates the playbook structure and YAML syntax without executing tasks on remote hosts."
	case strings.HasPrefix(command, "ansible-playbook --check"):
		return "Runs Ansible in check mode so tasks report intended changes without making most changes, although some modules may not fully support check mode."
	case strings.HasPrefix(command, "ssh -G"):
		return "Shows the effective SSH client configuration for a host, including hostname, port, user, and identity file settings, without opening a session."
	case strings.HasPrefix(command, "ssh-keyscan"):
		return "Fetches the remote host's presented SSH public keys so you can inspect or pin them before connecting."
	case strings.HasPrefix(command, "nc -zv"):
		return "Checks whether a TCP port is reachable. Here it is used to test whether the SSH port is reachable before debugging keys or authentication."
	case strings.HasPrefix(command, "rsync -avhn"):
		return "Runs rsync in archive, verbose, human-readable, and dry-run mode so you can preview a remote transfer without copying files."
	case strings.HasPrefix(command, "aws sts get-caller-identity"):
		return "Shows the currently authenticated AWS identity and account context without changing any AWS resources."
	case strings.HasPrefix(command, "aws configure list-profiles"):
		return "Lists named AWS CLI profiles configured on the local machine."
	case strings.HasPrefix(command, "az account show"):
		return "Shows the currently selected Azure subscription and tenant context for the active Azure CLI session."
	case strings.HasPrefix(command, "az account list"):
		return "Lists Azure subscriptions available to the current login and indicates which one is selected."
	case strings.HasPrefix(command, "gcloud auth list"):
		return "Lists authenticated Google Cloud accounts and marks the currently active one."
	case strings.HasPrefix(command, "gcloud config get-value project"):
		return "Shows the active Google Cloud project configured in the current gcloud profile."
	case strings.HasPrefix(command, "argocd account get-user-info"):
		return "Shows the currently authenticated Argo CD account and capability information without changing application state."
	case strings.HasPrefix(command, "argocd app list"):
		return "Lists Argo CD applications and summarizes their sync and health status."
	case strings.HasPrefix(command, "argocd app get"):
		return "Shows detailed status for a single Argo CD application, including sync state, health, and managed resources."
	case strings.HasPrefix(command, "argocd proj list"):
		return "Lists Argo CD projects and their configured scope boundaries."
	case strings.HasPrefix(command, "argocd cluster list"):
		return "Lists Kubernetes clusters registered with Argo CD."
	case strings.HasPrefix(command, "getenforce"):
		return "Shows the current SELinux enforcement mode without changing policy state."
	case strings.HasPrefix(command, "sestatus"):
		return "Shows SELinux status, policy details, and configured enforcement state."
	case strings.HasPrefix(command, "firewall-cmd --list-all"):
		return "Shows the active firewalld zone configuration, including services, ports, interfaces, and rules."
	case strings.HasPrefix(command, "firewall-cmd --get-active-zones"):
		return "Lists active firewalld zones and the interfaces or sources assigned to them."
	case strings.HasPrefix(command, "openssl x509 -in"):
		return "Decodes an X.509 certificate file so you can inspect subject, issuer, validity, SANs, and other extensions without modifying the certificate."
	case strings.HasPrefix(command, "lscpu"):
		return "Shows CPU architecture, model, socket, core, thread, and virtualization details from the local system."
	case strings.HasPrefix(command, "free -h"):
		return "Shows system memory and swap totals, usage, and available memory in human-readable units."
	case strings.HasPrefix(command, "lsblk -o"):
		return "Lists block devices, sizes, types, filesystems, and mountpoints so you can inspect disk layout without changing it."
	case strings.HasPrefix(command, "dmidecode -t system"):
		return "Reads SMBIOS or DMI system information such as vendor, product name, serial, and UUID. It is read-only but may require elevated privileges on some hosts."
	case strings.HasPrefix(command, "smartctl -H"):
		return "Shows the overall SMART health assessment for a disk without changing the device state."
	case strings.HasPrefix(command, "ipmitool mc info"):
		return "Shows basic IPMI management controller information, which helps confirm BMC reachability and firmware identity."
	case strings.HasPrefix(command, "nvidia-smi"):
		return "Shows NVIDIA GPU inventory, driver state, memory usage, utilization, and active processes without changing GPU settings."
	case strings.HasPrefix(command, "psql --version"):
		return "Shows the installed PostgreSQL client version."
	case strings.HasPrefix(command, "psql -l"):
		return "Lists PostgreSQL databases visible to the current connection settings, including owner, encoding, and access privileges."
	case strings.HasPrefix(command, "mysql --version"):
		return "Shows the installed MySQL client version."
	case strings.HasPrefix(command, "mysql --execute=\"show databases;\""):
		return "Asks the MySQL client to list databases visible to the current login context."
	case strings.HasPrefix(command, "redis-cli --version"):
		return "Shows the installed Redis CLI version."
	case strings.HasPrefix(command, "redis-cli ping"):
		return "Sends a simple Redis ping command and expects PONG when a Redis server is reachable with the current connection settings."
	default:
		return fmt.Sprintf("Explains the command starting with `%s`. The tool can identify the base command, classify its risk, and use local help evidence when the command exists on this host.", base)
	}
}

func firstWord(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func confidenceFor(ev evidence.Evidence) string {
	switch {
	case ev.Exists && len(ev.HelpSnippet) > 0:
		return "High"
	case ev.Exists:
		return "Medium"
	default:
		return "Low"
	}
}
