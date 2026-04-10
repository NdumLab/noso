package registry

import (
	"fmt"
	"strings"

	"github.com/NdumLab/noso/internal/evidence"
	"github.com/NdumLab/noso/internal/safety"
	"github.com/NdumLab/noso/pkg/models"
)

func selinuxModeIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("getenforce")
	command := "getenforce"
	response := models.Response{
		IntentID:       "inspect_selinux_mode",
		Command:        command,
		Explanation:    "Shows the current SELinux enforcement mode without changing policy state.",
		ExpectedOutput: "One line showing Enforcing, Permissive, or Disabled.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "getenforce")
	return response, nil
}

func selinuxStatusIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("sestatus")
	command := "sestatus"
	response := models.Response{
		IntentID:       "inspect_selinux_status",
		Command:        command,
		Explanation:    "Shows SELinux status, loaded policy details, enforcement mode, and configuration state.",
		ExpectedOutput: "A multi-line SELinux status report including mode, policy name, and config file state.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "sestatus")
	return response, nil
}

func firewallRulesIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("firewall-cmd")
	command := "firewall-cmd --list-all"
	response := models.Response{
		IntentID:       "inspect_firewalld_rules",
		Command:        command,
		Explanation:    "Shows the active firewalld zone configuration, including services, ports, rich rules, and interfaces.",
		ExpectedOutput: "A firewalld zone report showing enabled services, ports, interfaces, and related policy settings.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "firewall-cmd")
	return response, nil
}

func firewallZonesIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("firewall-cmd")
	command := "firewall-cmd --get-active-zones"
	response := models.Response{
		IntentID:       "inspect_firewalld_zones",
		Command:        command,
		Explanation:    "Lists active firewalld zones and the interfaces or sources assigned to them.",
		ExpectedOutput: "Zone names followed by bound interfaces or sources for each active zone.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "firewall-cmd")
	return response, nil
}

func opensslCertIntent(query string, collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("openssl")
	path := extractCertificatePath(query)
	command := fmt.Sprintf("openssl x509 -in %s -noout -text", path)
	response := models.Response{
		IntentID:       "inspect_certificate",
		Command:        command,
		Explanation:    fmt.Sprintf("Decodes certificate %s so you can inspect subject, issuer, validity dates, SANs, and key usage without modifying it.", path),
		ExpectedOutput: "A decoded certificate report with subject, issuer, validity window, public key info, extensions, and SAN entries.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "openssl")
	return response, nil
}

func extractCertificatePath(query string) string {
	lower := strings.ToLower(query)
	for _, marker := range []string{"certificate ", "cert "} {
		if idx := strings.Index(lower, marker); idx >= 0 {
			fields := strings.Fields(query[idx+len(marker):])
			if len(fields) > 0 {
				return strings.Trim(fields[0], "`\"'")
			}
		}
	}
	return "server.crt"
}
