package troubleshoot

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/NdumLab/noso/pkg/models"
)

var (
	dnsProbePattern    = regexp.MustCompile("Run `dig \\+short ([^` ]+)` or `nslookup ([^` ]+)`")
	socketProbePattern = regexp.MustCompile("Run `nc -vz ([^` ]+) ([0-9]{1,5})`")
)

func SpecializeInfrastructureProbes(response models.Response, env models.Environment, thread StateThread) models.Response {
	response.NextSteps = specializeSteps(response.NextSteps, env, thread)
	return response
}

func specializeSteps(steps []string, env models.Environment, thread StateThread) []string {
	if len(steps) == 0 {
		return nil
	}
	out := make([]string, 0, len(steps))
	for _, step := range steps {
		updated := step
		if matches := dnsProbePattern.FindStringSubmatch(updated); len(matches) == 3 && matches[1] == matches[2] {
			host := matches[1]
			switch command := preferredDNSProbe(env, thread, host); {
			case command != "":
				updated = strings.Replace(updated, matches[0], "Run `"+command+"`", 1)
			}
		}
		if matches := socketProbePattern.FindStringSubmatch(updated); len(matches) == 3 {
			host := matches[1]
			port := matches[2]
			switch command := preferredSocketProbe(env, thread, host, port); {
			case command != "":
				updated = strings.Replace(updated, matches[0], "Run `"+command+"`", 1)
			}
		}
		out = append(out, updated)
	}
	return out
}

func hasCommand(env models.Environment, name string) bool {
	info, ok := env.Commands[name]
	return ok && info.Exists
}

func shellSupportsTCPProbe(env models.Environment) bool {
	shell := strings.ToLower(strings.TrimSpace(env.Shell))
	return strings.Contains(shell, "bash")
}

func preferredDNSProbe(env models.Environment, thread StateThread, host string) string {
	if command := kubernetesExecProbe(thread, "nslookup "+host); command != "" {
		return command
	}
	switch {
	case hasCommand(env, "dig"):
		return "dig +short " + host
	case hasCommand(env, "nslookup"):
		return "nslookup " + host
	default:
		return ""
	}
}

func preferredSocketProbe(env models.Environment, thread StateThread, host, port string) string {
	if command := kubernetesExecProbe(thread, fmt.Sprintf("sh -lc 'nc -vz %s %s || </dev/tcp/%s/%s'", host, port, host, port)); command != "" {
		return command
	}
	switch {
	case hasCommand(env, "nc"):
		return fmt.Sprintf("nc -vz %s %s", host, port)
	case shellSupportsTCPProbe(env):
		return fmt.Sprintf("timeout 3 bash -lc '</dev/tcp/%s/%s'", host, port)
	default:
		return fmt.Sprintf("ss -ltn '( sport = :%s )' near %s or from the expected upstream path if the listener should be local", port, host)
	}
}

func kubernetesExecProbe(thread StateThread, inner string) string {
	if thread.ActiveFamily != "kubernetes" || thread.ActiveTarget == "" {
		return ""
	}
	containerArg := ""
	if thread.ActiveContainer != "" {
		containerArg = " -c " + thread.ActiveContainer
	}
	if thread.ActiveNamespace != "" {
		return fmt.Sprintf("kubectl exec -n %s %s%s -- %s", thread.ActiveNamespace, thread.ActiveTarget, containerArg, inner)
	}
	return fmt.Sprintf("kubectl exec %s%s -- %s", thread.ActiveTarget, containerArg, inner)
}
