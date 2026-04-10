package registry

import (
	"fmt"
	"strings"

	"github.com/noso-dev/noso/internal/evidence"
	"github.com/noso-dev/noso/internal/safety"
	"github.com/noso-dev/noso/pkg/models"
)

func sshVersionIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("ssh")
	command := "ssh -V"
	response := models.Response{
		IntentID:       "inspect_ssh_version",
		Command:        command,
		Explanation:    "Shows the installed OpenSSH client version.",
		ExpectedOutput: "OpenSSH client version information, typically printed on stderr.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "ssh")
	return response, nil
}

func sshConfigIntent(query string, collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("ssh")
	host := extractRemoteHost(query)
	command := fmt.Sprintf("ssh -G %s", host)
	response := models.Response{
		IntentID:       "inspect_ssh_config",
		Command:        command,
		Explanation:    fmt.Sprintf("Shows the effective SSH client configuration for host %s without opening a session.", host),
		ExpectedOutput: "Resolved SSH options such as hostname, port, user, identity file, and host key settings.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "ssh")
	return response, nil
}

func sshHostKeyIntent(query string, collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("ssh-keyscan")
	host := extractRemoteHost(query)
	command := fmt.Sprintf("ssh-keyscan -T 5 %s", host)
	response := models.Response{
		IntentID:       "inspect_ssh_host_key",
		Command:        command,
		Explanation:    fmt.Sprintf("Fetches the presented SSH host key for %s so you can review or pin it before connecting.", host),
		ExpectedOutput: "One or more known_hosts-style lines containing the host key algorithms and public keys offered by the remote host.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "ssh-keyscan")
	return response, nil
}

func sshPortCheckIntent(query string, collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("nc")
	host := extractRemoteHost(query)
	command := fmt.Sprintf("nc -zv %s 22", host)
	response := models.Response{
		IntentID:       "inspect_ssh_port_reachability",
		Command:        command,
		Explanation:    fmt.Sprintf("Checks whether TCP port 22 on %s is reachable before troubleshooting SSH auth or key issues.", host),
		ExpectedOutput: "A success or connection-refused style message showing whether the SSH port is reachable from this host.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "nc")
	return response, nil
}

func rsyncDryRunIntent(query string, collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("rsync")
	source := extractTransferSource(query)
	target := extractTransferTarget(query)
	command := fmt.Sprintf("rsync -avhn %s %s", source, target)
	response := models.Response{
		IntentID:       "preview_rsync_transfer",
		Command:        command,
		Explanation:    "Runs rsync in archive, verbose, human-readable, and dry-run mode so you can preview which files would transfer.",
		ExpectedOutput: "A dry-run file list plus transfer summary without copying any files.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "rsync")
	return response, nil
}

func scpPreviewIntent(query string, collector evidence.Collector) (models.Response, error) {
	source := extractTransferSource(query)
	target := extractTransferTarget(query)
	response, err := rsyncDryRunIntent("rsync "+source+" "+target, collector)
	if err != nil {
		return models.Response{}, err
	}
	response.IntentID = "preview_scp_transfer"
	response.Explanation = "SCP does not provide a true dry-run mode, so this uses an equivalent rsync dry-run preview before performing an SSH file copy."
	response.Command = fmt.Sprintf("rsync -avhn %s %s", source, target)
	response.ExpectedOutput = "A preview of which files would copy over SSH, without actually transferring them."
	return response, nil
}

func extractRemoteHost(query string) string {
	lower := strings.ToLower(query)
	for _, marker := range []string{"for host ", "to "} {
		if idx := strings.Index(lower, marker); idx >= 0 {
			fields := strings.Fields(query[idx+len(marker):])
			if len(fields) > 0 {
				return strings.Trim(fields[0], "`\"'")
			}
		}
	}
	for _, marker := range []string{"host ", "ssh ", "remote "} {
		if idx := strings.Index(lower, marker); idx >= 0 {
			fields := strings.Fields(query[idx+len(marker):])
			if len(fields) > 0 {
				return strings.Trim(fields[0], "`\"'")
			}
		}
	}
	return "remote-host"
}

func extractTransferSource(query string) string {
	lower := strings.ToLower(query)
	if idx := strings.Index(lower, "copy "); idx >= 0 {
		fields := strings.Fields(query[idx+len("copy "):])
		if len(fields) > 0 {
			return strings.Trim(fields[0], "`\"'")
		}
	}
	return "source/"
}

func extractTransferTarget(query string) string {
	lower := strings.ToLower(query)
	if idx := strings.Index(lower, " to "); idx >= 0 {
		fields := strings.Fields(query[idx+len(" to "):])
		if len(fields) > 0 {
			return strings.Trim(fields[0], "`\"'")
		}
	}
	if idx := strings.Index(lower, " into "); idx >= 0 {
		fields := strings.Fields(query[idx+len(" into "):])
		if len(fields) > 0 {
			return strings.Trim(fields[0], "`\"'")
		}
	}
	return "user@remote:/path/"
}
