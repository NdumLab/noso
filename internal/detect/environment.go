package detect

import (
	"os"
	"strings"

	"github.com/NdumLab/noso/internal/evidence"
	"github.com/NdumLab/noso/pkg/models"
)

var defaultCommands = []string{
	"bash", "dnf", "rpm", "systemctl", "journalctl", "ss", "ip", "ping", "curl",
	"wget", "git", "find", "du", "df", "grep", "awk", "sed", "tar", "ssh", "scp", "rsync", "ssh-keyscan", "nc",
	"docker", "podman", "containerd", "ctr", "crictl", "nerdctl", "kubectl", "helm", "terraform", "ansible", "ansible-playbook",
	"aws", "az", "gcloud", "argocd", "getenforce", "sestatus", "firewall-cmd", "openssl",
	"lscpu", "free", "lsblk", "dmidecode", "smartctl", "ipmitool", "nvidia-smi",
	"psql", "mysql", "redis-cli",
}

func Local() (models.Environment, error) {
	osRelease, err := parseOSRelease("/etc/os-release")
	if err != nil {
		return models.Environment{}, err
	}

	collector := evidence.NewCollector()
	commands := make(map[string]models.CommandInfo, len(defaultCommands))
	for _, name := range defaultCommands {
		ev := collector.BasicLookup(name)
		commands[name] = models.CommandInfo{
			Name:   name,
			Path:   ev.Path,
			Type:   ev.Kind,
			Exists: ev.Exists,
		}
	}

	id := osRelease["ID"]
	ver := osRelease["VERSION_ID"]

	return models.Environment{
		OSID:           id,
		VersionID:      ver,
		PrettyName:     osRelease["PRETTY_NAME"],
		Distro:         distroFamily(id, osRelease["ID_LIKE"]),
		PackageManager: detectPackageManager(id, osRelease["ID_LIKE"]),
		Shell:          os.Getenv("SHELL"),
		IsRHEL9:        id == "rhel" && strings.HasPrefix(ver, "9"),
		KubeConfig:     kubeConfigPath(),
		KubeContext:    kubeContext(collector),
		Commands:       commands,
	}, nil
}

func kubeConfigPath() string {
	if v := os.Getenv("KUBECONFIG"); v != "" {
		if fileExists(v) {
			return v
		}
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	path := home + "/.kube/config"
	if fileExists(path) {
		return path
	}
	return ""
}

func kubeContext(collector evidence.Collector) string {
	if ev := collector.Lookup("kubectl"); !ev.Exists {
		return ""
	}
	lines := collectorCommandLines("kubectl config current-context")
	if len(lines) == 0 {
		return ""
	}
	return lines[0]
}

func collectorCommandLines(script string) []string {
	collector := evidence.NewCollector()
	return collector.RunLinesForDetection(script)
}

func parseOSRelease(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	values := make(map[string]string)
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[key] = strings.Trim(value, `"`)
	}

	return values, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// distroFamily maps the os-release ID (and optional ID_LIKE) to a
// normalised family name for use in user-facing messages and logic.
func distroFamily(id, idLike string) string {
	id = strings.ToLower(id)
	idLike = strings.ToLower(idLike)

	switch {
	case id == "rhel" || id == "centos" || id == "rocky" || id == "almalinux" ||
		strings.Contains(idLike, "rhel") || strings.Contains(idLike, "centos"):
		return "rhel"
	case id == "fedora" || strings.Contains(idLike, "fedora"):
		return "fedora"
	case id == "ubuntu" || id == "debian" || id == "linuxmint" || id == "pop" ||
		strings.Contains(idLike, "ubuntu") || strings.Contains(idLike, "debian"):
		return "debian"
	case id == "opensuse" || id == "sles" || strings.Contains(idLike, "suse"):
		return "suse"
	case id == "arch" || id == "manjaro" || strings.Contains(idLike, "arch"):
		return "arch"
	default:
		return "unknown"
	}
}

// detectPackageManager returns the primary package manager for the distro.
func detectPackageManager(id, idLike string) string {
	family := distroFamily(id, idLike)
	switch family {
	case "rhel", "fedora":
		return "dnf"
	case "debian":
		return "apt"
	case "suse":
		return "zypper"
	case "arch":
		return "pacman"
	default:
		return "unknown"
	}
}
