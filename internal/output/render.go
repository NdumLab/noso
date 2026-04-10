package output

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/noso-dev/noso/pkg/models"
)

// RenderResponse formats a Response for terminal output.
// When quiet is true, warnings and next-step lines are suppressed.
func RenderResponse(response models.Response, asJSON bool, quiet bool) (string, error) {
	if asJSON {
		data, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data) + "\n", nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Intent: %s\n\n", response.IntentID)
	fmt.Fprintf(&b, "Command:\n%s\n\n", response.Command)
	fmt.Fprintf(&b, "Explanation:\n%s\n\n", response.Explanation)
	fmt.Fprintf(&b, "Expected output:\n%s\n\n", response.ExpectedOutput)
	fmt.Fprintf(&b, "Risk: %s\n", response.Risk)
	fmt.Fprintf(&b, "Confidence: %s\n", response.Confidence)
	if len(response.VerifiedFrom) > 0 {
		fmt.Fprintf(&b, "Verified from: %s\n", strings.Join(response.VerifiedFrom, ", "))
	}
	if !quiet {
		for _, step := range response.NextSteps {
			fmt.Fprintf(&b, "Next step: %s\n", step)
		}
		for _, warning := range response.Warnings {
			fmt.Fprintf(&b, "Warning: %s\n", warning)
		}
	}
	return b.String(), nil
}

// RenderEnvironment formats an Environment for terminal output.
func RenderEnvironment(env models.Environment, asJSON bool) (string, error) {
	if asJSON {
		data, err := json.MarshalIndent(env, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data) + "\n", nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "OS: %s\n", env.PrettyName)
	fmt.Fprintf(&b, "Distro: %s\n", env.Distro)
	fmt.Fprintf(&b, "Package manager: %s\n", env.PackageManager)
	fmt.Fprintf(&b, "Shell: %s\n", env.Shell)
	if env.KubeConfig != "" {
		fmt.Fprintf(&b, "KubeConfig: %s\n", env.KubeConfig)
	}
	if env.KubeContext != "" {
		fmt.Fprintf(&b, "KubeContext: %s\n", env.KubeContext)
	}
	for _, name := range []string{
		"dnf", "systemctl", "ss", "git", "find", "ssh", "scp", "rsync", "ssh-keyscan", "nc",
		"docker", "podman", "containerd", "ctr", "crictl", "nerdctl",
		"kubectl", "helm", "terraform", "ansible", "ansible-playbook",
		"aws", "az", "gcloud", "argocd",
		"getenforce", "sestatus", "firewall-cmd", "openssl",
		"lscpu", "free", "lsblk", "dmidecode", "smartctl", "ipmitool", "nvidia-smi",
		"psql", "mysql", "redis-cli",
	} {
		cmd := env.Commands[name]
		state := "missing"
		if cmd.Exists {
			state = cmd.Path
		}
		fmt.Fprintf(&b, "%s: %s\n", name, state)
	}
	return b.String(), nil
}

