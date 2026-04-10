package doctor

import (
	"fmt"
	"strings"

	"github.com/NdumLab/noso/internal/config"
	"github.com/NdumLab/noso/internal/safety"
	"github.com/NdumLab/noso/pkg/models"
)

var coreCommands = []string{"bash", "systemctl", "ss", "find", "git"}

func Check(cfg config.Config, env models.Environment) models.Response {
	response := models.Response{
		IntentID:       "doctor",
		Command:        "doctor",
		ExpectedOutput: "A summary of local readiness, config health, platform fit, and command availability.",
		Risk:           safety.RiskLow,
		Confidence:     "High",
		VerifiedFrom:   []string{"/etc/os-release", "command -v", "config"},
	}

	var warnings []string
	switch env.Distro {
	case "rhel", "fedora", "debian", "suse", "arch":
		// Known and supported distro families — no warning needed.
	case "":
		// Distro field not yet populated (old environment struct); fall back
		// to the legacy IsRHEL9 check so existing tests and deployments
		// continue to work.
		if !env.IsRHEL9 {
			warnings = append(warnings, fmt.Sprintf("host is %s %s; package-manager commands may not match this distribution", env.OSID, env.VersionID))
		}
	default:
		warnings = append(warnings, fmt.Sprintf("host distro family %q is not yet fully validated; some package-manager commands may differ", env.Distro))
	}
	if cfg.AuditLogPath == "" {
		warnings = append(warnings, "audit log path is empty")
	} else if !config.AuditPathUsable(cfg.AuditLogPath) {
		warnings = append(warnings, "audit log path is not writable: "+cfg.AuditLogPath)
	}

	var missingCore []string
	for _, name := range coreCommands {
		if !env.Commands[name].Exists {
			missingCore = append(missingCore, name)
		}
	}
	if len(missingCore) > 0 {
		warnings = append(warnings, "core commands missing: "+strings.Join(missingCore, ", "))
	}

	response.Warnings = warnings
	response.Explanation = summary(env, cfg, warnings)
	response.NextSteps = nextSteps(cfg, warnings)
	return response
}

func summary(env models.Environment, cfg config.Config, warnings []string) string {
	base := "Local readiness checks completed."
	switch {
	case env.Distro != "" && env.PackageManager != "":
		base += fmt.Sprintf(" Detected distro family: %s (package manager: %s).", env.Distro, env.PackageManager)
	case env.IsRHEL9:
		base += " Host matches the RHEL 9 target."
	default:
		base += fmt.Sprintf(" Host OS: %s %s.", env.OSID, env.VersionID)
	}
	if cfg.AuditLogPath != "" {
		base += " Audit logging is configured."
	}
	if len(warnings) == 0 {
		return base + " No blocking issues were detected."
	}
	return fmt.Sprintf("%s %d issue(s) need attention before full coverage is available.", base, len(warnings))
}

func nextSteps(cfg config.Config, warnings []string) []string {
	var steps []string
	for _, warning := range warnings {
		switch {
		case strings.Contains(warning, "not yet fully validated"),
			strings.Contains(warning, "may not match this distribution"):
			steps = append(steps, "Run `cli-helper env` to confirm which package-manager commands are available on this host.")
		case strings.Contains(warning, "audit log path"):
			steps = append(steps, "Set `NOSO_AUDIT_LOG_PATH` to a writable path if you want local audit logging.")
		case strings.Contains(warning, "core commands missing"):
			steps = append(steps, "Install the missing core commands to restore the corresponding intent coverage.")
		}
	}
	if len(steps) == 0 && cfg.AuditLogPath != "" {
		steps = append(steps, "Run `cli-helper env` to inspect optional tool coverage for extended domains.")
	}
	return unique(steps)
}

func unique(values []string) []string {
	var out []string
	for _, value := range values {
		seen := false
		for _, existing := range out {
			if existing == value {
				seen = true
				break
			}
		}
		if !seen {
			out = append(out, value)
		}
	}
	return out
}
